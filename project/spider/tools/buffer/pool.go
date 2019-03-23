package buffer

import (
	"fmt"
	"helper/logs"
	"spider/exceptions"
	"sync"
	"sync/atomic"
)

// 缓冲池
type Pool interface {
	BufferCap() uint32
	MaxBufferNumber() uint32
	BufferNumber() uint32
	Total() uint64
	Put(data interface{}) error //put用于向缓冲池放入数据, 本方法是阻塞的,如果缓冲池已关闭，返回非nil的错误
	Get() (data interface{}, err error)
	Close() bool  // 缓冲区关闭返回true
	Closed() bool //用于判断是否已经关闭
	String() string
}

/**
	双层通道设计好处
	bufCh中每一个缓冲器一次只会被一个goroutine中的程序拿到，在放回bufCh之前对其他并发程序不可见，一个缓冲器在每次只会被并发程序放入或者拿取一个数据
	bufCh是FiFo的把先拿出来的穿红漆归还给bufCh时，该缓冲器总在队尾
 */
type pool struct {
	bufferCap       uint32      //缓冲器容量
	maxBufferNumber uint32      // 缓冲器最大容量
	bufferNumber    uint32      // 缓冲器实际容量
	total           uint64      //池中数据总数
	bufCh           chan Buffer // 存放缓冲器通道,双层通道设计
	closed          uint32      // 关闭状态
	lock            sync.RWMutex
	name            string
}

func (p *pool) String() string {
	p.lock.Lock()
	defer p.lock.Unlock()
	n := p.name
	return n
}

func (p *pool) BufferCap() uint32 {
	return atomic.LoadUint32(&p.bufferCap)
}

func (p *pool) MaxBufferNumber() uint32 {
	return atomic.LoadUint32(&p.maxBufferNumber)
}

func (p *pool) BufferNumber() uint32 {
	return atomic.LoadUint32(&p.bufferNumber)
}

func (p *pool) Total() uint64 {
	return atomic.LoadUint64(&p.total)
}

func (p *pool) Put(data interface{}) (err error) {

	//log.Printf("[BufferPool] --> new data coming %#v", data)

	if p.Closed() {
		return ErrClosedBuffer
	}

	var count uint32
	maxCount := p.BufferNumber() * 5
	var ok bool
	for buf := range p.bufCh {
		ok, err = p.putData(buf, data, &count, maxCount)
		if ok || err != nil {
			//log.Printf("[BufferPool] --> put faild status %v,cause %#v", ok, err)
			break
		}
	}
	return
}

func (p *pool) putData(buffer Buffer, data interface{}, count *uint32, max uint32) (ok bool, err error) {
	//log.Printf("[%sBufferPool] --> Puting Data %#v", p.name, data)
	if p.Closed() {
		logs.Debug("[Buffer] --> BufferPool is closed")
		return false, ErrClosedBuffer
	}
	defer func() {
		p.lock.RLock()
		if p.Closed() {
			atomic.AddUint32(&p.bufferNumber, ^uint32(0))
			err = ErrClosedBuffer
		} else {
			p.bufCh <- buffer
		}
		p.lock.RUnlock()
	}()
	ok, err = buffer.Put(data)
	if ok {
		atomic.AddUint64(&p.total, 1)
		return
	}
	if err != nil {
		logs.Error("[Buffer] --> puting data failed cause %v", err)
		return
	}
	*count++

	if *count >= max && p.BufferNumber() < p.MaxBufferNumber() {
		p.lock.Lock()
		/**
			双检锁
			防止向已关闭的缓冲池追加缓冲器
			防止缓冲器的数量超过最大值
		 */
		if p.BufferNumber() < p.MaxBufferNumber() {
			if p.Closed() {
				p.lock.Unlock()
				return
			}
			newBuf, err := NewBuffer(p.bufferCap)
			if err != nil {
				logs.Error(err, "at new buffer")
			}
			_, err = newBuf.Put(data)
			if err != nil {
				logs.Error(err, "at buffer put")
			}
			p.bufCh <- newBuf
			atomic.AddUint32(&p.bufferNumber, 1)
			atomic.AddUint64(&p.total, 1)
			ok = true
		}
		p.lock.Unlock()
		*count = 0
	}
	return
}

func (p *pool) Get() (data interface{}, err error) {
	if p.Closed() {
		return nil, ErrClosedBuffer
	}
	var count uint32
	maxCount := p.BufferNumber() * 10
	for buf := range p.bufCh {
		data, err = p.getData(buf, &count, maxCount)
		if data != nil || err != nil {
			break
		}
	}
	return
}

func (p *pool) getData(buffer Buffer, count *uint32, max uint32) (data interface{}, err error) {
	if p.Closed() {
		return nil, ErrClosedPool
	}

	defer func() {
		if *count >= max && buffer.Len() == 0 && p.BufferNumber() > 1 {
			buffer.Close()
			atomic.AddUint32(&p.bufferNumber, ^uint32(0))
			*count = 0
			return
		}
		p.lock.RLock()
		if p.Closed() {
			atomic.AddUint32(&p.bufferNumber, ^uint32(0))
			err = ErrClosedPool
		} else {
			p.bufCh <- buffer
		}
		p.lock.RUnlock()
	}()

	data, err = buffer.Get()
	if data != nil {
		atomic.AddUint64(&p.total, ^uint64(0))
		return
	}
	if err != nil {
		return
	}
	*count++
	return
}

func (p *pool) Close() bool {
	if !atomic.CompareAndSwapUint32(&p.closed, OPEN, CLOSE) {
		return false
	}
	p.lock.Lock()
	defer p.lock.Unlock()

	close(p.bufCh)
	for buf := range p.bufCh {
		buf.Closed()
	}
	return true
}

func (p *pool) Closed() bool {
	return atomic.LoadUint32(&p.closed) == CLOSE
}

func NewPool(bufferCap uint32, maxBufferNumber uint32, name string) (Pool, error) {
	if bufferCap == 0 {
		return nil, exceptions.NewIllegalParameterError(fmt.Sprintf("illegal buffer cap for buffer pool: %d", bufferCap))
	}
	if maxBufferNumber == 0 {
		return nil, exceptions.NewIllegalParameterError(fmt.Sprintf("iillegal max buffer number for buffer pool: %d", maxBufferNumber))
	}

	buChan := make(chan Buffer, maxBufferNumber)
	buf, _ := NewBuffer(bufferCap)
	buChan <- buf
	return &pool{
		bufferCap:       bufferCap,
		maxBufferNumber: maxBufferNumber,
		bufferNumber:    1,
		bufCh:           buChan,
		name:            name,
	}, nil
}
