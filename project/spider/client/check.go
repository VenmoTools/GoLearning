package client

import (
	"fmt"
	"net/http"
	"spider/datastructs"
	"strings"
)

func ResponseCheck(httpResp *http.Response) []error {
	if httpResp == nil {
		return []error{fmt.Errorf("nil HTTP response")}
	}
	httpReq := httpResp.Request
	if httpReq == nil {
		return []error{fmt.Errorf("nil HTTP request")}
	}
	reqURL := httpReq.URL
	if httpResp.StatusCode != 200 {
		err := fmt.Errorf("unsupported status code %d (requestURL: %s)", httpResp.StatusCode, reqURL)
		return []error{err}
	}
	httpRespBody := httpResp.Body
	if httpRespBody == nil {
		err := fmt.Errorf("nil HTTP response body (requestURL: %s)", reqURL)
		return []error{err}
	}
	return nil
}

func CheckContentType(httpResp *http.Response, ty string) ([]datastructs.Data, string) {
	// 检查HTTP响应头中的内容类型。
	dataList := make([]datastructs.Data, 0)
	var pictureFormat string
	if httpResp.Header != nil {
		contentTypes := httpResp.Header["Content-Type"]
		var contentType string
		for _, ct := range contentTypes {
			if strings.HasPrefix(ct, ty) {
				contentType = ct
				break
			}
		}
		index1 := strings.Index(contentType, "/")
		index2 := strings.Index(contentType, ";")
		if index1 > 0 {
			if index2 < 0 {
				pictureFormat = contentType[index1+1:]
			} else if index1 < index2 {
				pictureFormat = contentType[index1+1 : index2]
			}
		}
	}
	return dataList, pictureFormat
}
