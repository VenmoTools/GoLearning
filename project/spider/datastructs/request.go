package datastructs

import (
	"fmt"
	"net/http"
)

type Request struct {
	requests *http.Request
	depth    uint32
}

func (req *Request) Valid() bool {
	return req.requests != nil && req.requests.URL != nil
}

func NewRequest(r *http.Request, depth uint32) *Request {
	return &Request{requests: r, depth: depth}
}

func (req *Request) Request() *http.Request {
	return req.requests
}

func (req *Request) String() string{
	return fmt.Sprintf("Request --> URL:%s,Method: %s",req.requests.URL.String(),req.requests.Method)
}

/**
depth计算方法

下载其首次请求A 并接受相应经过分析器分析后找到新的地址并生成新的请求，B，c这两个深度为1，
b，发送请求并分析后经过分析后找到新的地址D，深度为2，一个请求的深度等于他父请求的深度递增1后的结果
*/
func (req *Request) Depth() uint32 {
	return req.depth
}

