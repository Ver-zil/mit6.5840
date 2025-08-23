package kvraft

import (
	// "log"

	"time"

	"6.5840/kvsrv1/rpc"
	"6.5840/kvtest1"
	"6.5840/tester1"
)

type Clerk struct {
	clnt    *tester.Clnt
	servers []string
	// You will have to modify this struct.
	lastLeaderIdx int
}

func MakeClerk(clnt *tester.Clnt, servers []string) kvtest.IKVClerk {
	ck := &Clerk{clnt: clnt, servers: servers}
	// You'll have to add code here.
	return ck
}

// Get fetches the current value and version for a key.  It returns
// ErrNoKey if the key does not exist. It keeps trying forever in the
// face of all other errors.
//
// You can send an RPC to server i with code like this:
// ok := ck.clnt.Call(ck.servers[i], "KVServer.Get", &args, &reply)
//
// The types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. Additionally, reply must be passed as a pointer.
func (ck *Clerk) Get(key string) (string, rpc.Tversion, rpc.Err) {

	// You will have to modify this function.

	// get操作只能返回ok，但是如果no key也应该返回，判错就是框架bug
	args := rpc.GetArgs{Key: key}
	reply := rpc.GetReply{}
	leaderIdx := ck.lastLeaderIdx

	for {
		ok := ck.clnt.Call(ck.servers[leaderIdx], "KVServer.Get", &args, &reply)
		if ok && reply.Err != rpc.ErrWrongLeader {
			break
		}

		time.Sleep(100 * time.Millisecond)

		leaderIdx = (leaderIdx + 1) % len(ck.servers)
	}
	ck.lastLeaderIdx = leaderIdx

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
// but the response was lost, and the the Clerk doesn't know if
// the Put was performed or not.
//
// You can send an RPC to server i with code like this:
// ok := ck.clnt.Call(ck.servers[i], "KVServer.Put", &args, &reply)
//
// The types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. Additionally, reply must be passed as a pointer.
func (ck *Clerk) Put(key string, value string, version rpc.Tversion) rpc.Err {
	// You will have to modify this function.
	args := rpc.PutArgs{Key: key, Value: value, Version: version}
	reply := rpc.PutReply{}

	leaderIdx := ck.lastLeaderIdx
	isRirst := true

	// 只有err==ok || err==errMaybe才能进行返回
	for {
		ok := ck.clnt.Call(ck.servers[leaderIdx], "KVServer.Put", &args, &reply)
		if ok && reply.Err != rpc.ErrWrongLeader {
			break
		}

		time.Sleep(100 * time.Millisecond)

		// note: 这里的难点在于如何定义非第一次
		// note: 为了兼容reliable网络，isFirst&&Err==ErrWrongLeader的情况可以不判定非第一次
		// if !isRirst || reply.Err != rpc.ErrWrongLeader {
		// 	isRirst = false
		// }
		isRirst = false

		// if reply.Err == rpc.ErrWrongLeader {
		// 	isRirst = true
		// }
		leaderIdx = (leaderIdx + 1) % len(ck.servers)
	}
	ck.lastLeaderIdx = leaderIdx

	// 网络波动，多次操作中间可能成功一次但是没办法判定到底是不是真的自己操作的
	if !isRirst && reply.Err == rpc.ErrVersion {
		return rpc.ErrMaybe
	}

	return reply.Err
}
