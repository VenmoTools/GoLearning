# 通道版爬虫

## 使用

使用时需要编写内容解析函数，解析函数的内容如下

```go
func (content []byte) core.Response
```

如果需要提取所有的URL然后把URL中内容再次提取可以使用如下方法

```go
// URL解析函数
func ParserIndex(context []byte) core.Response{
    res := core.NewRequestResult()
    // 数据转码
    doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		fmt.Println(err)
    }
    // 提取出所有的a标签中的href属性中内容
	doc.Find("a").Each(func(i int, selection *goquery.Selection) {
		href, ok := selection.Attr("href")
		if !ok || strings.HasPrefix(href, "//") || strings.Contains(href, "javascript") ||
			strings.HasPrefix(href, "#") || !strings.Contains(href, "jobs") {
			return
        }
        // 使用AppendRequest添加新的请求，ParserURL函数用于解析提取出URL中的内容
		res.AppendRequest(core.NewGetRequest(href, ParserURL))
	})
}

func ParserURL(content []byte) core.Response {
    // 转码
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
            // 使用AppendItem存储提取出来的item
			res.AppendItem(strings.TrimSpace(c))
		}
	})
	return res
}
```


##### main.go
```go
func main() {
	req := core.NewGetRequest("http://xxxx.com", project.ParserIndex)
	eng := core.NewEngine()
	eng.Run(req)
}
```

engine默认使用10个worker

