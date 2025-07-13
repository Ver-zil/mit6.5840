# lab3

## lab3c

### test2

#### dc98fd99d282fdc12a235a540e8b8ecb140ba599

---------
初始版本
---------

Starting: /home/verzil/coding/compiler/goworks/bin/dlv dap --listen=127.0.0.1:39323 --log-dest=3 from /home/verzil/coding/project/6.5840/src/raft1
DAP server listening at: 127.0.0.1:39323
Type 'dlv help' for list of commands.
Test (3C): more persistence (reliable network)...
2025/07/13 07:18:31 节点1开始进行选举 当前term:0
2025/07/13 07:18:31 当前节点:node(2),term(1) 请求节点:node(1),term(1) 选举结果true
2025/07/13 07:18:31 当前节点:node(4),term(1) 请求节点:node(1),term(1) 选举结果true
2025/07/13 07:18:31 节点1向节点2发起投票 结果是true
2025/07/13 07:18:31 节点1向节点4发起投票 结果是true
2025/07/13 07:18:31 Election success 节点1选举成功 当前term:1 lastLog[idx:0,term:0]
2025/07/13 07:18:31 当前节点:node(0),term(1) 请求节点:node(1),term(1) 选举结果true
2025/07/13 07:18:31 当前节点:node(3),term(1) 请求节点:node(1),term(1) 选举结果true
2025/07/13 07:18:31 节点1向节点0发起投票 结果是true
2025/07/13 07:18:31 节点1向节点3发起投票 结果是true
2025/07/13 07:18:31 节点2 log rep 
2025/07/13 07:18:31 节点4 log rep
2025/07/13 07:18:31 AE Access:当前节点4 当前leader:1 提交点0 日志长度1
2025/07/13 07:18:31 节点3 log rep 
2025/07/13 07:18:31 AE Access:当前节点3 当前leader:1 提交点0 日志长度1
2025/07/13 07:18:31 节点0 log rep 
2025/07/13 07:18:31 AE Access:当前节点0 当前leader:1 提交点0 日志长度1
2025/07/13 07:18:31 AE Access:当前节点2 当前leader:1 提交点0 日志长度1
2025/07/13 07:18:31 Log Commit 当前leader:1 commitIdx:1
2025/07/13 07:18:31 Log Apply 当前节点:1 applyLogTerm:1 applyLogIdx:1
2025/07/13 07:18:32 Log Apply 当前节点:2 applyLogTerm:1 applyLogIdx:1 
2025/07/13 07:18:32 Log Apply 当前节点:4 applyLogTerm:1 applyLogIdx:1 
2025/07/13 07:18:32 Log Apply 当前节点:3 applyLogTerm:1 applyLogIdx:1 
2025/07/13 07:18:32 Log Apply 当前节点:0 applyLogTerm:1 applyLogIdx:1
2025/07/13 07:18:32 ShutDown Node [2, 3]
2025/07/13 07:18:32 节点0 log rep
2025/07/13 07:18:32 AE Access:当前节点0 当前leader:1 提交点1 日志长度2
2025/07/13 07:18:32 节点4 log rep 
2025/07/13 07:18:32 AE Access:当前节点4 当前leader:1 提交点1 日志长度2
2025/07/13 07:18:32 Log Commit 当前leader:1 commitIdx:2
2025/07/13 07:18:32 Log Apply 当前节点:1 applyLogTerm:1 applyLogIdx:2
2025/07/13 07:18:32 Log Apply 当前节点:0 applyLogTerm:1 applyLogIdx:2 
2025/07/13 07:18:32 Log Apply 当前节点:4 applyLogTerm:1 applyLogIdx:2
2025/07/13 07:18:32 ShutDown Node [1, 4, 0]
2025/07/13 07:18:32 Recovery curNode:2 rf.log[[{0 <nil>} {1 11}]]
2025/07/13 07:18:32 Recovery curNode:3 rf.log[[{0 <nil>} {1 11}]]
2025/07/13 07:18:32 Restart Node [2, 3]
2025/07/13 07:18:32 节点2 log rep 
2025/07/13 07:18:32 AE Access:当前节点2 当前leader:1 提交点2 日志长度2
2025/07/13 07:18:32 Log Apply 当前节点:2 applyLogTerm:1 applyLogIdx:1 
2025/07/13 07:18:32 Log Apply 当前节点:2 applyLogTerm:1 applyLogIdx:2
-------------
问题就在这了，正常情况下leader 1应该已经停止运作了，但是还是将log发送给了刚重启的节点2
所以关键点在于，因为leader1的log没有和follower2同步，所以replicator卡在循环里没真的kill
而对于网络体系来说，shutdown的节点不能接收信号，但是能发送信号给活的节点
但是follower3没有进行同步，原因可能是因为节点0开始进行选举（leader1给follower0的replicator停止了）了，term更大，所以节点1从leader变follower，然后真正的停止了
-------------
2025/07/13 07:18:33 节点0开始进行选举 当前term:1
2025/07/13 07:18:33 当前节点:node(3),term(2) 请求节点:node(0),term(2) 选举结果true
2025/07/13 07:18:33 当前节点:node(2),term(2) 请求节点:node(0),term(2) 选举结果true
2025/07/13 07:18:33 节点0向节点2发起投票 结果是true
2025/07/13 07:18:33 节点0向节点3发起投票 结果是true
2025/07/13 07:18:33 Election success 节点0选举成功 当前term:2 lastLog[idx:2,term:1]
2025/07/13 07:18:33 AE Conflict:当前节点:3 当前leader:0 冲突点2 重定位点[idx:1, term:1]
2025/07/13 07:18:33 节点3 log rep 
2025/07/13 07:18:33 AE Access:当前节点3 当前leader:0 提交点2 日志长度2
2025/07/13 07:18:33 Log Apply 当前节点:3 applyLogTerm:1 applyLogIdx:1 
2025/07/13 07:18:33 Log Apply 当前节点:3 applyLogTerm:1 applyLogIdx:2
-----------
后面为什么发生了这么多次的选举？按道理说说leader0应该给他们发送心跳才对吧
因为leader0其实已经被关闭了，但是因为没有进入下一次的循环，所以没有kill，所以进行了选举，并成为了leader
日志同步的replicator停止了，只留下了没有日志同步的replicator的3，但是日志同步完以后，检测到kill就结束了
-----------
2025/07/13 07:18:33 节点3开始进行选举 当前term:1
2025/07/13 07:18:33 当前节点:node(2),term(2) 请求节点:node(3),term(2) 选举结果false
2025/07/13 07:18:33 节点3向节点2发起投票 结果是false
2025/07/13 07:18:33 节点4开始进行选举 当前term:1
2025/07/13 07:18:33 当前节点:node(3),term(2) 请求节点:node(4),term(2) 选举结果false
2025/07/13 07:18:33 当前节点:node(2),term(2) 请求节点:node(4),term(2) 选举结果false
2025/07/13 07:18:33 节点4向节点3发起投票 结果是false
2025/07/13 07:18:33 节点4向节点2发起投票 结果是false
2025/07/13 07:18:33 节点2开始进行选举 当前term:1
---------
节点4的问题也是一样的，没被真正的kill
---------
2025/07/13 07:18:33 Recovery curNode:4 rf.log[[{0 <nil>} {1 11} {1 12}]]
2025/07/13 07:18:33 Restart Node [4]
2025/07/13 07:18:33 节点3开始进行选举 当前term:2
2025/07/13 07:18:33 当前节点:node(4),term(3) 请求节点:node(3),term(3) 选举结果true
2025/07/13 07:18:33 当前节点:node(2),term(3) 请求节点:node(3),term(3) 选举结果true
2025/07/13 07:18:33 节点3向节点2发起投票 结果是true
2025/07/13 07:18:33 节点3向节点4发起投票 结果是true
2025/07/13 07:18:33 Election success 节点3选举成功 当前term:3 lastLog[idx:2,term:1]
2025/07/13 07:18:33 Log Apply 当前节点:4 applyLogTerm:1 applyLogIdx:1 
2025/07/13 07:18:33 Log Apply 当前节点:4 applyLogTerm:1 applyLogIdx:2
2025/07/13 07:18:33 节点4 log rep 
2025/07/13 07:18:33 AE Access:当前节点4 当前leader:3 提交点2 日志长度3
2025/07/13 07:18:33 节点2 log rep 
2025/07/13 07:18:33 AE Access:当前节点2 当前leader:3 提交点2 日志长度3
2025/07/13 07:18:33 Log Commit 当前leader:3 commitIdx:3
2025/07/13 07:18:33 Log Apply 当前节点:3 applyLogTerm:3 applyLogIdx:3
2025/07/13 07:18:33 Log Apply 当前节点:2 applyLogTerm:3 applyLogIdx:3 
2025/07/13 07:18:33 Log Apply 当前节点:4 applyLogTerm:3 applyLogIdx:3
2025/07/13 07:18:33 Recovery curNode:0 rf.log[[{0 <nil>} {1 11} {1 12}]]
2025/07/13 07:18:33 Recovery curNode:1 rf.log[[{0 <nil>} {1 11} {1 12}]]
2025/07/13 07:18:33 Restart Node [0, 1]
Detaching and terminating target process
dlv dap (411668) exited with code: 0


---------------
加入replicator和ticker停止逻辑
--------------
Test (3C): more persistence (reliable network)...
2025/07/13 15:02:06 节点0开始进行选举 当前term:0
2025/07/13 15:02:06 当前节点:node(1),term(1) 请求节点:node(0),term(1) 选举结果true
2025/07/13 15:02:06 当前节点:node(4),term(1) 请求节点:node(0),term(1) 选举结果true
2025/07/13 15:02:06 节点0向节点1发起投票 结果是true
2025/07/13 15:02:06 节点0向节点4发起投票 结果是true
2025/07/13 15:02:06 Election success 节点0选举成功 当前term:1 lastLog[idx:0,term:0]
2025/07/13 15:02:06 当前节点:node(3),term(1) 请求节点:node(0),term(1) 选举结果true
2025/07/13 15:02:06 leader:0 follower:4 
2025/07/13 15:02:06 节点0向节点3发起投票 结果是true
2025/07/13 15:02:06 当前节点:node(2),term(1) 请求节点:node(0),term(1) 选举结果true
2025/07/13 15:02:06 节点0向节点2发起投票 结果是true
2025/07/13 15:02:06 leader:0 follower:2 
2025/07/13 15:02:06 leader:0 follower:1 
2025/07/13 15:02:06 leader:0 follower:3 
2025/07/13 15:02:06 leader:0 follower:4 
2025/07/13 15:02:06 leader:0 follower:2 
2025/07/13 15:02:06 节点4 log rep 
2025/07/13 15:02:06 AE Access:当前节点4 当前leader:0 提交点0 日志长度1
2025/07/13 15:02:06 节点2 log rep 
2025/07/13 15:02:06 AE Access:当前节点2 当前leader:0 提交点0 日志长度1
2025/07/13 15:02:06 leader:0 follower:1 
2025/07/13 15:02:06 Log Commit 当前leader:0 commitIdx:1
2025/07/13 15:02:06 Log Apply 当前节点:0 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:06 leader:0 follower:3 
2025/07/13 15:02:06 节点3 log rep 
2025/07/13 15:02:06 AE Access:当前节点3 当前leader:0 提交点1 日志长度1
2025/07/13 15:02:06 节点1 log rep 
2025/07/13 15:02:06 AE Access:当前节点1 当前leader:0 提交点0 日志长度1
2025/07/13 15:02:06 Log Apply 当前节点:3 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:07 leader:0 follower:4 
2025/07/13 15:02:07 leader:0 follower:2 
2025/07/13 15:02:07 leader:0 follower:3 
2025/07/13 15:02:07 leader:0 follower:1 
2025/07/13 15:02:07 Log Apply 当前节点:4 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:07 Log Apply 当前节点:1 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:07 Log Apply 当前节点:2 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:07 leader:0 follower:2 
2025/07/13 15:02:07 leader:0 follower:4 
2025/07/13 15:02:07 leader:0 follower:3 
2025/07/13 15:02:07 leader:0 follower:1 
2025/07/13 15:02:07 leader:0 follower:4 
2025/07/13 15:02:07 leader:0 follower:2 
2025/07/13 15:02:07 leader:0 follower:1 
2025/07/13 15:02:07 leader:0 follower:3 
2025/07/13 15:02:07 leader:0 follower:4 
2025/07/13 15:02:07 leader:0 follower:1 
2025/07/13 15:02:07 leader:0 follower:2 
2025/07/13 15:02:07 leader:0 follower:3 
2025/07/13 15:02:07 leader:0 follower:4 
2025/07/13 15:02:07 leader:0 follower:2 
2025/07/13 15:02:07 leader:0 follower:1 
2025/07/13 15:02:07 leader:0 follower:3 
2025/07/13 15:02:07 ShutDown Node [1, 2]
2025/07/13 15:02:07 leader:0 follower:4 
2025/07/13 15:02:07 leader:0 follower:2 
2025/07/13 15:02:07 leader:0 follower:1 
2025/07/13 15:02:07 leader:0 follower:3 
2025/07/13 15:02:07 节点4 log rep 
2025/07/13 15:02:07 AE Access:当前节点4 当前leader:0 提交点1 日志长度2
2025/07/13 15:02:07 节点3 log rep 
2025/07/13 15:02:07 AE Access:当前节点3 当前leader:0 提交点1 日志长度2
2025/07/13 15:02:07 Log Commit 当前leader:0 commitIdx:2
2025/07/13 15:02:07 Log Apply 当前节点:0 applyLogTerm:1 applyLogIdx:2 
2025/07/13 15:02:07 leader:0 follower:4 
2025/07/13 15:02:07 leader:0 follower:3 
2025/07/13 15:02:07 Log Apply 当前节点:4 applyLogTerm:1 applyLogIdx:2 
2025/07/13 15:02:07 Log Apply 当前节点:3 applyLogTerm:1 applyLogIdx:2 
2025/07/13 15:02:07 ShutDown Node [0, 3, 4]
2025/07/13 15:02:07 Recovery curNode:1 rf.log[[{0 <nil>} {1 11}]]
2025/07/13 15:02:07 Recovery curNode:2 rf.log[[{0 <nil>} {1 11}]]
2025/07/13 15:02:07 Restart Node [1, 2]
2025/07/13 15:02:07 leader:0 follower:4 
2025/07/13 15:02:07 leader:0 follower:3 
2025/07/13 15:02:08 节点2开始进行选举 当前term:1
2025/07/13 15:02:08 当前节点:node(1),term(2) 请求节点:node(2),term(2) 选举结果true
2025/07/13 15:02:08 节点2向节点1发起投票 结果是true
2025/07/13 15:02:08 Recovery curNode:3 rf.log[[{0 <nil>} {1 11} {1 12}]]
2025/07/13 15:02:08 Restart Node [3]
2025/07/13 15:02:09 节点1开始进行选举 当前term:2
2025/07/13 15:02:09 当前节点:node(2),term(3) 请求节点:node(1),term(3) 选举结果true
2025/07/13 15:02:09 当前节点:node(3),term(3) 请求节点:node(1),term(3) 选举结果false
2025/07/13 15:02:09 节点1向节点3发起投票 结果是false
2025/07/13 15:02:09 节点1向节点2发起投票 结果是true
2025/07/13 15:02:09 节点2开始进行选举 当前term:3
2025/07/13 15:02:09 当前节点:node(1),term(4) 请求节点:node(2),term(4) 选举结果true
2025/07/13 15:02:09 当前节点:node(3),term(4) 请求节点:node(2),term(4) 选举结果false
2025/07/13 15:02:09 节点2向节点1发起投票 结果是true
2025/07/13 15:02:09 节点2向节点3发起投票 结果是false
2025/07/13 15:02:09 节点3开始进行选举 当前term:4
2025/07/13 15:02:09 当前节点:node(1),term(5) 请求节点:node(3),term(5) 选举结果true
2025/07/13 15:02:09 当前节点:node(2),term(5) 请求节点:node(3),term(5) 选举结果true
2025/07/13 15:02:09 节点3向节点1发起投票 结果是true
2025/07/13 15:02:09 节点3向节点2发起投票 结果是true
2025/07/13 15:02:09 Election success 节点3选举成功 当前term:5 lastLog[idx:2,term:1]
2025/07/13 15:02:09 leader:3 follower:4 
2025/07/13 15:02:09 leader:3 follower:1 
2025/07/13 15:02:09 leader:3 follower:0 
2025/07/13 15:02:09 leader:3 follower:2 
2025/07/13 15:02:09 AE Conflict:当前节点:1 当前leader:3 冲突点2 重定位点[idx:1, term:1]
2025/07/13 15:02:09 AE Conflict:当前节点:2 当前leader:3 冲突点2 重定位点[idx:1, term:1]
2025/07/13 15:02:09 节点1 log rep 
2025/07/13 15:02:09 AE Access:当前节点1 当前leader:3 提交点0 日志长度2
2025/07/13 15:02:09 节点2 log rep 
2025/07/13 15:02:09 AE Access:当前节点2 当前leader:3 提交点0 日志长度2
2025/07/13 15:02:09 leader:3 follower:2 
2025/07/13 15:02:09 leader:3 follower:1 
2025/07/13 15:02:09 节点1 log rep 
2025/07/13 15:02:09 AE Access:当前节点1 当前leader:3 提交点0 日志长度3
2025/07/13 15:02:09 节点2 log rep 
2025/07/13 15:02:09 AE Access:当前节点2 当前leader:3 提交点0 日志长度3
2025/07/13 15:02:09 Log Commit 当前leader:3 commitIdx:3
2025/07/13 15:02:09 Log Apply 当前节点:3 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:09 Log Apply 当前节点:3 applyLogTerm:1 applyLogIdx:2 
2025/07/13 15:02:09 Log Apply 当前节点:3 applyLogTerm:5 applyLogIdx:3 
2025/07/13 15:02:09 leader:3 follower:2 
2025/07/13 15:02:09 leader:3 follower:1 
2025/07/13 15:02:09 Log Apply 当前节点:1 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:09 Log Apply 当前节点:1 applyLogTerm:1 applyLogIdx:2 
2025/07/13 15:02:09 Log Apply 当前节点:1 applyLogTerm:5 applyLogIdx:3 
2025/07/13 15:02:09 Log Apply 当前节点:2 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:09 Log Apply 当前节点:2 applyLogTerm:1 applyLogIdx:2 
2025/07/13 15:02:09 Log Apply 当前节点:2 applyLogTerm:5 applyLogIdx:3 
2025/07/13 15:02:09 Recovery curNode:4 rf.log[[{0 <nil>} {1 11} {1 12}]]
2025/07/13 15:02:09 Recovery curNode:0 rf.log[[{0 <nil>} {1 11} {1 12}]]
2025/07/13 15:02:09 Restart Node [4, 0]
2025/07/13 15:02:09 Seg -----------------------------------------------------------------
2025/07/13 15:02:09 leader:3 follower:2 
2025/07/13 15:02:09 leader:3 follower:1 
2025/07/13 15:02:09 节点1 log rep 
2025/07/13 15:02:09 AE Access:当前节点1 当前leader:3 提交点3 日志长度4
2025/07/13 15:02:09 节点2 log rep 
2025/07/13 15:02:09 AE Access:当前节点2 当前leader:3 提交点3 日志长度4
2025/07/13 15:02:09 Log Commit 当前leader:3 commitIdx:4
2025/07/13 15:02:09 Log Apply 当前节点:3 applyLogTerm:5 applyLogIdx:4 
2025/07/13 15:02:09 leader:3 follower:2 
2025/07/13 15:02:09 leader:3 follower:1 
2025/07/13 15:02:09 Log Apply 当前节点:2 applyLogTerm:5 applyLogIdx:4 
2025/07/13 15:02:09 Log Apply 当前节点:1 applyLogTerm:5 applyLogIdx:4 
2025/07/13 15:02:10 leader:3 follower:2 
2025/07/13 15:02:10 leader:3 follower:1 
2025/07/13 15:02:10 leader:3 follower:2 
2025/07/13 15:02:10 leader:3 follower:1 
2025/07/13 15:02:10 leader:3 follower:2 
2025/07/13 15:02:10 leader:3 follower:1 
2025/07/13 15:02:10 leader:3 follower:2 
2025/07/13 15:02:10 leader:3 follower:1 
-------------------
之前重新唤醒了节点0，4，在这里这两个之一一定会有一个进行选举的
因为原来leader节点里的peers没有更新，所以heartbeat发送不到这两个新唤醒的节点上
之前的一个stateTransform逻辑上，因为每次发送vote的时候，都会因为term的原因，transform一下
这个事情本身没什么问题，但是在这个test里，会因为节点4发送vote，其他节点重置自己的选举时间导致节点4有概率频繁进行选举又一直失败浪费很多时间，导致一直没有leader出现，最终导致test超时，所以对stateTransform进行一个重新设计
follower->follower和candidate->candidate不允许重置时间，这两个东西的重置逻辑直接自己写在外面进行把控
-------------------
2025/07/13 15:02:10 节点4开始进行选举 当前term:1
2025/07/13 15:02:10 当前节点:node(3),term(5) 请求节点:node(4),term(2) 选举结果false
2025/07/13 15:02:10 当前节点:node(1),term(5) 请求节点:node(4),term(2) 选举结果false
2025/07/13 15:02:10 当前节点:node(0),term(2) 请求节点:node(4),term(2) 选举结果true
2025/07/13 15:02:10 当前节点:node(2),term(5) 请求节点:node(4),term(2) 选举结果false
2025/07/13 15:02:10 节点4向节点1发起投票 结果是false
2025/07/13 15:02:10 节点4向节点0发起投票 结果是true
2025/07/13 15:02:10 节点4向节点3发起投票 结果是false
2025/07/13 15:02:10 节点4向节点2发起投票 结果是false
2025/07/13 15:02:10 leader:3 follower:2 
2025/07/13 15:02:10 leader:3 follower:1 
2025/07/13 15:02:10 leader:3 follower:2 
2025/07/13 15:02:10 leader:3 follower:1 
2025/07/13 15:02:10 leader:3 follower:2 
2025/07/13 15:02:10 leader:3 follower:1 
2025/07/13 15:02:11 leader:3 follower:2 
2025/07/13 15:02:11 leader:3 follower:1 
2025/07/13 15:02:11 leader:3 follower:2 
2025/07/13 15:02:11 leader:3 follower:1 
2025/07/13 15:02:11 leader:3 follower:2 
2025/07/13 15:02:11 leader:3 follower:1 
2025/07/13 15:02:11 节点0开始进行选举 当前term:2
2025/07/13 15:02:11 当前节点:node(4),term(5) 请求节点:node(0),term(3) 选举结果false
2025/07/13 15:02:11 节点0向节点4发起投票 结果是false
2025/07/13 15:02:11 当前节点:node(2),term(5) 请求节点:node(0),term(3) 选举结果false
2025/07/13 15:02:11 当前节点:node(3),term(5) 请求节点:node(0),term(3) 选举结果false
2025/07/13 15:02:11 节点0向节点2发起投票 结果是false
2025/07/13 15:02:11 当前节点:node(1),term(5) 请求节点:node(0),term(3) 选举结果false
2025/07/13 15:02:11 节点0向节点3发起投票 结果是false
2025/07/13 15:02:11 节点0向节点1发起投票 结果是false
2025/07/13 15:02:11 leader:3 follower:2 
2025/07/13 15:02:11 leader:3 follower:1 
2025/07/13 15:02:11 节点4开始进行选举 当前term:5
2025/07/13 15:02:11 当前节点:node(3),term(6) 请求节点:node(4),term(6) 选举结果false
2025/07/13 15:02:11 当前节点:node(1),term(6) 请求节点:node(4),term(6) 选举结果false
2025/07/13 15:02:11 当前节点:node(2),term(6) 请求节点:node(4),term(6) 选举结果false
2025/07/13 15:02:11 节点4向节点3发起投票 结果是false
2025/07/13 15:02:11 当前节点:node(0),term(6) 请求节点:node(4),term(6) 选举结果true
2025/07/13 15:02:11 节点4向节点2发起投票 结果是false
2025/07/13 15:02:11 节点4向节点1发起投票 结果是false
2025/07/13 15:02:11 节点4向节点0发起投票 结果是true
2025/07/13 15:02:11 节点2开始进行选举 当前term:6
2025/07/13 15:02:11 当前节点:node(4),term(7) 请求节点:node(2),term(7) 选举结果true
2025/07/13 15:02:11 当前节点:node(0),term(7) 请求节点:node(2),term(7) 选举结果true
2025/07/13 15:02:11 当前节点:node(3),term(7) 请求节点:node(2),term(7) 选举结果true
2025/07/13 15:02:11 当前节点:node(1),term(7) 请求节点:node(2),term(7) 选举结果true
2025/07/13 15:02:11 节点2向节点0发起投票 结果是true
2025/07/13 15:02:11 节点2向节点3发起投票 结果是true
2025/07/13 15:02:11 Election success 节点2选举成功 当前term:7 lastLog[idx:4,term:5]
2025/07/13 15:02:11 节点2向节点4发起投票 结果是true
2025/07/13 15:02:11 节点2向节点1发起投票 结果是true
2025/07/13 15:02:11 leader:2 follower:4 
2025/07/13 15:02:11 leader:2 follower:1 
2025/07/13 15:02:11 AE Conflict:当前节点:4 当前leader:2 冲突点4 重定位点[idx:2, term:1]
2025/07/13 15:02:11 leader:2 follower:3 
2025/07/13 15:02:11 leader:2 follower:0 
2025/07/13 15:02:11 节点4 log rep 
2025/07/13 15:02:11 AE Conflict:当前节点:0 当前leader:2 冲突点4 重定位点[idx:2, term:1]
2025/07/13 15:02:11 AE Access:当前节点4 当前leader:2 提交点4 日志长度4
2025/07/13 15:02:11 Log Apply 当前节点:4 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:11 Log Apply 当前节点:4 applyLogTerm:1 applyLogIdx:2 
2025/07/13 15:02:11 Log Apply 当前节点:4 applyLogTerm:5 applyLogIdx:3 
2025/07/13 15:02:11 Log Apply 当前节点:4 applyLogTerm:5 applyLogIdx:4 
2025/07/13 15:02:11 节点0 log rep 
2025/07/13 15:02:11 AE Access:当前节点0 当前leader:2 提交点4 日志长度4
2025/07/13 15:02:11 Log Apply 当前节点:0 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:11 Log Apply 当前节点:0 applyLogTerm:1 applyLogIdx:2 
2025/07/13 15:02:11 Log Apply 当前节点:0 applyLogTerm:5 applyLogIdx:3 
2025/07/13 15:02:11 Log Apply 当前节点:0 applyLogTerm:5 applyLogIdx:4 
2025/07/13 15:02:11 leader:2 follower:4 
2025/07/13 15:02:11 leader:2 follower:3 
2025/07/13 15:02:11 leader:2 follower:1 
2025/07/13 15:02:11 节点4 log rep 
2025/07/13 15:02:11 AE Access:当前节点4 当前leader:2 提交点4 日志长度5
2025/07/13 15:02:11 节点3 log rep 
2025/07/13 15:02:11 AE Access:当前节点3 当前leader:2 提交点4 日志长度5
2025/07/13 15:02:11 leader:2 follower:0 
2025/07/13 15:02:11 节点1 log rep 
2025/07/13 15:02:11 AE Access:当前节点1 当前leader:2 提交点4 日志长度5
2025/07/13 15:02:11 Log Commit 当前leader:2 commitIdx:5
2025/07/13 15:02:11 Log Apply 当前节点:2 applyLogTerm:7 applyLogIdx:5 
2025/07/13 15:02:11 节点0 log rep 
2025/07/13 15:02:11 AE Access:当前节点0 当前leader:2 提交点5 日志长度5
2025/07/13 15:02:11 Log Apply 当前节点:0 applyLogTerm:7 applyLogIdx:5 
2025/07/13 15:02:12 leader:2 follower:4 
2025/07/13 15:02:12 leader:2 follower:1 
2025/07/13 15:02:12 leader:2 follower:3 
2025/07/13 15:02:12 Log Apply 当前节点:4 applyLogTerm:7 applyLogIdx:5 
2025/07/13 15:02:12 leader:2 follower:0 
2025/07/13 15:02:12 Log Apply 当前节点:1 applyLogTerm:7 applyLogIdx:5 
2025/07/13 15:02:12 Log Apply 当前节点:3 applyLogTerm:7 applyLogIdx:5 
2025/07/13 15:02:12 leader:2 follower:4 
2025/07/13 15:02:12 leader:2 follower:1 
2025/07/13 15:02:12 leader:2 follower:3 
2025/07/13 15:02:12 leader:2 follower:0 
2025/07/13 15:02:12 leader:2 follower:4 
2025/07/13 15:02:12 leader:2 follower:1 
2025/07/13 15:02:12 leader:2 follower:0 
2025/07/13 15:02:12 leader:2 follower:3 
2025/07/13 15:02:12 leader:2 follower:4 
2025/07/13 15:02:12 leader:2 follower:1 
2025/07/13 15:02:12 leader:2 follower:0 
2025/07/13 15:02:12 leader:2 follower:3 
2025/07/13 15:02:12 leader:2 follower:4 
2025/07/13 15:02:12 leader:2 follower:1 
2025/07/13 15:02:12 leader:2 follower:0 
2025/07/13 15:02:12 leader:2 follower:3 
2025/07/13 15:02:12 ShutDown Node [3, 4]
2025/07/13 15:02:12 leader:2 follower:4 
2025/07/13 15:02:12 leader:2 follower:3 
2025/07/13 15:02:12 leader:2 follower:0 
2025/07/13 15:02:12 leader:2 follower:1 
2025/07/13 15:02:12 节点0 log rep 
2025/07/13 15:02:12 AE Access:当前节点0 当前leader:2 提交点5 日志长度6
2025/07/13 15:02:12 节点1 log rep 
2025/07/13 15:02:12 AE Access:当前节点1 当前leader:2 提交点5 日志长度6
2025/07/13 15:02:12 Log Commit 当前leader:2 commitIdx:6
2025/07/13 15:02:12 Log Apply 当前节点:2 applyLogTerm:7 applyLogIdx:6 
2025/07/13 15:02:12 leader:2 follower:1 
2025/07/13 15:02:12 leader:2 follower:0 
2025/07/13 15:02:12 Log Apply 当前节点:1 applyLogTerm:7 applyLogIdx:6 
2025/07/13 15:02:12 Log Apply 当前节点:0 applyLogTerm:7 applyLogIdx:6 
2025/07/13 15:02:12 ShutDown Node [2, 0, 1]
2025/07/13 15:02:12 Recovery curNode:3 rf.log[[{0 <nil>} {1 11} {1 12} {5 13} {5 14} {7 14}]]
2025/07/13 15:02:12 Recovery curNode:4 rf.log[[{0 <nil>} {1 11} {1 12} {5 13} {5 14} {7 14}]]
2025/07/13 15:02:12 Restart Node [3, 4]
2025/07/13 15:02:12 leader:2 follower:1 
2025/07/13 15:02:12 leader:2 follower:0 
2025/07/13 15:02:13 节点4开始进行选举 当前term:7
2025/07/13 15:02:13 当前节点:node(3),term(8) 请求节点:node(4),term(8) 选举结果true
2025/07/13 15:02:13 节点4向节点3发起投票 结果是true
2025/07/13 15:02:13 Recovery curNode:0 rf.log[[{0 <nil>} {1 11} {1 12} {5 13} {5 14} {7 14} {7 15}]]
2025/07/13 15:02:13 Restart Node [0]
2025/07/13 15:02:13 节点4开始进行选举 当前term:8
2025/07/13 15:02:13 当前节点:node(3),term(9) 请求节点:node(4),term(9) 选举结果true
2025/07/13 15:02:13 当前节点:node(0),term(9) 请求节点:node(4),term(9) 选举结果false
2025/07/13 15:02:13 节点4向节点3发起投票 结果是true
2025/07/13 15:02:13 节点4向节点0发起投票 结果是false
2025/07/13 15:02:14 节点0开始进行选举 当前term:9
2025/07/13 15:02:14 当前节点:node(4),term(10) 请求节点:node(0),term(10) 选举结果true
2025/07/13 15:02:14 节点0向节点4发起投票 结果是true
2025/07/13 15:02:14 当前节点:node(3),term(10) 请求节点:node(0),term(10) 选举结果true
2025/07/13 15:02:14 节点0向节点3发起投票 结果是true
2025/07/13 15:02:14 Election success 节点0选举成功 当前term:10 lastLog[idx:6,term:7]
2025/07/13 15:02:14 leader:0 follower:4 
2025/07/13 15:02:14 leader:0 follower:2 
2025/07/13 15:02:14 leader:0 follower:1 
2025/07/13 15:02:14 leader:0 follower:3 
2025/07/13 15:02:14 AE Conflict:当前节点:4 当前leader:0 冲突点6 重定位点[idx:5, term:7]
2025/07/13 15:02:14 AE Conflict:当前节点:3 当前leader:0 冲突点6 重定位点[idx:5, term:7]
2025/07/13 15:02:14 节点4 log rep 
2025/07/13 15:02:14 AE Access:当前节点4 当前leader:0 提交点0 日志长度6
2025/07/13 15:02:14 节点3 log rep 
2025/07/13 15:02:14 AE Access:当前节点3 当前leader:0 提交点0 日志长度6
2025/07/13 15:02:14 leader:0 follower:4 
2025/07/13 15:02:14 leader:0 follower:3 
2025/07/13 15:02:14 节点3 log rep 
2025/07/13 15:02:14 节点4 log rep 
2025/07/13 15:02:14 AE Access:当前节点3 当前leader:0 提交点0 日志长度7
2025/07/13 15:02:14 AE Access:当前节点4 当前leader:0 提交点0 日志长度7
2025/07/13 15:02:14 Log Commit 当前leader:0 commitIdx:7
2025/07/13 15:02:14 Log Apply 当前节点:0 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:14 Log Apply 当前节点:0 applyLogTerm:1 applyLogIdx:2 
2025/07/13 15:02:14 Log Apply 当前节点:0 applyLogTerm:5 applyLogIdx:3 
2025/07/13 15:02:14 Log Apply 当前节点:0 applyLogTerm:5 applyLogIdx:4 
2025/07/13 15:02:14 Log Apply 当前节点:0 applyLogTerm:7 applyLogIdx:5 
2025/07/13 15:02:14 Log Apply 当前节点:0 applyLogTerm:7 applyLogIdx:6 
2025/07/13 15:02:14 Log Apply 当前节点:0 applyLogTerm:10 applyLogIdx:7 
2025/07/13 15:02:14 leader:0 follower:4 
2025/07/13 15:02:14 leader:0 follower:3 
2025/07/13 15:02:14 Log Apply 当前节点:3 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:14 Log Apply 当前节点:3 applyLogTerm:1 applyLogIdx:2 
2025/07/13 15:02:14 Log Apply 当前节点:3 applyLogTerm:5 applyLogIdx:3 
2025/07/13 15:02:14 Log Apply 当前节点:3 applyLogTerm:5 applyLogIdx:4 
2025/07/13 15:02:14 Log Apply 当前节点:3 applyLogTerm:7 applyLogIdx:5 
2025/07/13 15:02:14 Log Apply 当前节点:3 applyLogTerm:7 applyLogIdx:6 
2025/07/13 15:02:14 Log Apply 当前节点:3 applyLogTerm:10 applyLogIdx:7 
2025/07/13 15:02:14 Log Apply 当前节点:4 applyLogTerm:1 applyLogIdx:1 
2025/07/13 15:02:14 Log Apply 当前节点:4 applyLogTerm:1 applyLogIdx:2 
2025/07/13 15:02:14 Log Apply 当前节点:4 applyLogTerm:5 applyLogIdx:3 
2025/07/13 15:02:14 Log Apply 当前节点:4 applyLogTerm:5 applyLogIdx:4 
2025/07/13 15:02:14 Log Apply 当前节点:4 applyLogTerm:7 applyLogIdx:5 
2025/07/13 15:02:14 Log Apply 当前节点:4 applyLogTerm:7 applyLogIdx:6 
2025/07/13 15:02:14 Log Apply 当前节点:4 applyLogTerm:10 applyLogIdx:7 
2025/07/13 15:02:14 Recovery curNode:1 rf.log[[{0 <nil>} {1 11} {1 12} {5 13} {5 14} {7 14} {7 15}]]
2025/07/13 15:02:14 Recovery curNode:2 rf.log[[{0 <nil>} {1 11} {1 12} {5 13} {5 14} {7 14} {7 15}]]
2025/07/13 15:02:14 Restart Node [1, 2]
2025/07/13 15:02:14 Seg -----------------------------------------------------------------