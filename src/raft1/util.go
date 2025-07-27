package raft

import (
	"fmt"
	"log"
	"time"

	tester "6.5840/tester1"
)

// Debugging
const Debug = false

func DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(format, a...)
	}
}

func Record(tag, desp, details string) {
	// 这个函数将DPrint和tester.Annotation集成
	tester.Annotate(tag, desp, details)
	DPrintf(details)
}

const (
	NodeTag      string = "节点%v"
	ElectionTag  string = "选举"
	LogCommitTag string = "日志提交"
)

const (
	NodeElectionVoteDesc         string = "ele:%v"
	NodeLogReplicateDesc         string = "repIdx:%v"
	NodeLogReplicateConflictDesc string = "冲突"

	ElectionStartDesc string = "%v开选举"
	ElectionResDesc   string = "%v选举成功"

	LogReplicateDesc string = "log过半复制idx:%v"
	LogCommitDesc    string = "log提交到idx:%v"
	AEConflictDesc   string = "AE Conf"
	AEAccessDesc     string = "AE Acc"
)

const (
	NodeElectionVoteDetail string = "当前节点term:%v 请求节点%v term:%v"
	NodeLogReplicateDetail string = ""

	ElectionStartDetail string = "节点%v开始进行选举 当前term:%v"
)

type ExecutionTimer struct {
	start time.Time
	// messages []string
}

// 创建计时器（自动开始计时）
func NewTimer() *ExecutionTimer {
	return &ExecutionTimer{
		start: time.Now(),
	}
}

func (t *ExecutionTimer) LogPhase(format string, a ...interface{}) {
	if Debug {
		elapsed := time.Since(t.start)
		log.Printf("msg:%v time:%v", fmt.Sprintf(format, a...), elapsed)
		// t.messages = append(t.messages, fmt.Sprintf("[%s] %v", phase, elapsed))
	}
}
