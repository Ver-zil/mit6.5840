package kvsrv

import (
	"log"
	"time"

	"6.5840/kvsrv1/rpc"
	"6.5840/kvtest1"
	"6.5840/tester1"
)

// 对call之类的方法进行封装，有一部分原因是为了方便在测试的时候，模拟随机丢包的问题
type Clerk struct {
	clnt   *tester.Clnt // 其中定义好了rpc通讯的方法call
	server string
}

// kvtest.IKVClerk有get和put方法的通用接口
func MakeClerk(clnt *tester.Clnt, server string) kvtest.IKVClerk {
	ck := &Clerk{clnt: clnt, server: server}
	// You may add code here.
	return ck
}

// Get fetches the current value and version for a key.  It returns
// ErrNoKey if the key does not exist. It keeps trying forever in the
// face of all other errors.
//
// You can send an RPC with code like this:
// ok := ck.clnt.Call(ck.server, "KVServer.Get", &args, &reply)
//
// The types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. Additionally, reply must be passed as a pointer.
func (ck *Clerk) Get(key string) (string, rpc.Tversion, rpc.Err) {
	// You will have to modify this function.
	// 当前存在丢包问题的时候，get操作只需要重复循环就行了，不会有什么问题
	args := rpc.GetArgs{Key: key}
	reply := rpc.GetReply{}
	for !ck.clnt.Call(ck.server, "KVServer.Get", &args, &reply) {

		// 按照描述除了ErrNoKey的其他错误，都需要不停尝试
		if reply.Err == rpc.ErrNoKey {
			return "", 0, rpc.ErrNoKey
		} else if reply.Err == rpc.OK {
			break
		}
	}

	return reply.Value, reply.Version, reply.Err
}

// Put updates key with value only if the version in the
// request matches the version of the key at the server.  If the
// versions numbers don't match, the server should return
// ErrVersion.  If Put receives an ErrVersion on its first RPC, Put
// should return ErrVersion, since the Put was definitely not
// performed at the server. If the server returns ErrVersion on a
// resend RPC, then Put must return ErrMaybe to the application, since
// its earlier RPC might have been processed by the server successfully
// but the response was lost, and the Clerk doesn't know if
// the Put was performed or not.
//
// You can send an RPC with code like this:
// ok := ck.clnt.Call(ck.server, "KVServer.Put", &args, &reply)
//
// The types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. Additionally, reply must be passed as a pointer.
func (ck *Clerk) Put(key, value string, version rpc.Tversion) rpc.Err {
	// You will have to modify this function.
	// 开始进行lab2-3的修改：rpc.ErrMaybe
	// 1. client进行put的时候没有收到reply对应两种情况
	//	第一种是request丢包了，这种就意味着没有执行
	//	第二种是reply丢包了，这种意味着执行了
	//	对于应用来说，并不直到到底执行还是没有执行，所以需要不停重复直到收到回复为止
	//	第一次收到err直接将err返回，第二次还是err的话，直接返回rpc.ErrMaybe
	args := rpc.PutArgs{Key: key, Value: value, Version: version}
	reply := rpc.PutReply{}

	isRirst := true
	for !ck.clnt.Call(ck.server, "KVServer.Put", &args, &reply) {
		log.Printf("client put connect err \n")
		isRirst = false
		time.Sleep(100 * time.Millisecond)
	}

	// 只有多次rpc并且操作失败了，版本号不匹配，才返回rpc.ErrMaybe
	if !isRirst && reply.Err == rpc.ErrVersion {
		return rpc.ErrMaybe
	}

	return reply.Err
}
