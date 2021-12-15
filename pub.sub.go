package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type PubSubChs struct {
	chs map[string]chan string
	*sync.RWMutex
}

type SubState struct {
	subName string
	status  bool
}

var stateCh = make(chan SubState, 10)

func (psc *PubSubChs) get(key string) chan string {
	psc.RLock()
	defer psc.RUnlock()
	return psc.chs[key]
}

func (psc *PubSubChs) set(key string, val chan string) {
	psc.Lock()
	defer psc.Unlock()
	psc.chs[key] = val
}

func (psc *PubSubChs) del(key string) int {
	psc.Lock()
	defer psc.Unlock()
	delete(psc.chs, key)
	return len(psc.chs)
}

func pub(ctx context.Context, psc *PubSubChs) {

	for {
		select {
		case <-ctx.Done():
			fmt.Println("发布端已退出.")
			return
		default:
			// 向所有subs广播消息
			fmt.Printf("发布端开始广播消息, 当前订阅数: %d\n", len(psc.chs))
			for subName, subCh := range psc.chs {
				subCh <- fmt.Sprintf("Hello Sub: %s, nice to meet you.", subName)
			}
			time.Sleep(8 * time.Second)
		}
	}

}

func sub(ctx context.Context, psc *PubSubChs) {
	// 首先要告知pub端, 有sub端上线
	subName := fmt.Sprintf("sub-%d", time.Now().UnixNano())
	psc.set(subName, make(chan string, 10))
	fmt.Printf("订阅端: %s 已上线.\n", subName)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("订阅端: %s 已下线.\n", subName)
			return
		case msg, ok := <-psc.get(subName):
			if ok {
				fmt.Printf("订阅端接收消息: %s\n", msg)
			}
		default:
		}
	}

}

func testSub(exit <-chan struct{}, psc *PubSubChs) {
	// 首先要告知pub端, 有sub端上线
	subName := fmt.Sprintf("sub-%d", time.Now().UnixNano())
	psc.set(subName, make(chan string, 10))
	fmt.Printf("订阅端: %s 已上线.\n", subName)

	for {
		select {
		case <-exit:
			psc.del(subName)
			fmt.Printf("订阅端: %s 已下线.\n", subName)
			return
		case msg, ok := <-psc.get(subName):
			if ok {
				fmt.Printf("订阅端接收消息: %s\n", msg)
			}
		default:
		}
	}

}

func main() {
	after := time.After(25 * time.Second)
	ctx, cancel := context.WithCancel(context.Background())

	// 初始化 pub/sub 的 channel
	pubSubChs := &PubSubChs{
		make(map[string]chan string),
		&sync.RWMutex{},
	}

	// 创建多个 sub 端
	for i := 0; i < 5; i++ {
		go sub(ctx, pubSubChs)
		time.Sleep(time.Second)
	}

	// 启动 pub 端
	go pub(ctx, pubSubChs)

	// 新增一个 sub 端
	exitCh := make(chan struct{}, 1)
	time.Sleep(1 * time.Second)
	fmt.Println("此时新增一个订阅端...")
	go testSub(exitCh, pubSubChs)

	// 移除一个 sub 端
	time.Sleep(12 * time.Second)
	fmt.Println("此时移除一个订阅端...")
	exitCh <- struct{}{}

	for {
		select {
		case <-after:
			cancel()
			time.Sleep(5 * time.Second)
			fmt.Printf("程序退出.\n")
			return
		default:
		}
	}
}
