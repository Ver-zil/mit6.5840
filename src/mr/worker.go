package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// mapf函数会帮忙将文档里的key提取出来，但是需要你单独处理合并逻辑
	// mapf("ignore this","word1 word2 word1")-> {"word1":1,"word2":1,"word1":1}

	// Your worker implementation here.
	// 和coordinator进行第一次通讯，询问编号
	request := RequestArgs{}
	reply := ResponseReply{}

	isok := call("Coordinator.AssignWorkid", &request, &reply)

	if !isok {
		log.Fatal("connect error")
		return
	}

	workid := reply.Workerid
	// 后续再进行while循环去询问任务
	// map任务执行map逻辑，reduce任务执行reduce逻辑，如果没任务了则直接结束
	for {
		request = RequestArgs{Workerid: workid}
		reply = ResponseReply{}

		isok = call("Coordinator.Assigntasks", &request, &reply)

		if !isok {
			log.Println("connect error")
			return
		}

		// 根据reply的结果判断当前是什么任务
		switch reply.Task.TaskType {
		case Map:
			log.Printf("doMapTask workid: %v, taskNumbber: %v\n", workid, reply.Task.TaskNumber)
			doMapTask(reply, mapf)
		case Reduce:
			log.Printf("doReduceTask workid: %v, taskNumbber: %v\n", workid, reply.Task.TaskNumber)
			doReduceTask(reply, reducef)
		case HeartBeat:
			log.Println("HeartBeat")
		case End:
			log.Println("End")
			return
		}

		// 结束任务以后睡眠1s
		time.Sleep(1 * time.Second)
	}
	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

}

func doMapTask(reply ResponseReply, mapf func(string, string) []KeyValue) {
	rawContent, err := os.ReadFile(reply.Task.FilePath[0])
	if err != nil {
		// 错误处理逻辑
		log.Println("文件处理错误，需要重新处理")
	}

	keyValueList := mapf(reply.Task.FilePath[0], string(rawContent))
	// 执行maplist合并逻辑，将mpa任务的nreduce个分区进行保存
	savePathList := mergeAndSave(keyValueList, reply)
	// 将任务进行submit
	submitTask := TaskSubmissionArgs{Workerid: reply.Workerid, Task: Task{reply.Task.TaskType, savePathList, reply.Task.TaskNumber}}
	submitTaskMsg := TaskSubmissionReply{}
	isOk := call("Coordinator.SubmitMapTaskHandler", &submitTask, &submitTaskMsg)

	// fmt.Println("reply msg %v", submitTaskMsg.msg)
	if !isOk {
		log.Println("connect err，进行额外处理")
	}
}

// 返回保存路径的集合
func mergeAndSave(keyValueList []KeyValue, reply ResponseReply) []string {
	// savePathList是最终的保存路径
	// mapList是长度为nReduce，其中每个元素都是一个map
	nReduce := reply.NReduce
	taskNumber := reply.Task.TaskNumber
	savePathList := make([]string, nReduce)
	mapList := make([]map[string]int, nReduce)
	for i := range mapList {
		mapList[i] = make(map[string]int)
	}

	for _, kv := range keyValueList {
		idx := ihash(kv.Key) % nReduce
		mapList[idx][kv.Key] = mapList[idx][kv.Key] + 1
	}

	// 将文件sava起来，文件名mr-taskNumber-nReduceBlock
	for i := 0; i < nReduce; i++ {
		// todo这里其实有个小问题，如果mrworker和mrcoordinator不在一个dir下，那么路径解析就会出问题
		// todo提交任务的时候应该将相对路径转化成绝对路径
		filename := fmt.Sprintf("/mr-%d-%d.json", taskNumber, i)

		jsondata, err := json.Marshal(mapList[i])

		if err != nil {
			log.Fatalf("jsondata出现问题：wokerid：%d，idx：%d", reply.Workerid, i)
		}

		dirPath := "./mrtmp"
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			log.Fatalf("Error creating directory:%v", err)
		}
		file, err := os.Create(filepath.Join(dirPath, filename))
		if err != nil {
			log.Fatalf("Error creating file:%v", err)
		}
		defer file.Close()

		_, err = file.Write(jsondata)
		if err != nil {
			log.Fatalf("文件写入出错 wokerid：%d，idx：%d", reply.Workerid, i)
		}

		// 这一步保证了提交给coordinator的任务，是按照hash顺序进行提交的
		savePathList[i] = filepath.Join(dirPath, filename)
	}

	return savePathList
}

func doReduceTask(reply ResponseReply, reducef func(string, []string) string) {
	// 本reduce任务中，可能不太需要reducef，因为返回的都是些没用的东西
	// 根据reply中的路径，将读取的map全都汇总
	sum := map[string]int{}

	for _, filePath := range reply.Task.FilePath {
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}

		// 解析 JSON 数据
		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		if err != nil {
			fmt.Println("Error unmarshaling JSON:", err)
			return
		}

		for key, value := range result {
			sum[key] = sum[key] + int(value.(float64))
		}
	}

	// 对sum的结果进行字典序排序，然后进行存储
	keys := []string{}
	for k := range sum {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	filename := fmt.Sprintf("mr-out-%d", reply.Task.TaskNumber)
	ofile, _ := os.Create(filename)
	defer ofile.Close()

	for idx := range keys {
		fmt.Fprintf(ofile, "%v %v\n", keys[idx], sum[keys[idx]])
	}

	// 将最终结果进行提交
	submitTask := TaskSubmissionArgs{Workerid: reply.Workerid, Task: Task{reply.Task.TaskType, []string{filename}, reply.Task.TaskNumber}}
	submitTaskMsg := TaskSubmissionReply{}
	isOk := call("Coordinator.SubmitReduceTaskHandler", &submitTask, &submitTaskMsg)

	log.Printf("reduce task finished and submit workerid:%v, taskNumber:%v, reply msg:%v\n", reply.Workerid, reply.Task.TaskNumber, submitTaskMsg.Msg)

	if !isOk {
		log.Println("connect err，进行额外处理")
	}
}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
