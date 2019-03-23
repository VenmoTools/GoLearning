package module

import (
	"fmt"
	"spider/exceptions"
	"sync/atomic"
)

type Module interface {
	Id() MID                         // 获取当前组建的id
	Addr() string                    // 获取当前组建的网络地址的字符串形式
	Score() uint64                   // 当前评分
	SetScore(score uint64)           // 设置积分
	ScoreCalculator() CalculateScore // 获取评分计算器
	CallCount() uint64               // 获取当前组建被调用的计数
	AcceptCount() uint64             // 获取当前组建接受的调用计数，组建一般由于超负荷或参数有误调用的计数
	CompletedCount() uint64          // 获取当前组建已经完成的调用计数
	HandlingNumber() uint64          // 获取当前组建正在处理的调用的数量
	Counts() Counts                  // 一次性获取所有计数
	Summary() SummaryStruct          // 获取组建摘要
}

type Register interface {
	Register(module Module) (bool, error)                 // 用于注册组件实例
	Unregister(module MID) (bool, error)                  // 用于注销组建
	Get(moduleType Type) (Module, error)                  // 用于获取一个指定类型的组建，基于负载均衡策略返回
	GetAllByType(moduleType Type) (map[MID]Module, error) // 获取追定类型的所有组建
	GetAll() map[MID]Module                               // 获取所有组建
	Clear()                                               // 清除所有注销的记录
}

type SummaryStruct struct {
	Id        MID         `json:"id"`
	Called    uint64      `json:"called"`
	Accepted  uint64      `json:"accepted"`
	Completed uint64      `json:"completed"`
	Handling  uint64      `json:"handling"`
	Extra     interface{} `json:"extra,omitempty"` // 填写额外的组建信息
}

type ModuleInternal interface {
	Module
	IncrCalledCount()
	IncrAcceptCount()
	IncrCompletedCount()
	IncrHandlingNumber()
	DecrHandlingNumber()
	Clear()
}

type module struct {
	mid             MID
	addr            string
	score           uint64
	scoreCalculator CalculateScore
	calledCount     uint64
	acceptCount     uint64
	completedCount  uint64
	handlingNumber  uint64 // 实时处理的数据
}

func (m *module) Id() MID {
	return m.mid
}

func (m *module) Addr() string {
	return m.addr
}

func (m *module) Score() uint64 {
	return atomic.LoadUint64(&m.score)
}

func (m *module) SetScore(score uint64) {
	atomic.StoreUint64(&m.score, score)
}

func (m *module) ScoreCalculator() CalculateScore {
	return m.scoreCalculator
}

func (m *module) CallCount() uint64 {
	return atomic.LoadUint64(&m.calledCount)
}

func (m *module) AcceptCount() uint64 {
	return atomic.LoadUint64(&m.acceptCount)
}

func (m *module) CompletedCount() uint64 {
	return atomic.LoadUint64(&m.completedCount)
}

func (m *module) HandlingNumber() uint64 {
	return atomic.LoadUint64(&m.handlingNumber)
}

func (m *module) Counts() Counts {
	return Counts{
		CalledCount:    atomic.LoadUint64(&m.calledCount),
		AcceptedCount:  atomic.LoadUint64(&m.acceptCount),
		CompletedCount: atomic.LoadUint64(&m.completedCount),
		HandlingNumber: atomic.LoadUint64(&m.handlingNumber),
	}
}

func (m *module) Summary() SummaryStruct {
	counts := m.Counts()
	return SummaryStruct{
		Id:        m.mid,
		Called:    counts.CalledCount,
		Accepted:  counts.AcceptedCount,
		Completed: counts.CompletedCount,
		Handling:  counts.HandlingNumber,
		Extra:     nil,
	}
}

func (m *module) IncrCalledCount() {
	atomic.AddUint64(&m.calledCount, 1)
}

func (m *module) IncrAcceptCount() {
	atomic.AddUint64(&m.acceptCount, 1)
}

func (m *module) IncrCompletedCount() {
	atomic.AddUint64(&m.completedCount, 1)
}

func (m *module) IncrHandlingNumber() {
	atomic.AddUint64(&m.handlingNumber, 1)
}

func (m *module) DecrHandlingNumber() {
	atomic.AddUint64(&m.handlingNumber, 1)
}

func (m *module) Clear() {
	atomic.StoreUint64(&m.handlingNumber, 0)
	atomic.StoreUint64(&m.completedCount, 0)
	atomic.StoreUint64(&m.acceptCount, 0)
	atomic.StoreUint64(&m.calledCount, 0)
}

func NewModuleInternal(mid MID, score CalculateScore) (ModuleInternal, error) {
	parts, err := SplitMid(mid)
	if err != nil {
		return nil, exceptions.NewIllegalParameterError(fmt.Sprintf("illegal ID %q:%s", mid, err))
	}
	return &module{
		mid:             mid,
		addr:            string(parts[2]),
		scoreCalculator: score,
	}, nil
}
