package buffer

import (
	"fmt"
	"helper/logs"
	"spider/exceptions"
	"sync"
	"sync/atomic"
)

// FIFO的缓冲器类型
type Buffer interface {
	Cap() uint32
	Len() uint32
	Put(data interface{}) (bool, error)
	Get() (interface{}, error)
	Close() bool
	Closed() bool
}


type fifoBuffer struct {
	ch          chan interface{}
	closed      uint32
	closingLock sync.RWMutex
}

func (buf *fifoBuffer) Cap() uint32 {
	return uint32(cap(buf.ch))
}

func (buf *fifoBuffer) Len() uint32 {
	return uint32(len(buf.ch))
}

func (buf *fifoBuffer) Put(data interface{}) (ok bool, err error) {
	buf.closingLock.RLock()
	defer buf.closingLock.RUnlock()

	if buf.Closed() {
		logs.Debug("[Buffer] --> buffer is closed")
		return false, ErrClosedBuffer
	}
	select {
	case buf.ch <- data:
		ok = true
	default:
		ok = false
	}
	return
}

func (buf *fifoBuffer) Get() (interface{}, error) {
	select {
	case data, ok := <-buf.ch:
		if !ok {
			return nil, ErrClosedBuffer
		}
		return data, nil
	default:
		return nil, nil
	}
}

func (buf *fifoBuffer) Close() bool {
	if atomic.CompareAndSwapUint32(&buf.closed, OPEN, CLOSE) {
		buf.closingLock.Lock()
		close(buf.ch)
		buf.closingLock.Unlock()
		return true
	}
	return false
}

func (buf *fifoBuffer) Closed() bool {
	return atomic.LoadUint32(&buf.closed) == CLOSE
}

func NewBuffer(size uint32) (Buffer, error) {
	if size == 0 {
		err := fmt.Sprintf("illegal size for buffer %d", size)
		return nil, exceptions.NewIllegalParameterError(err)
	}
	return &fifoBuffer{ch: make(chan interface{}, size)}, nil
}