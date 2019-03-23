package project

import (
	"bytes"
	"down/core"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/djimenez/iconv-go"
	"strconv"
	"strings"
)

func ParserJob(content []byte) core.Response {
	doc, err := goquery.NewDocumentFromReader(bytes.NewBuffer(content))
	if err != nil {
		fmt.Println(err)
	}
	res := core.NewRequestResult()
	doc.Find("div").Each(func(i int, selection *goquery.Selection) {
		if selection.HasClass("job_msg") {
			str := selection.Text()
			c, err := iconv.ConvertString(str, "gb2312", "utf-8")
			if err != nil {
				fmt.Println(err)
			}
			if c == "" {
				return
			}
			res.AppendItem(strings.TrimSpace(c))
		}
	})
	return res
}

func ParserCompany(content []byte) core.Response {
	res := core.NewRequestResult()
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		fmt.Println(err)
	}
	doc.Find("a").Each(func(i int, selection *goquery.Selection) {
		href, ok := selection.Attr("href")
		if !ok || strings.HasPrefix(href, "//") || strings.Contains(href, "javascript") ||
			strings.HasPrefix(href, "#") || !strings.Contains(href, "jobs") {
			return
		}
		res.AppendRequest(core.NewGetRequest(href, ParserJob))
	})


	return res
}

func GetUrl(i int) string {
	url := "https://search.51job.com/list/020000,000000,0000,00,9,99,golang,2," + strconv.Itoa(i)
	return url + ".html?lang=c&stype=&postchannel=0000&workyear=99&cotype=99&degreefrom=99&jobterm=99&companysize=99&providesalary=99&lonlat=0%2C0&radius=-1&ord_field=0&confirmdate=9&fromType=&dibiaoid=0&address=&line=&specialarea=00&from=&welfare="
}
