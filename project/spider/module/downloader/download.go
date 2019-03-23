package downloader

import (
	"errors"
	"fmt"
	"helper/logs"
	"net/http"
	"spider/datastructs"
	"spider/exceptions"
	"spider/module"
)

type downloader struct {
	module.ModuleInternal
	client *http.Client
}

func (d *downloader) Download(req *datastructs.Request) (*datastructs.Response, error) {
	d.ModuleInternal.IncrHandlingNumber()
	defer d.ModuleInternal.DecrHandlingNumber()

	d.ModuleInternal.IncrCalledCount()
	if req == nil {
		return nil, exceptions.NewIllegalParameterError("nil request")
	}
	httpReq := req.Request()
	if httpReq == nil {
		return nil, errors.New("nil http request")
	}
	d.ModuleInternal.IncrAcceptCount()
	logs.Info(fmt.Sprintf("[Downloader] --> Do the request (URL %s, depth %d )\n", httpReq.URL, req.Depth()))

	httpResp, err := d.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	d.ModuleInternal.IncrCompletedCount()
	return datastructs.NewResponse(httpResp, req.Depth()), nil
}

func NewDownloader(mid module.MID, client *http.Client, score module.CalculateScore) (module.Downloader, error) {
	moduelBase, err := module.NewModuleInternal(mid, score)
	if err != nil {
		return nil, err
	}
	return &downloader{ModuleInternal: moduelBase, client: client}, nil
}

func GetDownloaders(number uint8, client *http.Client) ([]module.Downloader, error) {
	var downloaders []module.Downloader
	if number == 0 {
		return downloaders, nil
	}
	for i := uint8(0); i < number; i++ {
		mid, err := module.GenMID(module.TYPE_DOWNLOADER, module.Sn.Get(), nil)
		if err != nil {
			return downloaders, err
		}
		d, err := NewDownloader(mid, client, module.CalculateScoreSimple)
		if err != nil {
			return downloaders, err
		}
		downloaders = append(downloaders, d)
	}
	return downloaders, nil
}
