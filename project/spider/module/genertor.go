package module

import (
	"math"
	"sync"
)

/**
序列号生成器，
	组建类型字母：   序列号       组建地址模块
		|		     |   			|
	组建类型模块   序列号生成器    网络地址
*/
type SNGenertor interface {
	Start() uint64      // 用于获取预设的最小序列号
	Max() uint64        // 获取预设的最大序列号
	Next() uint64       // 获取下一个序列号
	CycleCount() uint64 //获取循环计数
	Get() uint64        // 获得一个序列号并准备以下个序列号
}

type genertor struct {
	start      uint64       // 生成的最小序列号
	max        uint64       // 生成的最大序列号
	next       uint64       //下一个序列号
	cycleCount uint64       //循环计数
	lock       sync.RWMutex //读写锁
}

func (g *genertor) Start() uint64 {
	return g.start
}

func (g *genertor) Max() uint64 {
	return g.max
}

func (g *genertor) Next() uint64 {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.next
}

func (g *genertor) CycleCount() uint64 {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.cycleCount
}

func (g *genertor) Get() uint64 {
	g.lock.Lock()
	defer g.lock.Unlock()
	id := g.next
	if id == g.max {
		g.next = g.start
		g.cycleCount++
	} else {
		g.next++
	}
	return id
}

func NewSnGenertor(start uint64, max uint64) SNGenertor {
	if max == 0 {
		max = math.MaxUint64
	}
	return &genertor{
		start: start,
		max:   max,
	}
}
