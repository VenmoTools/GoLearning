package schedluer

import (
	"encoding/json"
	"log"
	"sort"
	"spider/module"
	"spider/tools/buffer"
)

// 表示调度器摘要
type Summary struct {
	RequestArgs        RequestArgs             `json:"request_args"`
	DataArgs           DataArgs                `json:"data_args"`
	ModuleArgs         ModuleArgs              `json:"module_args"`
	Status             string                  `json:"status"`
	Downloaders        []module.SummaryStruct  `json:"downloaders"`
	Analyzer           []module.SummaryStruct  `json:"analyzer"`
	PipeLine           []module.SummaryStruct  `json:"pipe_line"`
	RequestBufferPool  BufferPoolSummaryStruct `json:"request_buffer_pool"`
	ResponseBufferPool BufferPoolSummaryStruct `json:"response_buffer_pool"`
	ItemBufferPool     BufferPoolSummaryStruct `json:"item_buffer_pool"`
	ErrorBufferPool    BufferPoolSummaryStruct `json:"error_buffer_pool"`
	NumUrl             uint64                  `json:"num_url"`
}

type summary struct {
	requestArgs RequestArgs //表示请求相关参数
	dataArgs    DataArgs    //数据相关参数的容器实例
	moduleArgs  ModuleArgs  // 组件相关参数的容器实例
	maxDepth    uint32      //爬去最大深度
	sched       *scheduler  // 调度器
}

func (s *summary) Struct() Summary {
	register := s.sched.register
	return Summary{
		RequestArgs:        s.requestArgs,
		DataArgs:           s.dataArgs,
		ModuleArgs:         s.moduleArgs,
		Status:             GetGetStatusDescription(s.sched.Status()),
		Downloaders:        getModuleSummaries(register, module.TYPE_DOWNLOADER),
		Analyzer:           getModuleSummaries(register, module.TYPE_ANALYZER),
		PipeLine:           getModuleSummaries(register, module.TYPE_PIPLINE),
		RequestBufferPool:  getBufferPoolSummary(s.sched.reqBufferPool),
		ResponseBufferPool: getBufferPoolSummary(s.sched.respBufferPool),
		ItemBufferPool:     getBufferPoolSummary(s.sched.itemBufferPool),
		ErrorBufferPool:    getBufferPoolSummary(s.sched.errBufferPool),
		NumUrl:             s.sched.urlMap.length,
	}
}

func (s *Summary) Same(other Summary) bool {
	if !s.RequestArgs.Same(&other.RequestArgs) {
		return false
	}

	if s.DataArgs != other.DataArgs {
		return false
	}

	if !s.ModuleArgs.Same(other.ModuleArgs) {
		return false
	}

	if s.Status != other.Status {
		return false
	}

	if other.Downloaders == nil || len(other.Downloaders) != len(s.Downloaders) {
		return false
	}

	if other.Analyzer == nil || len(other.Analyzer) != len(other.Analyzer) {
		return false
	}

	if other.PipeLine == nil || len(other.PipeLine) != len(other.PipeLine) {
		return false
	}

	if other.RequestBufferPool != s.RequestBufferPool {
		return false
	}
	if other.ResponseBufferPool != s.ResponseBufferPool {
		return false
	}
	if other.ItemBufferPool != s.ItemBufferPool {
		return false
	}
	if other.ErrorBufferPool != s.ErrorBufferPool {
		return false
	}
	if other.NumUrl != s.NumUrl {
		return false
	}
	return true
}

func (s *summary) String() string {
	b, err := json.MarshalIndent(s.Struct(), "", ", ")
	if err != nil {
		log.Fatalf("An error occurs when generating scheduler summary: %s\n", err)
		return ""
	}
	return string(b)

}

// 创建调度器实例
func newSchedulerSummary(args RequestArgs, dataArgs DataArgs, moduleArgs ModuleArgs, scheduler *scheduler) SchedSummary {
	if scheduler == nil {
		return nil
	}
	return &summary{
		requestArgs: args,
		dataArgs:    dataArgs,
		moduleArgs:  moduleArgs,
		sched:       scheduler,
	}
}

type BufferPoolSummaryStruct struct {
	BufferCap       uint32 `json:"buffer_cap"`
	MaxBufferNumber uint32 `json:"max_buffer_number"`
	BufferNumber    uint32 `json:"buffer_number"`
	Total           uint64 `json:"total"`
}

func getBufferPoolSummary(pool buffer.Pool) BufferPoolSummaryStruct {
	return BufferPoolSummaryStruct{
		BufferCap:       pool.BufferCap(),
		MaxBufferNumber: pool.MaxBufferNumber(),
		BufferNumber:    pool.BufferNumber(),
		Total:           pool.Total(),
	}
}

func getModuleSummaries(register module.Register, p module.Type) []module.SummaryStruct {
	moduleMap, _ := register.GetAllByType(p)
	var ss []module.SummaryStruct

	if len(moduleMap) > 0 {
		for _, m := range moduleMap {
			ss = append(ss, m.Summary())
		}
	}
	if len(ss) > 1 {
		sort.Slice(ss, func(i, j int) bool {
			return ss[i].Id < ss[j].Id
		})
	}
	return ss
}
