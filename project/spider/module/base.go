package module

import (
	"net/http"
	"spider/datastructs"
	"sync"
)

var Sn SNGenertor

func init(){
	onc := sync.Once{}
	onc.Do(func() {
		Sn = NewSnGenertor(1, 0)
	})
}

type ParserResponse func(response *http.Response, resDepth uint32) ([]datastructs.Data, []error)


type Analyzer interface {
	Module
	RespParsers() []ParserResponse
	Analyze(resp *datastructs.Response) ([]datastructs.Data, []error)
}

type Downloader interface {
	Module
	Download(req *datastructs.Request) (*datastructs.Response, error)
}

type ProcessItem func(item datastructs.Item) (datastructs.Item, error)

type Pipeline interface {
	Module
	ItemProcessors() []ProcessItem      //用于返回当前条目处理管道的使用条目处理函数的列表
	Send(item datastructs.Item) []error //send会向条目处理管道发送条目，条目需要一次经过若干处理函数的处理
	FailFast() bool                     // 表示当前条目处理管道是否是快速失败（只要在处理某个条目时在某一个步骤上出错，那条目处理管道就会忽略后续的所有步骤并报告）
	SetFailFast(fail bool)              // 设置是否快速失败
}