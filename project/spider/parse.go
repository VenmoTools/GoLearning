package spider

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"net/url"
	"path"
	"spider/client"
	"spider/datastructs"
	"spider/module"
	"strings"
)

func parse(response *http.Response, depth uint32) ([]datastructs.Data, []error) {
	dataList := make([]datastructs.Data, 0)
	// 检查响应。
	if response == nil {
		return nil, []error{fmt.Errorf("nil HTTP response")}
	}
	httpReq := response.Request
	if httpReq == nil {
		return nil, []error{fmt.Errorf("nil HTTP request")}
	}
	reqURL := httpReq.URL
	if response.StatusCode != 200 {
		err := fmt.Errorf("unsupported status code %d (requestURL: %s)",
			response.StatusCode, reqURL)
		return nil, []error{err}
	}
	body := response.Body
	if body == nil {
		err := fmt.Errorf("nil HTTP response body (requestURL: %s)",
			reqURL)
		return nil, []error{err}
	}
	var matchedContentType bool
	if response.Header != nil {
		contentTypes := response.Header["Content-Type"]
		for _, ct := range contentTypes {
			if strings.HasPrefix(ct, "text/html") {
				matchedContentType = true
				break
			}
		}
	}

	if !matchedContentType {
		return dataList, nil
	}

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return dataList, []error{err}
	}
	errs := make([]error, 0)
	// 查找a标签并提取链接地址。
	doc.Find("a").Each(func(index int, sel *goquery.Selection) {
		href, exists := sel.Attr("href")
		// 前期过滤。
		if !exists || href == "" || href == "#" || href == "/" {
			return
		}
		href = strings.TrimSpace(href)
		lowerHref := strings.ToLower(href)
		if href == "" || strings.HasPrefix(lowerHref, "javascript") {
			return
		}
		aURL, err := url.Parse(href)
		if err != nil {
			log.Printf("An error occurs when parsing attribute %q in tag %q : %s (href: %s)", err, "href", "a", href)
			return
		}
		if !aURL.IsAbs() {
			aURL = reqURL.ResolveReference(aURL)
		}
		httpReq, err := http.NewRequest("GET", aURL.String(), nil)
		if err != nil {
			errs = append(errs, err)
		} else {
			req := datastructs.NewRequest(httpReq, depth)
			dataList = append(dataList, req)
		}
	})

	// 查找img标签并提取地址。
	doc.Find("img").Each(func(index int, sel *goquery.Selection) {
		// 前期过滤。
		imgSrc, exists := sel.Attr("src")
		if !exists || imgSrc == "" || imgSrc == "#" || imgSrc == "/" {
			return
		}
		imgSrc = strings.TrimSpace(imgSrc)
		imgURL, err := url.Parse(imgSrc)
		if err != nil {
			errs = append(errs, err)
			return
		}
		if !imgURL.IsAbs() {
			imgURL = reqURL.ResolveReference(imgURL)
		}
		httpReq, err := http.NewRequest("GET", imgURL.String(), nil)
		if err != nil {
			errs = append(errs, err)
		} else {
			req := datastructs.NewRequest(httpReq, depth)
			dataList = append(dataList, req)
		}
	})
	return dataList, errs
}



func img(httpResp *http.Response, respDepth uint32) ([]datastructs.Data, []error) {
	// 检查响应。
	if errs := client.ResponseCheck(httpResp); errs != nil {
		return nil, errs
	}
	dataList,pictureFormat := client.CheckContentType(httpResp,"image")

	item := datastructs.Item{}
	// 生成条目。
	item["reader"] = httpResp.Body
	item["name"] = path.Base(httpResp.Request.URL.Path)
	item["ext"] = pictureFormat
	dataList = append(dataList, &item)
	return dataList, nil
}

func GenResponseParsers() []module.ParserResponse {
	return []module.ParserResponse{parse, img}
}
