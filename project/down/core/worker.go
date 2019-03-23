package core

import (
	"bufio"
	"errors"
	"fmt"
	"helper/logs"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type Work interface {
	StartWork(req *http.Request) (res []byte, err error)
}

func Worker(r Request, work Work) (Response, error) {
	res, err := work.StartWork(r.Req)
	if err != nil || r.ParserFunc == nil {
		return Response{}, err
	}
	return r.ParserFunc(res), nil
}

type ParseWork struct {
	client  *http.Client
	timeSec time.Duration
}

func NewParserWork() *ParseWork {
	p := &ParseWork{}
	p.client = GenHttpClient()
	p.timeSec = 2
	return p
}

func GenHttpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   5,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

func (t *ParseWork) StartWork(req *http.Request) (res []byte, err error) {
	if req == nil {
		return
	}
	time.After(time.Second * t.timeSec)
	logs.Info(req.URL.String())
	resp, err := t.client.Do(req)
	if err != nil {
		return
	}
	err = ResponseCheck(resp)
	if err != nil {
		return
	}
	defer func() {
		err = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("Response code not ok code<%d>", resp.StatusCode))
		return
	}
	res, err = ioutil.ReadAll(bufio.NewReader(resp.Body))

	return
}

func ResponseCheck(httpResp *http.Response) error {
	if httpResp == nil {
		return errors.New(fmt.Sprintf("nil HTTP response"))
	}
	httpReq := httpResp.Request
	if httpReq == nil {
		return errors.New(fmt.Sprintf("nil HTTP request"))
	}
	reqURL := httpReq.URL
	if httpResp.StatusCode != 200 {
		return errors.New(fmt.Sprintf("unsupported status code %d (requestURL: %s)", httpResp.StatusCode, reqURL))
	}
	httpRespBody := httpResp.Body
	if httpRespBody == nil {
		return errors.New(fmt.Sprintf("nil HTTP response body (requestURL: %s)", reqURL))
	}
	return nil
}
