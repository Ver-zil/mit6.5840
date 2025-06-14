package kvsrv

import (
	"log"
	"sync"

	"6.5840/kvsrv1/rpc"
	"6.5840/labrpc"
	"6.5840/tester1"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type kvValue struct {
	value   string
	version rpc.Tversion // 版本号
}

type KVServer struct {
	mu sync.Mutex

	// Your definitions here.
	data map[string]*kvValue // 因为每次put的时候需要顺便比较版本号，所以上锁是不可避免的，所以不用sync.Map
}

func MakeKVServer() *KVServer {
	kv := &KVServer{data: make(map[string]*kvValue, 0)}
	// Your code here.

	// 可能需要自己注册rpc
	return kv
}

// Get returns the value and version for args.Key, if args.Key
// exists. Otherwise, Get returns ErrNoKey.
func (kv *KVServer) Get(args *rpc.GetArgs, reply *rpc.GetReply) {
	// Your code here.
	// 读操作没必要上锁，因为没有删除操作
	// todo:判断key存在与否和*val这个事情不是原子的，如果有remove操作就需要上锁了
	if val, ok := kv.data[args.Key]; ok {
		// 避免高并发环境下value和version最后读到了不同版本的值，采用变量副本来赋值
		realVal := *val
		reply.Value = realVal.value
		reply.Version = realVal.version
		reply.Err = rpc.OK
	} else {
		reply.Err = rpc.ErrNoKey
	}
}

// Update the value for a key if args.Version matches the version of
// the key on the server. If versions don't match, return ErrVersion.
// If the key doesn't exist, Put installs the value if the
// args.Version is 0, and returns ErrNoKey otherwise.
func (kv *KVServer) Put(args *rpc.PutArgs, reply *rpc.PutReply) {
	// Your code here.
	// todo:锁的粒度比较粗，高并发环境下可以考虑根据key进行加锁
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// 如果args的版本号和key对应的版本号匹配则正常赋值和version+1
	// 如果not ok但是args.version=0可以直接添加键值
	// 不匹配返回rpc.ErrVersion
	// key不存在且版本号有问题返回rpc.ErrNoKey
	if val, ok := kv.data[args.Key]; ok && val.version == args.Version {
		val.value = args.Value
		val.version = val.version + 1
		reply.Err = rpc.OK
	} else if !ok && args.Version == 0 {
		val = &kvValue{value: args.Value, version: 1}
		kv.data[args.Key] = val
		reply.Err = rpc.OK
	} else if ok && val.version != args.Version {
		reply.Err = rpc.ErrVersion
	} else if !ok && args.Version != 0 {
		reply.Err = rpc.ErrNoKey
	}
}

// You can ignore Kill() for this lab
func (kv *KVServer) Kill() {
}

// You can ignore all arguments; they are for replicated KVservers
func StartKVServer(ends []*labrpc.ClientEnd, gid tester.Tgid, srv int, persister *tester.Persister) []tester.IService {
	kv := MakeKVServer()
	return []tester.IService{kv}
}
