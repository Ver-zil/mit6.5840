package lock

import (
	"log"
	"time"

	"6.5840/kvsrv1/rpc"
	"6.5840/kvtest1"
)

type Lock struct {
	// IKVClerk is a go interface for k/v clerks: the interface hides
	// the specific Clerk type of ck but promises that ck supports
	// Put and Get.  The tester passes the clerk in when calling
	// MakeLock().
	ck kvtest.IKVClerk
	// You may add code here
	id      string // 通过kvtest里的RandValue生成，给每把锁一个唯一标识
	lockKey string // 抢锁的key键
}

// todo:当前lab的节点不会crash，所以可以不考虑lease的问题，但是可以作为一个扩展点进行思考
// The tester calls MakeLock() and passes in a k/v clerk; your code can
// perform a Put or Get by calling lk.ck.Put() or lk.ck.Get().
//
// Use l as the key to store the "lock state" (you would have to decide
// precisely what the lock state is).
func MakeLock(ck kvtest.IKVClerk, l string) *Lock {
	lk := &Lock{ck: ck}
	// You may add code here
	// note:根据代码来看，每一个lock都是唯一给一个clinet
	// 通过RandValue生成一个id，送到server里进行注册，直到注册成功为止
	id := kvtest.RandValue(8)
	// todo:理论上在高并发的环境下，不同的节点仍然可能生成相同的id，虽然概率很低，应该将其能否成功加入map中作为判断id是否生成成功
	// for ck.Put(id, "", 0) != rpc.OK {
	// 	log.Printf("key:%v 已被注册\n", id)
	// 	id = kvtest.RandValue(8)
	// }
	lk.id = id
	lk.lockKey = l
	return lk
}

func (lk *Lock) Acquire() {
	// Your code here
	// todo:理论上没抢到锁应该设计成一种阻塞式状态，我们这采用time.sleep简单模拟

	for {
		value, version, err := lk.ck.Get(lk.lockKey)

		// 如果不存在这个key或者key存在并且对应的value=""则进行上锁，否则等待一段时间
		if err == rpc.ErrNoKey || err == rpc.OK && value == "" {
			// 根据lockKey进行抢锁，失败等待一会，成功则退出
			err = lk.ck.Put(lk.lockKey, lk.id, version)

			if err == rpc.OK {
				log.Printf("")
				return
			}

		} else if err == rpc.OK && value == lk.id {
			// todo:这里并未实现可重入的设计，但是如果锁是自己的，理应算抢锁成功
			log.Printf("锁是自己的 lock id:%v \n", lk.id)
			return
		}

		// 进行抢锁失败的逻辑
		log.Printf("抢锁失败，进行等待 lock id:%v \n", lk.id)
		time.Sleep(100 * time.Millisecond)
	}
}

func (lk *Lock) Release() {
	// Your code here
	// 首先先获取当前锁的状态，如果是自己的，则修改锁状态成为""，如果不是则pass

	value, version, err := lk.ck.Get(lk.lockKey)

	if err == rpc.OK && value == lk.id {
		for lk.ck.Put(lk.lockKey, "", version) != rpc.OK {
		}
	}

}
