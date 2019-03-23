package schedluer

import "spider/module"

type Args interface {
	// 自检参数的有效性
	Check() error
}

/**
爬取范围
*/
type RequestArgs struct {
	AcceptedDomains []string `json:"accepted_primary_domains"`
	MaxDepth        uint32   `json:"max_depth"`
}

func (r *RequestArgs) Same(args *RequestArgs) bool {
	if args == nil {
		return false
	}
	if args.MaxDepth != args.MaxDepth {
		return false
	}
	anotherDomains := args.AcceptedDomains
	anotherDomainsLen := len(anotherDomains)
	if anotherDomainsLen != len(args.AcceptedDomains) {
		return false
	}
	if anotherDomainsLen > 0 {
		for i, domain := range anotherDomains {
			if domain != args.AcceptedDomains[i] {
				return false
			}
		}
	}
	for i, ar := range args.AcceptedDomains {
		if r.AcceptedDomains[i] != ar {
			return false
		}
	}
	if r.MaxDepth != args.MaxDepth {
		return false
	}
	return true
}

func (r *RequestArgs) Check() error {
	if r.AcceptedDomains == nil {
		return genError("nil accepted primary domain list")
	}
	return nil
}

type DataArgs struct {
	RequestBufferCap        uint32 `json:"request_buffer_cap"`         // 请求缓冲的容量
	RequestMaxBufferNumber  uint32 `json:"request_max_number"`         // 请求缓冲的最大容量
	ResponseBufferCao       uint32 `json:"response_buffer_cao"`        // 响应缓冲的容量
	ResponseMaxBufferNumber uint32 `json:"response_max_buffer_number"` // 响应缓冲的最大容量
	ItemBufferCap           uint32 `json:"item_buffer_cap"`            // 条目缓冲的容量
	ItemMaxBufferNumber     uint32 `json:"item_max_buffer_number"`     // 条目缓冲的最大容量
	ErrorBufferCap          uint32 `json:"error_buffer_cap"`           // 错误缓冲的容量
	ErrorMaxBufferNumber    uint32 `json:"error_max_buffer_number"`    // 错误缓冲的最大容量
}

func (args *DataArgs) Check() error {
	if args.RequestBufferCap == 0 {
		return genError("zero request buffer capacity")
	}
	if args.RequestMaxBufferNumber == 0 {
		return genError("zero max request buffer number")
	}
	if args.ResponseBufferCao == 0 {
		return genError("zero response buffer capacity")
	}
	if args.ResponseMaxBufferNumber == 0 {
		return genError("zero max response buffer number")
	}
	if args.ItemBufferCap == 0 {
		return genError("zero item buffer capacity")
	}
	if args.ItemMaxBufferNumber == 0 {
		return genError("zero max item buffer number")
	}
	if args.ErrorBufferCap == 0 {
		return genError("zero error buffer capacity")
	}
	if args.ErrorMaxBufferNumber == 0 {
		return genError("zero max error buffer number")
	}
	return nil
}

type ModuleArgs struct {
	Downloaders []module.Downloader
	Analyzer    []module.Analyzer
	Pipline     []module.Pipeline
}

func (m *ModuleArgs) Same(other ModuleArgs) bool {

	for i, d := range m.Downloaders {
		if other.Downloaders[i] != d {
			return false
		}
	}
	for i, d := range m.Analyzer {
		if other.Analyzer[i] != d {
			return false
		}
	}
	for i, d := range m.Analyzer {
		if other.Analyzer[i] != d {
			return false
		}
	}
	return true
}

func (args *ModuleArgs) Check() error {
	if len(args.Downloaders) == 0 {
		return genError("empty downloader list")
	}
	if len(args.Analyzer) == 0 {
		return genError("empty analyzer list")
	}
	if len(args.Pipline) == 0 {
		return genError("empty pipeline list")
	}
	return nil
}

// ModuleArgsSummary 代表组件相关的参数容器的摘要类型。
type ModuleArgsSummary struct {
	DownloaderListSize int `json:"downloader_list_size"`
	AnalyzerListSize   int `json:"analyzer_list_size"`
	PipelineListSize   int `json:"pipeline_list_size"`
}

func (args *ModuleArgs) Summary() ModuleArgsSummary {
	return ModuleArgsSummary{
		DownloaderListSize: len(args.Downloaders),
		AnalyzerListSize:   len(args.Analyzer),
		PipelineListSize:   len(args.Pipline),
	}
}
