package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Map       string = "map"
	Reduce    string = "reduce"
	End       string = "end"
	HeartBeat string = ""
)

const (
	// 这个const是给map task state用的
	finished   int = 0
	unfinished int = -1
)

type Coordinator struct {
	// Your definitions here.
	files              []string         // 初始化的时候将所有的file split解析出来
	state              string           // map,reduce,end
	nextWorkId         int32            // 给每个阶段的work分配的id
	mu                 sync.Mutex       // 上锁用的
	nReduce            int              // 相当于最后n个结果块，mod的求余对象
	stageWg            *sync.WaitGroup  // 转阶段用的
	mapWorkTodoList    chan Task        // 内部可进行分配的map task
	mapTaskNumber      map[int]Task     // taskid->task
	mapTaskState       map[int]int      // taskid->workerid
	reduceWorkTodoList chan Task        // 内部所有的存放reduce task所需要的路径
	reduceTaskNumber   map[int][]string // reduce block number -> reduce task file path
	reduceTaskState    map[int]int      // reduce task block number -> state 竞态资源，提交任务和超市检查

}

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

func (c *Coordinator) Assigntasks(args *RequestArgs, reply *ResponseReply) error {
	// 然后再从channel中取出一个任务进行分配

	// 先制作分配任务的逻辑
	reply.Workerid = args.Workerid
	reply.NReduce = c.nReduce
	if c.state == Map {
		// 通过非阻塞式方法分配map的任务
		select {
		case assigntask := <-c.mapWorkTodoList:
			reply.Task = assigntask
			c.mapTaskState[assigntask.TaskNumber] = int(args.Workerid)
			// todo:后续需要处理因为超时之类的事情导致的处理逻辑
			go c.monitor(assigntask)
		default:
			reply.Task.TaskType = HeartBeat
		}
	} else if c.state == Reduce {
		select {
		case assigntask := <-c.reduceWorkTodoList:
			reply.Task = assigntask
			c.reduceTaskState[assigntask.TaskNumber] = int(args.Workerid)
			go c.monitor(assigntask)
		default:
			reply.Task.TaskType = HeartBeat
		}
	}

	return nil
}

func (c *Coordinator) AssignWorkid(args *RequestArgs, reply *ResponseReply) error {
	// 首先看一下workid是否为0，如果是0则分配一个workid
	// 通过原子类保证线程安全
	reply.Workerid = atomic.AddInt32(&c.nextWorkId, 1)

	return nil
}

// 负责处理worker提交map task任务
func (c *Coordinator) SubmitMapTaskHandler(args *TaskSubmissionArgs, reply *TaskSubmissionReply) error {
	// 需要上锁的核心原因是怕和撤销任务这个事情产生冲突
	// c.mapTaskState是竞态资源
	// 将Task重新入队的判定条件就是看当前任务还是workerid在执行
	c.mu.Lock()
	defer c.mu.Unlock()

	// 首先判断一下当前的task是否还是自己在执行，如果是就处理提交逻辑，如果不是应该返回错误
	// todo:理论上这里还有一个else，里面返回一个err
	if args.Task.TaskType == c.state && c.mapTaskState[args.Task.TaskNumber] == int(args.Workerid) {
		// 标记当前任务为已完成
		// todo：map阶段如果考虑到coordinator会崩溃的情况，可能需要做更多的修改
		c.mapTaskState[args.Task.TaskNumber] = int(finished)

		// 再对map任务进行分解成为reduce任务需要的模块
		mapTaskIntermediateFilepath := args.Task.FilePath
		for idx, filepath := range mapTaskIntermediateFilepath {
			if c.reduceTaskNumber[idx] == nil {
				c.reduceTaskNumber[idx] = make([]string, 0)
			}
			c.reduceTaskNumber[idx] = append(c.reduceTaskNumber[idx], filepath)
		}

		reply.Msg = TaskSubmitSuccess

		// 唤醒wg
		c.stageWg.Done()
	} else {
		// 返回一个超时msg
		reply.Msg = TaskOverDue
	}

	return nil
}

// 负责处理worker提交map task任务
func (c *Coordinator) SubmitReduceTaskHandler(args *TaskSubmissionArgs, reply *TaskSubmissionReply) error {
	// 需要上锁的核心原因是怕和撤销任务这个事情产生冲突
	// c.mapTaskState是竞态资源
	// 将Task重新入队的判定条件就是看当前任务还是workerid在执行
	c.mu.Lock()
	defer c.mu.Unlock()

	// 首先判断一下当前的task是否还是自己在执行，如果是就处理提交逻辑，如果不是应该返回错误
	// todo:理论上这里还有一个else，里面返回一个err
	if args.Task.TaskType == c.state && c.reduceTaskState[args.Task.TaskNumber] == int(args.Workerid) {
		// 标记当前任务为已完成
		c.reduceTaskState[args.Task.TaskNumber] = finished

		// todo正经的mr任务应该需要将路径刷入磁盘
		// 唤醒wg
		reply.Msg = TaskSubmitSuccess
		c.stageWg.Done()
	} else {
		reply.Msg = TaskOverDue
	}

	return nil
}

func (c *Coordinator) monitor(task Task) {
	time.Sleep(10 * time.Second)

	c.mu.Lock()
	defer c.mu.Unlock()

	// 首先需要确保当前监控的对象task和当前的coordinator是同一阶段的，如果不在一个阶段，意味着是一个过期的监控
	if task.TaskType != c.state {
		return
	}

	// 判断一下当前任务是否已经完成，如果没有，则重新设置任务状态，并且将任务重新加入队列
	// todo:理论上应该再检查一下是否是自己的workerid
	if c.state == Map && c.mapTaskState[task.TaskNumber] != finished {
		c.mapTaskState[task.TaskNumber] = unfinished
		c.mapWorkTodoList <- task
	} else if c.state == Reduce && c.reduceTaskState[task.TaskNumber] != finished {
		c.reduceTaskState[task.TaskNumber] = unfinished
		c.reduceWorkTodoList <- task
	}
}

// 转阶段判定，转成reduce
func (c *Coordinator) stateTransform() {
	c.stageWg.Wait()
	// 当所有计数完成以后，就可以开始转阶段了
	// 转阶段需要将所有的reduce任务封装起来，然后再将状态设置成reduce
	// 顺序反了可能也没太大问题(除了需要注意stagewg)，因为任务都是通过channel来获取的，本身就是线程安全的
	// time.Sleep(10 * time.Second) // 测试用的
	if c.state == Map {
		c.stageWg = new(sync.WaitGroup)
		c.stageWg.Add(len(c.reduceTaskNumber))
		for key, value := range c.reduceTaskNumber {
			task := Task{TaskType: Reduce, TaskNumber: key, FilePath: value}
			c.reduceWorkTodoList <- task
			c.reduceTaskState[key] = unfinished
		}

		c.state = Reduce

		go c.stateTransform()
	} else if c.state == Reduce {
		// todo清理中间文件
		os.RemoveAll("./mrtmp")
		c.state = End
	}
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.
	if c.state == End {
		ret = true
	}

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
// files是传进来的pg-*.txt，需要进行一定的解析，nRduce指定n个reduce task 相当于mod nReduce
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	// 解析files
	var allFiles []string
	for _, file := range files {
		// 解析通配符
		matches, err := filepath.Glob(file)
		if err != nil {
			log.Println("Error:", err)
			continue
		}
		// 将匹配到的文件路径添加到 allFiles 数组中
		//  ...类似于python里的*[]
		allFiles = append(allFiles, matches...)
	}

	c := Coordinator{files: allFiles, nReduce: nReduce, state: Map, stageWg: new(sync.WaitGroup),
		mapWorkTodoList: make(chan Task, len(allFiles)), mapTaskNumber: make(map[int]Task), mapTaskState: make(map[int]int),
		reduceWorkTodoList: make(chan Task, nReduce), reduceTaskNumber: make(map[int][]string), reduceTaskState: make(map[int]int)}

	// Your code here.

	// 初始化coordinator里的maptask
	for idx, file := range c.files {
		task := Task{TaskType: Map, FilePath: []string{file}, TaskNumber: idx}
		c.mapWorkTodoList <- task
		c.mapTaskNumber[idx] = task
		c.mapTaskState[idx] = unfinished
	}

	// 初始wg
	c.stageWg.Add(len(c.files))

	// 开启转阶段的goroutine
	go c.stateTransform()

	c.server()
	return &c
}
