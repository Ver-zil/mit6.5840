package rsm

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"6.5840/kvsrv1/rpc"
	"6.5840/labrpc"
	"6.5840/raft1"
	"6.5840/raftapi"
	"6.5840/tester1"
)

var useRaftStateMachine bool // to plug in another raft besided raft1

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.

	Me         int      // rsm
	OpId       int64    // 操作符id
	Req        any      // msg
	LeaderTerm int      // raft leader term
	ReplyChan  chan any // 参考labrpc设计
}

// A server (i.e., ../server.go) that wants to replicate itself calls
// MakeRSM and must implement the StateMachine interface.  This
// interface allows the rsm package to interact with the server for
// server-specific operations: the server must implement DoOp to
// execute an operation (e.g., a Get or Put request), and
// Snapshot/Restore to snapshot and restore the server's state.
type StateMachine interface {
	DoOp(any) any
	Snapshot() []byte
	Restore([]byte)
}

type RSM struct {
	mu           sync.Mutex
	me           int
	rf           raftapi.Raft
	applyCh      chan raftapi.ApplyMsg
	maxraftstate int // snapshot if log grows this big
	sm           StateMachine

	// Your definitions here.
	opId int64 // 给msg分配的id
}

// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
//
// me is the index of the current server in servers[].
//
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// The RSM should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
//
// MakeRSM() must return quickly, so it should start goroutines for
// any long-running work.
func MakeRSM(servers []*labrpc.ClientEnd, me int, persister *tester.Persister, maxraftstate int, sm StateMachine) *RSM {
	rsm := &RSM{
		me:           me,
		maxraftstate: maxraftstate,
		applyCh:      make(chan raftapi.ApplyMsg),
		sm:           sm,
	}
	if !useRaftStateMachine {
		rsm.rf = raft.Make(servers, me, persister, rsm.applyCh)
	}
	// 开启reader go routine
	go rsm.applyMsgReader()
	return rsm
}

// 持续读取apply，设计参考
func (rsm *RSM) applyMsgReader() {
	for msg := range rsm.applyCh {
		if msg.SnapshotValid {
			DPrintf("RSM applier Snapshot deal it after")
		} else if msg.CommandValid {
			op, ok := msg.Command.(Op)
			if !ok {
				DPrintf("RSM Fatal Command is not op, may it's a bug cmd:%v type:%v", msg.Command, reflect.TypeOf(msg.Command))
			}
			DPrintf("RSM applier me:%v reqtype:%v req:%v", rsm.me, reflect.TypeOf(op.Req), op.Req)
			rep := rsm.sm.DoOp(op.Req)
			DPrintf("RSM applier me:%v cmd:%v reptype:%v rep:%v", rsm.me, msg.Command, reflect.TypeOf(rep), rep)

			// 防止其他follower调用chan
			// 需要让submit的节点能收到反馈，也需要让重启的节点正常运行
			curTerm, _ := rsm.rf.GetState()
			// note:这里没放isLeader是因为state并不作为状态存储，所以重启后就是false
			if op.Me == rsm.me && op.LeaderTerm == curTerm {
				DPrintf("RSM applier DoOp and submit response pre server:%v", rsm.me)
				op.ReplyChan <- rep
				DPrintf("RSM applier DoOp and submit response after server:%v", rsm.me)
			} else {

			}

		} else {

		}
	}
}

func (rsm *RSM) Raft() raftapi.Raft {
	return rsm.rf
}

// Submit a command to Raft, and wait for it to be committed.  It
// should return ErrWrongLeader if client should find new leader and
// try again.
func (rsm *RSM) Submit(req any) (rpc.Err, any) {

	// Submit creates an Op structure to run a command through Raft;
	// for example: op := Op{Me: rsm.me, Id: id, Req: req}, where req
	// is the argument to Submit and id is a unique id for the op.

	// your code here
	// 将req封装成op
	DPrintf("RSM Submit Call")
	if startTerm, isLeader := rsm.rf.GetState(); isLeader {
		op := Op{
			Me:         rsm.me,
			OpId:       atomic.AddInt64(&rsm.opId, 1),
			Req:        req,
			ReplyChan:  make(chan any),
			LeaderTerm: startTerm,
		}
		DPrintf("RSM Submit------------start req:%v", req)
		_, curTerm, isLeader := rsm.rf.Start(op)
		DPrintf("RSM submit------------- startTerm:%v curTerm:%v server:%v isLeader:%v", startTerm, curTerm, rsm.me, isLeader)
		for isLeader && curTerm == startTerm {
			DPrintf("RSM leader Submit op:%v", op)

			select {
			case rep := <-op.ReplyChan:
				return rpc.OK, rep
			case <-time.After(time.Second):
				DPrintf("RSM Submit Overtime op:%v", op)
			}

			curTerm, isLeader = rsm.rf.GetState()
			DPrintf("RSM submit overtime state")
		}
	}
	DPrintf("RSM Submit ErrWrong")
	return rpc.ErrWrongLeader, nil // i'm dead, try another server.
}
