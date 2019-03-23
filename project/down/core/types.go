package core

import (
	"helper/logs"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Request struct {
	Req        *http.Request
	ParserFunc func([]byte) Response
}

func NewGetRequest(url string, ParserFunc func([]byte) Response) Request {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel …) Gecko/20100101 Firefox/65.0")
	if err != nil {
		logs.Error(err)
	}
	return Request{Req: req, ParserFunc: ParserFunc}
}

func NewPostFormRequest(url string, data url.Values, ParserFunc func([]byte) Response) Request {
	return NewPostRequest(url, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded", ParserFunc)
}

func NewPostRequest(url string, reader io.Reader, contentType string, ParserFunc func([]byte) Response) Request {
	req, err := http.NewRequest("POST", url, reader)
	req.Header.Set("Content-Type", contentType)
	if err != nil {
		logs.Error(err)
	}
	return Request{Req: req, ParserFunc: ParserFunc}
}

type Response struct {
	req  []Request
	item []interface{}
}

func (r *Response) GetRequestQueue() []Request {
	return r.req
}
func (r *Response) GetItemQueue() []interface{} {
	return r.item
}

func (r *Response) SizeOfRequestQueue() int {
	return len(r.GetRequestQueue())
}

func (r *Response) SizeOfItemQueue() int {
	return len(r.GetItemQueue())
}

func (r *Response) AppendRequest(request Request) {
	if request.Req == nil {
		logs.Info("Request is nil ignore..")
		return
	}
	r.req = append(r.req, request)
}

func (r *Response) AppendItem(item interface{}) {
	if item == nil {
		logs.Info("item is nil ignore..")
		return
	}
	r.item = append(r.item, item)
}

func NewRequestResult() Response {
	return Response{}
}
