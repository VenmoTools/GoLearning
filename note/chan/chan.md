# channel
channel的定义在runtime/chan.go文件中
## 基本结构

```go
type hchan struct {
	qcount   uint           // total data in the queue
	dataqsiz uint           // 缓冲槽的大小
	buf      unsafe.Pointer // 缓冲槽的指针
	elemsize uint16         //数据项大小
	closed   uint32         // 是否关闭
	elemtype *_type         // 数据项类型
	sendx    uint           // 缓冲槽发送位置索引
	recvx    uint           // 缓冲槽接受位置索引
	recvq    waitq          // 接受者等待队列
	sendq    waitq          // 发送者等待队列

    //lock保护hchan中的所有字段，以及此通道上阻塞的sudog中的几个字段
    // 在保持此锁定时不要更改另一个G的状态（特别是，没有准备好G），因为这可能会因堆栈缩小而死锁
	lock mutex
}

type waitq struct {
    /*
    sudog是必要的，因为g↔同步对象关系是多对多的。一个g可以在很多等待名单上，所以一个g可能有很多次数;并且许多gs可能正在等待同一个同步对象，因此一个对象可能存在许多因素

    sudogs是从特殊的pool分配的。使用acquireSudog和releaseSudog来分配和释放它们。
    */
	first *sudog //sudog表示等待列表中的g，例如用于在频道上发送/接收
	last  *sudog
}
```

## 方法

```go


func makechan(t *chantype, size int) *hchan {
	elem := t.elem

    // 编译器检查这个但是安全的
    // 数据项不超过64kb
	if elem.size >= 1<<16 {
		throw("makechan: invalid channel element type")
    }
    // 检查数据对齐
	if hchanSize%maxAlign != 0 || elem.align > maxAlign {
		throw("makechan: bad alignment")
	}
    // 缓冲区大小检查
	if size < 0 || uintptr(size) > maxSliceCap(elem.size) || uintptr(size)*elem.size > maxAlloc-hchanSize {
		panic(plainError("makechan: size out of range"))
	}

	// 当存储在buf中的元素不包含指针时，Hchan不包含对GC感兴趣的指针。
    // buf指向相同的allocation，elemtype是持久的
	// SudoG是从他们自己的线程中引用的，因此无法收集它们。
	// TODO(dvyukov,rlh): Rethink when collector can move allocated objects.
	var c *hchan
	switch {
	case size == 0 || elem.size == 0: //检查数据项大小
        // 队列或元素大小为零
        // 分配一块内存
		c = (*hchan)(mallocgc(hchanSize, nil, true))
		// 竞争检测使用此位置进行同步
		c.buf = unsafe.Pointer(c)
	case elem.kind&kindNoPointers != 0:
		// 元素不包含指针
		// 在一次调用中分配hchan和buf.
		c = (*hchan)(mallocgc(hchanSize+uintptr(size)*elem.size, nil, true))
		c.buf = add(unsafe.Pointer(c), hchanSize)
	default:
		// 元素包含指针
		c = new(hchan)
		c.buf = mallocgc(uintptr(size)*elem.size, elem, true)
	}
    // 设置属性
	c.elemsize = uint16(elem.size)
	c.elemtype = elem
	c.dataqsiz = uint(size)

	if debugChan {
		print("makechan: chan=", c, "; elemsize=", elem.size, "; elemalg=", elem.alg, "; dataqsiz=", size, "\n")
	}
	return c
}

```

## 收发
必须对channel的收发双方(G)进行包装，因为携带数据项，并存储相关状态，channel还得维护发送和接受者等待队列，以及异步缓冲槽状队列索引位置
## 同步
同步模式的关键是找到匹配接受或发送方，找到则直接拷贝数据，找不到就将自身打包后放入等待队列，由另一方复制数据并唤醒

channel的作用仅是维护发送和接受队列，数据复制与channel无关，唤醒后，需要验证唤醒者身份，以此决定是否有实际的数据传递

```go
// 来自编译代码的c <- x的入口点
//go:nosplit
func chansend1(c *hchan, elem unsafe.Pointer) {
	chansend(c, elem, true, getcallerpc())
}

/*
 * 通用单通道发送/接收如果块不是nil，则协议将不会休眠，但如果无法完成则返回。
 *
 * 当睡眠中涉及的通道关闭时，睡眠可以通过g.param == nil唤醒。最简单的循环和重新运行操作;我们会看到它现在已经关闭了。
 * ep是数据项指针
 */
func chansend(c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
    // 参数检查
	if c == nil {
		if !block {
			return false
        }
        // 将当前goroutine置于等待状态并调用unlockf。如果unlockf返回false，则恢复goroutine。
		gopark(nil, nil, waitReasonChanSendNilChan, traceEvGoStop, 2)
		throw("unreachable")
	}

	if debugChan {
		print("chansend: chan=", c, "\n")
	}

	if raceenabled {
		racereadpc(unsafe.Pointer(c), callerpc, funcPC(chansend))
	}

	// 快速路径：检查失败的非阻塞操作不会获取lock
	//
	// 在察觉到通道未关闭后，我们发现通道尚未准备好发送。这些observations中的每一个都是单个字大小的读取（第一个c.closed和第二个c.recvq.first或c.qcount，取决于通道的类型）
	// 因为封闭的通道无法从“准备发送”转换为“未准备好发送”，即使在两个observations之间关闭了通道，,
	// 它们也意味着当通道尚未关闭且尚未准备好发送时两者之间的时刻。如果察觉到当前行为将发送报告并无法继续
	//
	// 如果读取在此处重新排序是可以的：如果我们观察到通道尚未准备好发送，然后观察它没有关闭，则意味着在第一次observation期间通道未关闭。
	if !block && c.closed == 0 && ((c.dataqsiz == 0 && c.recvq.first == nil) ||
		(c.dataqsiz > 0 && c.qcount == c.dataqsiz)) {
		return false
	}

	var t0 int64
	if blockprofilerate > 0 {
		t0 = cputicks()
	}

	lock(&c.lock)

	if c.closed != 0 {
		unlock(&c.lock)
		panic(plainError("send on closed channel"))
	}

	if sg := c.recvq.dequeue(); sg != nil {
		// Found a waiting receiver. We pass the value we want to send
		// directly to the receiver, bypassing the channel buffer (if any).
		send(c, sg, ep, func() { unlock(&c.lock) }, 3)
		return true
	}

	if c.qcount < c.dataqsiz {
		// 通道缓冲区中有空格。将要发送的元素排入队列
		qp := chanbuf(c, c.sendx)
		if raceenabled {
			raceacquire(qp)
			racerelease(qp)
        }
        // 将数据像发送给接受者
		typedmemmove(c.elemtype, qp, ep)
		c.sendx++
		if c.sendx == c.dataqsiz {
			c.sendx = 0
		}
		c.qcount++
		unlock(&c.lock)
		return true
	}

	if !block {
		unlock(&c.lock)
		return false
	}

	// Block on the channel. Some receiver will complete our operation for us.
	gp := getg()
	mysg := acquireSudog()
	mysg.releasetime = 0
	if t0 != 0 {
		mysg.releasetime = -1
	}
	// No stack splits between assigning elem and enqueuing mysg
	// on gp.waiting where copystack can find it.
	mysg.elem = ep
	mysg.waitlink = nil
	mysg.g = gp
	mysg.isSelect = false
	mysg.c = c
	gp.waiting = mysg
	gp.param = nil
	c.sendq.enqueue(mysg)
	goparkunlock(&c.lock, waitReasonChanSend, traceEvGoBlockSend, 3)

	// someone woke us up.
	if mysg != gp.waiting {
		throw("G waiting list is corrupted")
	}
	gp.waiting = nil
	if gp.param == nil {
		if c.closed == 0 {
			throw("chansend: spurious wakeup")
		}
		panic(plainError("send on closed channel"))
	}
	gp.param = nil
	if mysg.releasetime > 0 {
		blockevent(mysg.releasetime-t0, 2)
	}
	mysg.c = nil
	releaseSudog(mysg)
	return true
}

```

