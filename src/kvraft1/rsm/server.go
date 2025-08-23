package rsm

import (
	"bytes"
	"log"
	"sync"

	"6.5840/labgob"
	"6.5840/labrpc"
	"6.5840/raftapi"
	"6.5840/tester1"
)

type Inc struct {
}

type IncRep struct {
	N int
}

type Null struct {
}

type NullRep struct {
}

type Dec struct {
}

// rsm test的上层server
type rsmSrv struct {
	ts      *Test
	me      int
	rsm     *RSM
	mu      sync.Mutex
	counter int // doOP inc的次数
}

func makeRsmSrv(ts *Test, srv int, ends []*labrpc.ClientEnd, persister *tester.Persister, snapshot bool) *rsmSrv {
	//log.Printf("mksrv %d", srv)
	// 只有注册进去的才能进行序列化和反序列化(大写)
	labgob.Register(Op{})
	labgob.Register(Inc{})
	labgob.Register(IncRep{})
	labgob.Register(Null{})
	labgob.Register(NullRep{})
	labgob.Register(Dec{})
	s := &rsmSrv{
		ts: ts,
		me: srv,
	}
	s.rsm = MakeRSM(ends, srv, persister, ts.maxraftstate, s)
	return s
}

// 只接收Inc和Null类型，会返回对应的Rep
func (rs *rsmSrv) DoOp(req any) any {
	//log.Printf("%d: DoOp: %T(%v)", rs.me, req, req)
	switch req.(type) {
	case Inc:
		rs.mu.Lock()
		rs.counter += 1
		rs.mu.Unlock()
		return &IncRep{rs.counter}
	case Null:
		return &NullRep{}
	default:
		// wrong type! expecting an Inc.
		log.Fatalf("DoOp should execute only Inc and not %T", req)
	}
	return nil
}

// srv的snapshot其实只是把counter给记录了
func (rs *rsmSrv) Snapshot() []byte {
	//log.Printf("%d: snapshot", rs.me)
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rs.counter)
	DPrintf("RSM Server.go snapshot rs.counter:%v", rs.counter)
	return w.Bytes()
}

func (rs *rsmSrv) Restore(data []byte) {
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	if d.Decode(&rs.counter) != nil {
		log.Fatalf("%v couldn't decode counter", rs.me)
	}
	DPrintf("RSM Server.go restore rs.counter:%v", rs.counter)
	//log.Printf("%d: restore %d", rs.me, rs.counter)
}

// kill的时候，只把自己和rsm的连接断了，但是没有把rsm和srv的连接断了
func (rs *rsmSrv) Kill() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	//log.Printf("kill %d", rs.me)
	//rs.rsm.Kill()
	rs.rsm = nil
}

func (rs *rsmSrv) Raft() raftapi.Raft {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.rsm.Raft()
}
