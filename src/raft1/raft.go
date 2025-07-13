package raft

// The file raftapi/raft.go defines the interface that raft must
// expose to servers (or the tester), but see comments below for each
// of these functions for more details.
//
// Make() creates a new raft peer that implements the raft interface.

import (
	//	"bytes"
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labgob"
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
	return time.Duration(HeartBeatTimeout) * time.Millisecond
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
	replicatorCond []*sync.Cond          // 用来唤醒同步日志的底层机制
	applyChanCond  *sync.Cond            // 用于进行commit和apply的逻辑
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
//
// 任何修改了三个需要持久化状态的点都需要进行persist
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)

	// persist的时候在外面加锁，在这加锁容易死锁
	// rf.mu.RLock()
	// defer rf.mu.RUnlock()

	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	raftstate := w.Bytes()
	rf.persister.Save(raftstate, nil)
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

	// rf.mu.Lock()
	// defer rf.mu.Unlock()

	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm, votedFor int
	var log []LogEntry
	if d.Decode(&currentTerm) != nil || d.Decode(&votedFor) != nil || d.Decode(&log) != nil {
		DPrintf("curNode:%v decoder err current:%v voteFor:%v log:%v", rf.me, currentTerm, votedFor, log)
	}

	rf.currentTerm = currentTerm
	rf.votedFor = votedFor
	rf.log = log
	DPrintf("Recovery curNode:%v rf.log[%v]", rf.me, rf.log)
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
		// DPrintf("当前节点%v:节点%v请求选举结果%v", rf.me, args.CandidateId, reply.VoteGranted)
		Record("election",
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
		rf.stateTransform(follower)
		rf.currentTerm, rf.votedFor = args.Term, -1
		rf.persist()
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
	rf.persist()
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
	Term         int        // 当前的term
	LeaderId     int        // leader id
	PrevLogIdx   int        // 最后一个log的idx
	PreLogTerm   int        // 最后一个log的term
	Entries      []LogEntry // 发送给follower的entry
	LeaderCommit int        // leader的commitIdx
}

type AppendEntriesReply struct {
	Term         int  // follower的term，如果leader和follower之间对不上，从旧leader变成follower
	Success      bool // 是否可以添加成功
	ConflictIdx  int  // 日志冲突点
	ConflictTerm int  // 日志冲突点的Term
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	// follower处理append请求，涉及到追加日志之类的写请求，直接加写锁
	rf.mu.Lock()
	defer func() {
		if args.Term >= rf.currentTerm {
			if !reply.Success {
				// 日志复制失败
				Record(fmt.Sprintf(NodeTag, rf.me), AEConflictDesc,
					fmt.Sprintf("AE Conflict:当前节点:%v 当前leader:%v 冲突点%v 重定位点[idx:%v, term:%v]", rf.me, args.LeaderId, args.PrevLogIdx, reply.ConflictIdx, reply.ConflictTerm))
			} else if len(args.Entries) > 0 {
				// commit信息补充
				Record(fmt.Sprintf(NodeTag, rf.me), AEAccessDesc,
					fmt.Sprintf("AE Access:当前节点%v 当前leader:%v 提交点%v 日志长度%v", rf.me, args.LeaderId, rf.commitIdx, len(rf.log)-1))
			}
		}
	}()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		// 老leader的请求不做任何处理
		reply.Term, reply.Success = rf.currentTerm, false
		return
	}
	// 先简单的做选举的逻辑
	// 对于选举这个问题来说，第一个需要处理的就是当args.term比自己大的时候，不论是否为leader，都应该将自己变成follower
	// 如果是
	if args.Term > rf.currentTerm {
		rf.stateTransform(follower)
		rf.currentTerm, rf.votedFor = args.Term, -1
		rf.persist()
	}
	rf.electionTimer.Reset(GetRandomElectionTimeout())
	// DPrintf("节点%v接收心跳并重置计时器", rf.me)

	// 进行日志追加逻辑
	// 1.follower日志只落后leader，但是无冲突
	// 2.follower和leader日志冲突，需要进行覆盖问题

	// args.preLogIdx比当前log长度长 或者 preLogIdx点上两者的term不一致，则返回false
	if curLogLastIdx := len(rf.log) - 1; curLogLastIdx < args.PrevLogIdx || rf.log[args.PrevLogIdx].Term != args.PreLogTerm {
		if curLogLastIdx < args.PrevLogIdx {
			reply.ConflictIdx, reply.ConflictTerm = curLogLastIdx, rf.log[curLogLastIdx].Term
		} else if rf.log[args.PrevLogIdx].Term != args.PreLogTerm {
			// 为了防止极端情况陷入的死循环，所以需要保证冲突点的日志term<=args.Term
			for i := args.PrevLogIdx; i >= 0; i-- {
				if rf.log[i].Term <= args.PreLogTerm {
					reply.ConflictIdx, reply.ConflictTerm = args.PrevLogIdx, rf.log[args.PrevLogIdx].Term
					break
				}
			}
		}

		reply.Term, reply.Success = rf.currentTerm, false
		return
	}

	// 开始进行同步
	// 如果是空包就没必要浪费时间再复制一份了
	if len(args.Entries) > 0 {
		rf.log = append(rf.log[0:args.PrevLogIdx+1], args.Entries...)
		rf.persist()
		DPrintf("节点%v log rep ", rf.me)
	}

	// commit逻辑
	if args.LeaderCommit > rf.commitIdx {
		rf.commitIdx = min(args.LeaderCommit, len(rf.log)-1)
		rf.applyChanCond.Signal()
	}

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
	// Your code here (3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != leader {
		// 应该直接返回错误
		return -1, -1, false
	}

	index := len(rf.log)
	term := rf.currentTerm
	isLeader := rf.state == leader

	// todo:在rf.log中添加新日志，并且广播出去
	newLog := LogEntry{Command: command, Term: term}
	rf.log = append(rf.log, newLog)
	rf.persist()
	// 更新自己的matchIdx方便逻辑处理
	rf.matchIdx[rf.me] = len(rf.log) - 1
	go rf.sendHeartBeat(false)
	// DPrintf("start")

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

// 发送心跳和日志复制
func (rf *Raft) sendHeartBeat(isHeartBeat bool) {
	// todo:现在是lab3B了，需要对这段逻辑进行大改
	for server := range rf.peers {
		if server == rf.me {
			continue
		}

		if !isHeartBeat {
			// todo 可以单独处理非心跳的处理逻辑
		} else {
			// DPrintf("leader开始发送心跳")
		}
		// DPrintf("heart signal 唤醒%v ",server)

		rf.replicatorCond[server].Signal()
	}

}

// 生成AE，加锁的事情自己外面保证
func (rf *Raft) generateAppendEntriesArgs(server int) *AppendEntriesArgs {
	preLogIdx := rf.nextIdx[server] - 1
	// 执行深拷贝，是为了防止log发生扩容而产生的线程安全问题
	entries := make([]LogEntry, len(rf.log[preLogIdx+1:]))
	copy(entries, rf.log[preLogIdx+1:])
	args := &AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIdx:   preLogIdx,
		PreLogTerm:   rf.log[preLogIdx].Term,
		LeaderCommit: rf.commitIdx,
		Entries:      entries,
	}
	return args
}

// 发送和处理AE的逻辑，同步follower和leader之间的日志逻辑
func (rf *Raft) syncLogOnce(server int) {
	rf.mu.RLock()

	if rf.state != leader {
		// 为了解决非原子性的问题，外部逻辑判断再发送消息，并非原子的所以需要有这一段
		rf.mu.RUnlock()
		return
	}

	args := rf.generateAppendEntriesArgs(server)
	rf.mu.RUnlock()

	reply := &AppendEntriesReply{}
	if rf.sendAppendEntries(server, args, reply) {
		rf.mu.Lock()
		defer rf.mu.Unlock()

		// 这一步和election差不多，任何时候都只处理当前term的请求，以及需要确保自己的状态
		// 原则上不处理过期数据
		if args.Term == rf.currentTerm && rf.state == leader {
			// 判断一下unsuccess的原因
			if !reply.Success {
				// 如果term比自己大，那么就应该更新term，并且将自己转化成follower(当前节点可能是老leader)
				if reply.Term > rf.currentTerm {
					rf.stateTransform(follower)
					rf.currentTerm, rf.votedFor = reply.Term, -1
					rf.persist()
				} else {
					// 开始着手解决日志冲突的问题
					// 从conflict点出发，知道碰见第一个<=conflictTerm的log
					for i := reply.ConflictIdx; i >= 0; i-- {
						if rf.log[i].Term <= reply.ConflictTerm {
							// 如果rf.log[reply.ConflictIdx].Term <= reply.ConflictTerm，就会陷入无限死循环
							// 理论上不会出现上面说的，因为term大的才能选举成功，但是极端环境下还是会出现的
							// 假如旧leader的日志没有复制过去，并且正好断联了，新leader上任后又重联了，就会出现上面的问题
							rf.nextIdx[server] = i + 1
							break
						}

						// DPrintf("Log rep Conflict")
					}
				}
			} else if len(args.Entries) > 0 {
				// 增加if 对于心跳逻辑没必要单独多做其他方面的判断(心跳本身就是没有冲突的表示，就算回来的rpc丢了也不会有没同步的问题，下次依然会带着log发过去)
				// 日志追加成功，commit逻辑
				rf.nextIdx[server] = rf.nextIdx[server] + len(args.Entries)
				rf.matchIdx[server] = rf.nextIdx[server] - 1
				// DPrintf("follower %v rf.nextIdx:%v rf.matchIdx:%v", server, rf.nextIdx[server], rf.matchIdx[server])
				rf.commitAndApply()
			}

		}

	}
}

// 判断一下当前状态是否能进行commit，然后将apply唤醒
func (rf *Raft) commitAndApply() {
	matchIdx := make([]int, len(rf.peers))
	copy(matchIdx, rf.matchIdx)

	sort.Ints(matchIdx)

	// 日志过半复制点，进行commit，并唤醒apply
	// 为了防止奇怪的错误，leader只能间接提交其他term内的日志(论文5.4)
	overHalfReplicatedIdx := matchIdx[len(rf.peers)/2]
	if overHalfReplicatedIdx > rf.commitIdx && rf.log[overHalfReplicatedIdx].Term == rf.currentTerm {
		rf.commitIdx = overHalfReplicatedIdx
		rf.applyChanCond.Signal()
		Record(LogCommitTag, fmt.Sprintf(LogCommitDesc, overHalfReplicatedIdx),
			fmt.Sprintf("Log Commit 当前leader:%v commitIdx:%v", rf.me, rf.commitIdx))
	}
}

// 判断当前是否应该同步leader和follower[server]之间的日志差异
func (rf *Raft) isLogSync(server int) bool {
	// todo:加锁和比较日志逻辑
	rf.mu.RLock()
	defer rf.mu.RUnlock()
	return rf.state == leader && rf.nextIdx[server] == len(rf.log)
}

// 心跳和日志复制的底层机制
func (rf *Raft) replicator(server int) {
	rf.replicatorCond[server].L.Lock()
	defer rf.replicatorCond[server].L.Unlock()

	for rf.killed() == false {
		// 被唤醒了就发送一次日志同步
		rf.replicatorCond[server].Wait()
		DPrintf("leader:%v follower:%v ",rf.me,server)
		rf.syncLogOnce(server)
		// 进行同步检测，如果leader和follower之间日志不同步则通过for完成同步过程
		// 同样也需要保证在这个过程中state=leader，需要保证旧leader能正常变成follower
		// todo：判断和发送不是原子的，所以可能会出现一定的线程安全问题
		// 加入kill判断，如果没被kill才继续同步的逻辑
		for !rf.isLogSync(server) && !rf.killed() {
			rf.syncLogOnce(server)
		}
	}
}

// 当有日志被提交的时候，应该将其应用到rf.applyChan
func (rf *Raft) applier() {
	rf.applyChanCond.L.Lock()
	defer rf.applyChanCond.L.Unlock()

	for rf.killed() == false {
		for rf.commitIdx <= rf.lastApplied {
			rf.applyChanCond.Wait()

			// 这里应该加一个判断，如果kill则直接return
		}

		// 唤醒后开始进行commit逻辑
		rf.mu.RLock()
		// todo：这里有一个写操作，理论上应该加写锁
		// 保证可见性

		rf.lastApplied++
		applyMsg := raftapi.ApplyMsg{
			CommandValid: true,
			Command:      rf.log[rf.lastApplied].Command,
			CommandIndex: rf.lastApplied,
		}

		rf.mu.RUnlock()

		rf.applyChan <- applyMsg
		DPrintf("Log Apply 当前节点:%v applyLogTerm:%v applyLogIdx:%v ", rf.me, rf.log[rf.lastApplied].Term, applyMsg.CommandIndex)
	}
}

// leader上任成功后，需要进行一些参数的初始化
func (rf *Raft) initLeaderParms() {
	for server := range rf.peers {
		rf.nextIdx[server] = len(rf.log)
		rf.matchIdx[server] = 0
	}
}

// candidate用的，进行选举的逻辑
func (rf *Raft) startElection() {
	// 问题简化：candidate选举不需要和超时机制同步进行
	rf.currentTerm++
	rf.votedFor = rf.me
	rf.persist()

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

							Record(ElectionTag,
								fmt.Sprintf(ElectionResDesc, rf.me),
								fmt.Sprintf("Election success 节点%v选举成功 当前term:%v lastLog[idx:%v,term:%v]", rf.me, rf.currentTerm, len(rf.log)-1, rf.log[len(rf.log)-1].Term))

							rf.stateTransform(leader)
							rf.initLeaderParms()
							// election和log replicate用一个协程专门负责沟通，虽然也可以不用go
							go rf.sendHeartBeat(true)
						}

					} else if reply.Term > rf.currentTerm {
						// 当前节点的term有问题，可能是断联了很久的节点突然醒了，直接进入follower
						// term和voteFor同步更新
						rf.stateTransform(follower)
						rf.currentTerm, rf.votedFor = reply.Term, -1
						rf.persist()
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
			// 应该考虑3C里被kill的情况，kill后就不应该发起election了
			if rf.killed(){
				return
			}
			rf.mu.Lock()

			Record(ElectionTag,
				fmt.Sprintf(ElectionStartDesc, rf.me),
				fmt.Sprintf("节点%v开始进行选举 当前term:%v", rf.me, rf.currentTerm))

			rf.stateTransform(candidate)
			rf.startElection()
			// 直接进行重置，如果选举成功了，在状态转化里进行终止
			rf.electionTimer.Reset(GetRandomElectionTimeout())
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
// candidate->follower, candidate->leader，candidate->candidate
// leader->follower
func (rf *Raft) stateTransform(state int) {
	// todo 可能需要考虑一点数据安全性的问题，但是一般状态转化的时候外面都会上锁
	// 上锁的核心原因在于rf的一些数据需要进行写操作修改，为了保证这个环节的正确性，防止意外，还是上锁安全
	if rf.state == state {
		// follower->follower这情况下，为了防止lab3C新加入的节点容易在election上浪费太多时间
		return
	}
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
		nextIdx:        make([]int, len(peers)),
		matchIdx:       make([]int, len(peers)),
		state:          follower,
		electionTimer:  time.NewTimer(GetRandomElectionTimeout()),
		heartBeatTimer: time.NewTimer(GetStableHeartBeatTimeout()),
		applyChan:      applyCh,
		replicatorCond: make([]*sync.Cond, len(peers)),
		applyChanCond:  sync.NewCond(&sync.Mutex{}),
	}

	// Your initialization code here (3A, 3B, 3C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	for server := range rf.peers {
		if server != rf.me {
			rf.replicatorCond[server] = sync.NewCond(&sync.Mutex{})
			go rf.replicator(server)
		}

	}
	// start ticker goroutine to start elections
	go rf.ticker()
	go rf.applier()

	return rf
}
