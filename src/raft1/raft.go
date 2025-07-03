package raft

// The file raftapi/raft.go defines the interface that raft must
// expose to servers (or the tester), but see comments below for each
// of these functions for more details.
//
// Make() creates a new raft peer that implements the raft interface.

import (
	//	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
	"6.5840/raftapi"
	tester "6.5840/tester1"
)

const (
	leader    int = 1
	follower  int = 2
	candidate int = 3
)

const (
	ElectionTimeout  = 500
	HeartBeatTimeout = 125
)

// 随机函数rand.Intn在并发情况下有线程安全问题，所以这里为了保证线程安全制作了这个类
type LockedRand struct {
	mu   sync.Mutex
	rand *rand.Rand
}

func (lr *LockedRand) Intn(timeout int) int {
	lr.mu.Lock()
	defer lr.mu.Unlock()
	return lr.rand.Intn(timeout)
}

// 保证随机数线程安全使用的
var GlobalRand = &LockedRand{
	rand: rand.New(rand.NewSource(time.Now().UnixNano())),
}

func GetRandomElectionTimeout() time.Duration {
	return time.Duration(ElectionTimeout+GlobalRand.Intn(ElectionTimeout)) * time.Millisecond
}

func GetStableHeartBeatTimeout() time.Duration {
	return time.Duration(HeartBeatTimeout) * time.Microsecond
}

type LogEntry struct {
	Term    int         // entry commit的term
	Command interface{} // 指令，任意数据
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.RWMutex        // Lock to protect shared access to this peer's state(go里没有可重入锁的设计)
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *tester.Persister   // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (3A, 3B, 3C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	// persistent state on all servers
	currentTerm int        // 当前的term，理论上term应该和votedFor一起进行更新
	votedFor    int        // 选举阶段投票给了谁，选举结束以后应该将信息抹除
	log         []LogEntry // 用来记录当前log的日志信息的
	// volatile state on all servers
	commitIdx   int // 被过半节点commit的log
	lastApplied int // state machine最后应用log的位置 <=commitIdx
	// volitile state on leader
	// match和next之间的区别主要在于对于随机波动的处理上
	nextIdx  []int // 对于follower i nextIdx[i]是leader需要给其发送日志内容的部分，发送后直接进行一部分（需要做优化的地方）
	matchIdx []int // 对于follower i match[i]是为了提高效率而存在的，nextIdx发送的消息收到确认后再进行更新，悲观确认，可以作为commit的依据

	// 自定义属性
	state          int                   // 当前的一个状态leader,follower,candidate
	electionTimer  *time.Timer           // 选举用的
	heartBeatTimer *time.Timer           // leader定时给其他节点发送心跳用的
	applyChan      chan raftapi.ApplyMsg // 给外部将已commit日志进行apply的
}

// return currentTerm and whether this server
// believes it is the leader.
//
// return term, isLeader
func (rf *Raft) GetState() (int, bool) {

	// Your code here (3A).
	// 加锁是为了保证可见性
	rf.mu.RLock()
	defer rf.mu.RUnlock()

	return rf.currentTerm, rf.state == leader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// how many bytes in Raft's persisted log?
func (rf *Raft) PersistBytes() int {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.persister.RaftStateSize()
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term        int // vote节点的term
	CandidateId int // vote节点的idx
	LastLogIdx  int // vote节点log idx的标识
	LastLogTerm int // vote节点最后的一个term
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	VoteGranted bool // 是否赞成
	Term        int  // 给断联太久的节点更新term
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	// 选举逻辑，内部存在过多竞态资源，需要进行上锁
	// todo:重置当前节点的超时机制
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer func() {
		DPrintf("当前节点%v:节点%v请求选举结果%v", rf.me, args.CandidateId, reply.VoteGranted)
		tester.Annotate("election",
			fmt.Sprintf("当前节点%v,节点%v请求选举结果%v", rf.me, args.CandidateId, reply.VoteGranted),
			fmt.Sprintf("当前节点:node(%v),term(%v) 请求节点:node(%v),term(%v) 选举结果%v", rf.me, rf.currentTerm, args.CandidateId, args.Term, reply.VoteGranted))

	}()
	// rf.electionTimer.Reset(GetRandomElectionTimeout())
	// voteFor和term绑定，第二个条件隐含了不论之前选举的时候voteFor给了谁，只关注当前的term期间的voteFor
	if args.Term < rf.currentTerm || args.Term == rf.currentTerm && !(rf.votedFor == -1 || rf.votedFor == args.CandidateId) {
		// 拒绝投票
		reply.Term, reply.VoteGranted = rf.currentTerm, false
		return
	}

	if args.Term > rf.currentTerm {
		// 如果args的term更大，则需要比较其他方面的信息，重置投票逻辑
		// 考虑到旧leader的场景，需要将状态转成follower
		rf.currentTerm, rf.votedFor = args.Term, -1
		rf.stateTransform(follower)
	}


	if !rf.isUpToDateLog(args.LastLogIdx, args.LastLogTerm) {
		// 当前的logIdx和lastLogTerm不是最新的
		reply.Term, reply.VoteGranted = rf.currentTerm, false
		return
	}

	// 最终返回愿意投票
	// 重置当前节点的超时选举时间
	rf.electionTimer.Reset(GetRandomElectionTimeout())
	rf.votedFor = args.CandidateId
	reply.Term, reply.VoteGranted = rf.currentTerm, true
}

func (rf *Raft) isUpToDateLog(lastLogIdx int, lastLogTerm int) bool {
	rfLastLogIdx := len(rf.log) - 1
	rfLastLogTerm := rf.log[rfLastLogIdx].Term
	return lastLogTerm > rfLastLogTerm || lastLogTerm == rfLastLogTerm && lastLogIdx >= rfLastLogIdx
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

type AppendEntriesArgs struct {
	Term       int        // 当前的term
	LeadId     int        // leader id
	PrevLogIdx int        // 最后一个log的idx
	PreLogTerm int        // 最后一个log的term
	Entries    []LogEntry // 发送给follower的entry
	LeadCommit int        // leader的commitIdx
}

type AppendEntriesReply struct {
	Term    int  // follower的term，如果leader和follower之间对不上，从旧leader变成follower
	Success bool // 是否可以添加成功
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	// follower处理append请求，涉及到追加日志之类的写请求，直接加写锁
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// 先简单的做选举的逻辑
	// 对于选举这个问题来说，第一个需要处理的就是当args.term比自己大的时候，不论是否为leader，都应该将自己变成follower
	// 如果是
	if args.Term > rf.currentTerm {
		rf.currentTerm, rf.votedFor = args.Term, -1
		rf.stateTransform(follower)
	}
	rf.electionTimer.Reset(GetRandomElectionTimeout())
	DPrintf("节点%v接收心跳并重置计时器", rf.me)

	reply.Term, reply.Success = rf.currentTerm, true
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
// return index, term, isLeader
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != leader {
		// 应该直接返回错误
	}

	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) sendHeartBeat(isHeartBeat bool) {
	// 保证原子化读，加读锁
	rf.mu.RLock()
	args := &AppendEntriesArgs{
		Term:       rf.currentTerm,
		LeadId:     rf.me,
		PrevLogIdx: len(rf.log),
		PreLogTerm: rf.log[len(rf.log)-1].Term,
		LeadCommit: rf.commitIdx,
	}
	rf.mu.RUnlock()

	for i := range rf.peers {
		if i == rf.me {
			continue
		}

		// 给其他节点发送消息后，对于选举环节来说，只需要处理别人的term就行了
		go func(server int) {
			reply := &AppendEntriesReply{}
			// DPrintf("节点%v isleader%v 给节点%v 发送心跳", rf.me, rf.state == leader, server)
			if rf.sendAppendEntries(server, args, reply) {
				rf.mu.Lock()
				defer rf.mu.Unlock()

				// 如果term比自己大，那么就应该更新term，并且将自己转化成follower(当前节点可能是老leader)
				if reply.Term > rf.currentTerm {
					rf.currentTerm = reply.Term
					rf.stateTransform(follower)
				}
			}
		}(i)

	}

}

// candidate用的，进行选举的逻辑
func (rf *Raft) startElection() {
	// 问题简化：candidate选举不需要和超时机制同步进行
	rf.currentTerm++
	rf.votedFor = rf.me

	// 在处理其他vote的时候，可能会涉及到这些临界资源的修改，所以单独拿出来，外部加锁了，所以这里的资源是安全的
	args := &RequestVoteArgs{
		Term:        rf.currentTerm,
		CandidateId: rf.me,
		LastLogIdx:  len(rf.log) - 1,
		LastLogTerm: rf.log[len(rf.log)-1].Term,
	}

	// 进入选举状态开始选举
	voteCnt := 1

	for i := range rf.peers {
		if i == rf.me {
			continue
		}

		// 发送选举请求
		go func(server int) {
			reply := &RequestVoteReply{}
			if rf.sendRequestVote(server, args, reply) {
				// todo：这里其实可以选择在外面建立一个新的锁来提升安全和效率
				rf.mu.Lock()
				defer rf.mu.Unlock()

				// 只有当前轮次的选举消息是有效的（防历史rpc），并且如果节点的state变成了leader或者follower，则后续就不再需要处理了
				// args.term这个判断，如果自身的term再后续处理里被更新了，那么应该结束这个term的选举流程，不应该再推进了
				// rf.state的判断则是保证自己不论是变成follower还是leader，也不需要处理后续流程了
				DPrintf("节点%v向节点%v发起投票 结果是%v", rf.me, server, reply.VoteGranted)
				if args.Term == rf.currentTerm && rf.state == candidate {
					if reply.VoteGranted {
						voteCnt++
						if voteCnt > len(rf.peers)/2 {
							// 选举成功，开始发送心跳
							// todo：leader上任以后，对数据需要做一些额外处理
							Record("election", "选举成功", fmt.Sprintf("节点%v选举成功", rf.me))
							rf.stateTransform(leader)
							// 这里必须要用go，因为发送心跳的逻辑就有上锁逻辑，go又不设计可重入
							go rf.sendHeartBeat(true)
						}

					} else if reply.Term > rf.currentTerm {
						// 当前节点的term有问题，可能是断联了很久的节点突然醒了，直接进入follower
						// term和voteFor同步更新
						rf.currentTerm, rf.votedFor = reply.Term, -1
						rf.stateTransform(follower)
					}
				}

			}

		}(i)

	}

}

// follower用的，到点进行选举
func (rf *Raft) ticker() {
	// 核心麻烦点在于如何处理超时刷新机制之类的东西
	// ticker里的超时计时器，通过stateTransfer进行关停和重开
	for rf.killed() == false {

		// Your code here (3A)
		// Check if a leader election should be started.

		// pause for a random amount of time between 50 and 350
		// milliseconds.
		// todo：处理一下当选了leader后的逻辑
		// todo：超时选举的逻辑处理，需要配合heartbeat，在时间逻辑上需要特别注意
		// 随机分布在500-1000ms
		select {
		case <-rf.electionTimer.C:
			// 超时，开始进行选举，状态转换，问题进行简化，在选举过程中
			rf.mu.Lock()
			Record("election",
				fmt.Sprintf("节点%v开始进行选举", rf.me),
				fmt.Sprintf("节点%v超时开始进行选举,当前term是:%v", rf.me, rf.currentTerm))
			rf.stateTransform(candidate)
			rf.startElection()
			// 直接进行重置，如果选举成功了，在状态转化里进行终止
			// rf.electionTimer.Reset(GetRandomElectionTimeout())
			rf.mu.Unlock()
		case <-rf.heartBeatTimer.C:
			// 发送心跳包
			// 如果当前是leader，则发送心跳，并且重置计时器
			if rf.state == leader {
				go rf.sendHeartBeat(true)
				rf.heartBeatTimer.Reset(GetStableHeartBeatTimeout())
			}
		}
	}
}

// 状态转化只管打点计时器相关的部分
// 目前状态转化只存在几种形式
// follower->follower, follower->candidate
// candidate->follower, candidate->leader
// leader->follower
func (rf *Raft) stateTransform(state int) {
	// todo 可能需要考虑一点数据安全性的问题，但是一般状态转化的时候外面都会上锁
	// 上锁的核心原因在于rf的一些数据需要进行写操作修改，为了保证这个环节的正确性，防止意外，还是上锁安全
	// if rf.state == state {
	// 	// 重复请求不做处理
	// 	return
	// }
	rf.state = state

	switch state {
	case leader:
		// 终止选举超时器，开启心跳发送器
		rf.electionTimer.Stop()
		rf.heartBeatTimer.Reset(GetStableHeartBeatTimeout())
	case follower:
		// 开启选举超时器
		rf.electionTimer.Reset(GetRandomElectionTimeout())
	case candidate:
		// 开启选举超时器
		rf.electionTimer.Reset(GetRandomElectionTimeout())
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *tester.Persister, applyCh chan raftapi.ApplyMsg) raftapi.Raft {
	// 这段已经表明了，server端固定了所有的client并且会将通讯方式传进来，这些初始状态参数不太需要修改
	// 初始化的时候就需要开始后台线程来进行选举的工作
	rf := &Raft{
		mu:             sync.RWMutex{},
		peers:          peers,
		persister:      persister,
		me:             me,
		dead:           0,
		currentTerm:    0,
		votedFor:       -1,
		log:            make([]LogEntry, 1),
		commitIdx:      0,
		lastApplied:    0,
		nextIdx:        make([]int, 1),
		matchIdx:       make([]int, 1),
		state:          follower,
		electionTimer:  time.NewTimer(GetRandomElectionTimeout()),
		heartBeatTimer: time.NewTimer(GetStableHeartBeatTimeout()),
		applyChan:      applyCh,
	}

	// Your initialization code here (3A, 3B, 3C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
