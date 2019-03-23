package datastructs

import "net/http"

type Response struct {
	response *http.Response
	depth    uint32
}

func (r *Response) Valid() bool {
	return r.response != nil && r.response.Body != nil
}

func NewResponse(rep *http.Response, depth uint32) *Response {
	return &Response{response: rep, depth: depth}
}

func (r *Response) Response() *http.Response {
	return r.response
}

func (r *Response) Depth() uint32 {
	return r.depth
}
