package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

const (
	TaskOverDue       string = "任务超时,已经重新入队"
	TaskSubmitSuccess string = "任务提交成功, congratulations"
)

// task只保存和task相关的信息
// 比如 任务类型、任务路径
type Task struct {
	TaskType   string   // 标记当前任务是什么类型 map,reduce,end,notasks
	FilePath   []string // 记录当前任务的文件路径
	TaskNumber int      // task的分配号，map有自己的map号，reduce有reduce号
}

type RequestArgs struct {
	Workerid int32 // 规定workerid>0
	Task     Task
}

type ResponseReply struct {
	Workerid int32 // 给每个work分配的id
	NReduce  int   // hash对n求余
	Task     Task  // 将task进行封装
}

// 提交任务用的
type TaskSubmissionArgs struct {
	Workerid int32
	Task     Task
}

type TaskSubmissionReply struct {
	Msg string
}

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	// 中间过程的socket path
	// worker和coordinator沟通都通过该socket进行
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
