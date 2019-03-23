package analyzer

import (
	"fmt"
	"helper/logs"
	"spider/datastructs"
	"spider/module"
	"spider/tools/buffer"
)

type analyzer struct {
	module.ModuleInternal
	respParsers []module.ParserResponse
}

func (a *analyzer) RespParsers() []module.ParserResponse {
	parsers := make([]module.ParserResponse, len(a.respParsers))
	copy(parsers, a.respParsers)
	return parsers
}

func (a *analyzer) Analyze(resp *datastructs.Response) (dataList []datastructs.Data, errList []error) {
	a.ModuleInternal.IncrHandlingNumber()
	defer a.ModuleInternal.DecrHandlingNumber()

	a.ModuleInternal.IncrCalledCount()

	// 检查传入请求响应信息
	if resp == nil {
		errList = append(errList, genParameterError("nil response"))
		return
	}

	// 检查原始的http请求响应信息
	httpResp := resp.Response()
	if httpResp == nil {
		errList = append(errList, genParameterError("nil http response"))
		return
	}

	// 检查http请求信息
	var httpReq = httpResp.Request
	if httpReq == nil {
		errList = append(errList, genParameterError("nil http request"))
		return
	}

	// 检查请求url信息
	var url = httpReq.URL
	if url == nil {
		errList = append(errList, genParameterError("nil http request url"))
		return
	}
	a.ModuleInternal.IncrAcceptCount()

	respDpth := resp.Depth()

	logs.Info(fmt.Sprintf("[Analyzer] --> Parser the response (URL:%v, dpth %d)\n", url, respDpth))

	defer func() {
		if httpResp.Body != nil {
			err := httpResp.Body.Close()
			if err != nil {
				logs.Error(fmt.Sprintf("[Analyzer] --> Parser the response error %v", err))
				errList = append(errList, err)
			}
		}
	}()

	multipleReader, err := buffer.NewMultipleReader(httpResp.Body)
	if err != nil {
		logs.Error(fmt.Sprintf("[Analyzer] --> Parser the response error %v", err))
		errList = append(errList, err)
		return
	}
	dataList = []datastructs.Data{}

	for _, parser := range a.respParsers {
		// 先根据body字段初始化一个多重读取器，如何每次调用解析函数之前从多重读取器获取一个读取器，并多http响应的body字段重新赋值
		// 解决了body字段只能读取一次的问题
		httpResp.Body = multipleReader.Reader()
		pd, pe := parser(httpResp, respDpth)
		if pd != nil {
			for _, dData := range pd {
				if dData == nil {
					continue
				}
				dataList = append(dataList, dData)
			}
		}

		if pe != nil {
			for _, errs := range pe {
				if errs == nil {
					continue
				}
				errList = append(errList, errs)
			}
		}
	}
	if len(errList) == 0 {
		a.ModuleInternal.CompletedCount()
	}
	return dataList, errList
}

// ParserResponse是使用方提供属于外来代码，检验要仔细
func NewAnalyzer(mid module.MID, response []module.ParserResponse, score module.CalculateScore) (module.Analyzer, error) {
	moduleBase, err := module.NewModuleInternal(mid, score)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, genParameterError("nil response parsers")
	}

	if len(response) == 0 {
		return nil, genParameterError("empty response parser list")
	}

	var innerparser []module.ParserResponse
	for i, parser := range response {
		if parser == nil {
			return nil, genParameterError(fmt.Sprintf("nil response parser[%d]", i))
		}
		innerparser = append(innerparser, parser)
	}
	return &analyzer{respParsers: innerparser, ModuleInternal: moduleBase}, nil
}

func GetAnalyzers(number uint8, response []module.ParserResponse) ([]module.Analyzer, error) {
	var analyzers []module.Analyzer
	if number == 0 {
		return analyzers, nil
	}
	for i := uint8(0); i < number; i++ {
		mid, err := module.GenMID(
			module.TYPE_ANALYZER, module.Sn.Get(), nil)
		if err != nil {
			return analyzers, err
		}
		a, err := NewAnalyzer(mid, response, module.CalculateScoreSimple)
		if err != nil {
			return analyzers, err
		}
		analyzers = append(analyzers, a)
	}
	return analyzers, nil
}
