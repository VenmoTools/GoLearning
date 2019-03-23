package pipeline

import (
	"fmt"
	"helper/logs"
	"spider/datastructs"
	"spider/module"
)



type pipeline struct {
	module.ModuleInternal
	itemProcess []module.ProcessItem
	failFast    bool
}

func (p *pipeline) ItemProcessors() []module.ProcessItem {
	processor := make([]module.ProcessItem, len(p.itemProcess))
	copy(processor, p.itemProcess)
	return processor
}

func (p *pipeline) Send(item datastructs.Item) (errs []error) {

	p.IncrHandlingNumber()
	defer p.DecrHandlingNumber()
	p.IncrCalledCount()

	if item == nil {
		errs = append(errs, genParameterError("nil item processor list"))
	}

	p.IncrAcceptCount()

	logs.Info(fmt.Sprintf("[PipeLine] --> Process item +%v \n", item))
	var currentItem = item
	for _, processor := range p.itemProcess {
		pItem, err := processor(currentItem)
		if err != nil {
			errs = append(errs, err)
			if p.failFast {
				break
			}
		}
		if pItem != nil {
			currentItem = pItem
		}
	}
	if len(errs) == 0 {
		p.IncrCompletedCount()
	}
	return
}

func (p *pipeline) FailFast() bool {
	return p.failFast
}

func (p *pipeline) SetFailFast(fail bool) {
	p.failFast = fail
}

type extraSummaryStruct struct {
	FailFast        bool `json:"fail_fast"`
	ProcessorNumber int  `json:"processor_number"`
}

func (p *pipeline) Summary() module.SummaryStruct {
	summary := p.ModuleInternal.Summary()
	summary.Extra = extraSummaryStruct{
		FailFast:        p.failFast,
		ProcessorNumber: len(p.itemProcess),
	}
	return summary
}

func NewPipeline(mid module.MID, score module.CalculateScore, item []module.ProcessItem) (module.Pipeline, error) {
	moduleBase, err := module.NewModuleInternal(mid, score)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return nil, genParameterError("nil item processor list")
	}

	if len(item) == 0 {
		return nil, genParameterError("empty item processor list")
	}

	var innerItem []module.ProcessItem

	for i, it := range item {
		if it == nil {
			return nil, genParameterError(fmt.Sprintf("nil item processor[%d]", i))
		}
		innerItem = append(innerItem, it)
	}

	return &pipeline{itemProcess: innerItem, ModuleInternal: moduleBase}, nil
}

// GetPipelines 用于获取条目处理管道列表。
func GetPipelines(number uint8, item []module.ProcessItem) ([]module.Pipeline, error) {
	var pipelines []module.Pipeline
	if number == 0 {
		return pipelines, nil
	}
	for i := uint8(0); i < number; i++ {
		mid, err := module.GenMID(module.TYPE_PIPLINE, module.Sn.Get(), nil)
		if err != nil {
			return pipelines, err
		}
		a, err := NewPipeline(mid, module.CalculateScoreSimple, item, )
		if err != nil {
			return pipelines, err
		}
		a.SetFailFast(true)
		pipelines = append(pipelines, a)
	}
	return pipelines, nil
}
