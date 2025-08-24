package kvraft

import (
	"bytes"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"6.5840/kvraft1/rsm"
	"6.5840/kvsrv1/rpc"
	"6.5840/labgob"
	"6.5840/labrpc"
	"6.5840/tester1"
)

type KVServer struct {
	me   int
	dead int32 // set by Kill()
	rsm  *rsm.RSM

	// Your definitions here.
	mu    sync.Mutex // 其实不太需要这个锁，因为server层面的操作可在rsm层面去做控制
	kvMap map[string]KvValue
}

type KvValue struct {
	Value   string
	Version rpc.Tversion
}

// kvmap的编码包装
type KVWrapper struct {
	Keys   []string
	Values []KvValue
}

// To type-cast req to the right type, take a look at Go's type switches or type
// assertions below:
//
// https://go.dev/tour/methods/16
// https://go.dev/tour/methods/15
func (kv *KVServer) DoOp(req any) any {
	// Your code here
	// 反射出args的类型，然后决定进行什么操作，方式参考rsm server
	kv.mu.Lock()
	defer kv.mu.Unlock()
	rsm.DPrintf("Server doOp server:%v reqtyep:%v req:%v kvmap:%v meaddress:%v", kv.me, reflect.TypeOf(req), req, kv.kvMap, &kv.me)
	switch req.(type) {
	case rpc.GetArgs:
		return kv.getOpHandler(req.(rpc.GetArgs))
	case rpc.PutArgs:
		return kv.putOpHandler(req.(rpc.PutArgs))
	default:
		// wrong type! expecting an Inc.
		rsm.DPrintf("Sever doOp Fatal Err req is not the type of put or get")
	}
	return nil
}

func (kv *KVServer) getOpHandler(req rpc.GetArgs) any {
	reply := rpc.GetReply{}
	if val, ok := kv.kvMap[req.Key]; ok {
		copyVal := val
		reply.Value, reply.Version, reply.Err = copyVal.Value, copyVal.Version, rpc.OK
		rsm.DPrintf("Server getOp server:%v kvmap:%v", kv.me, kv.kvMap)
	} else {
		reply.Err = rpc.ErrNoKey
	}
	return reply
}

func (kv *KVServer) putOpHandler(req rpc.PutArgs) any {
	rep := rpc.PutReply{}

	if val, ok := kv.kvMap[req.Key]; ok && val.Version == req.Version {
		newVal := KvValue{Value: req.Value, Version: val.Version + 1}
		kv.kvMap[req.Key] = newVal
		rep.Err = rpc.OK
		rsm.DPrintf("Server putOp server:%v kvmap:%v", kv.me, kv.kvMap)
	} else if !ok && req.Version == 0 {
		// 每次进行赋值操作，直接用一个全新的
		val = KvValue{Value: req.Value, Version: 1}
		kv.kvMap[req.Key] = val
		rep.Err = rpc.OK
	} else if ok && val.Version != req.Version {
		rep.Err = rpc.ErrVersion
	} else if !ok && req.Version != 0 {
		rep.Err = rpc.ErrNoKey
		rsm.DPrintf("Server putOpHandler what this scenario mean")
	}

	return rep
}

func (kv *KVServer) Snapshot() []byte {
	// Your code here
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	wrapper := KVWrapper{
		Keys:   make([]string, 0, len(kv.kvMap)),
		Values: make([]KvValue, 0, len(kv.kvMap)),
	}
	for k, v := range kv.kvMap {
		wrapper.Keys = append(wrapper.Keys, k)
		wrapper.Values = append(wrapper.Values, v)
	}
	e.Encode(wrapper)
	rsm.DPrintf("Server snapshot server:%v bytes:%v kvmap:%v", kv.me, len(w.Bytes()), kv.kvMap)
	// kv.Restore(w.Bytes())
	return w.Bytes()
}

func (kv *KVServer) Restore(data []byte) {
	// Your code here
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var wrapper KVWrapper
	if d.Decode(&wrapper) != nil {
		rsm.DPrintf("Server Restore Err server:%v", kv.me)
	}

	kvMap := make(map[string]KvValue, len(wrapper.Keys))
	for i := 0; i < len(wrapper.Keys); i++ {
		kvMap[wrapper.Keys[i]] = wrapper.Values[i]
	}

	kv.kvMap = kvMap
	rsm.DPrintf("Server Restore server:%v kvMap:%v meaddress:%v", kv.me, kv.kvMap, &kv.me)
}

func (kv *KVServer) Get(args *rpc.GetArgs, reply *rpc.GetReply) {
	// Your code here. Use kv.rsm.Submit() to submit args
	// You can use go's type casts to turn the any return value
	// of Submit() into a GetReply: rep.(rpc.GetReply)
	err, res := kv.rsm.Submit(*args)
	if err == rpc.ErrWrongLeader {
		reply.Err = err
		return
	}
	rsm.DPrintf("Server Get submit after server:%v", kv.me)
	rep := res.(rpc.GetReply)
	reply.Value, reply.Version, reply.Err = rep.Value, rep.Version, rep.Err
}

func (kv *KVServer) Put(args *rpc.PutArgs, reply *rpc.PutReply) {
	// Your code here. Use kv.rsm.Submit() to submit args
	// You can use go's type casts to turn the any return value
	// of Submit() into a PutReply: rep.(rpc.PutReply)
	err, res := kv.rsm.Submit(*args)
	if err == rpc.ErrWrongLeader {
		reply.Err = err
		return
	}
	rsm.DPrintf("Server Put submit after server:%v", kv.me)
	rep := res.(rpc.PutReply)
	reply.Err = rep.Err
}

// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	// Your code here, if desired.
	rsm.DPrintf("")
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

func (kv *KVServer) listener() {
	for !kv.killed() {
		kv.mu.Lock()
		rsm.DPrintf("Server listener server:%v kvmap:%v", kv.me, kv.kvMap)
		time.Sleep(100 * time.Millisecond)
		kv.mu.Unlock()
	}
}

// StartKVServer() and MakeRSM() must return quickly, so they should
// start goroutines for any long-running work.
func StartKVServer(servers []*labrpc.ClientEnd, gid tester.Tgid, me int, persister *tester.Persister, maxraftstate int) []tester.IService {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(rsm.Op{})
	labgob.Register(rpc.PutArgs{})
	labgob.Register(rpc.GetArgs{})

	kv := &KVServer{me: me}

	kv.rsm = rsm.MakeRSM(servers, me, persister, maxraftstate, kv)
	// You may need initialization code here.
	kv.kvMap = map[string]KvValue{}
	go kv.listener()
	return []tester.IService{kv, kv.rsm.Raft()}
}
