# httpLib
## httpLib结构
Beego的HttpLib提供了很方便的API，主要结构如下 
```go
type BeegoHTTPRequest struct {
	url     string //请求URL
	req     *http.Request //请求对象
	params  map[string][]string // 参数，对于GET来说参数为？后面的的键值对
	files   map[string]string //文件相关
	setting BeegoHTTPSettings // 用于设置Request相关配置
	resp    *http.Response // 响应
	body    []byte //响应/请求体
	dump    []byte
}

type BeegoHTTPSettings struct {
	ShowDebug        bool // 是否显示Debug信息
	UserAgent        string // UA配置
	ConnectTimeout   time.Duration // 链接超时时间
	ReadWriteTimeout time.Duration // 读写超时时间
	TLSClientConfig  *tls.Config // TLS相关设置
	Proxy            func(*http.Request) (*url.URL, error) // 代理函数
	Transport        http.RoundTripper  
	CheckRedirect    func(req *http.Request, via []*http.Request) error // 重定向处理函数
	EnableCookie     bool //启动cookie
	Gzip             bool // 开启gzip编码
	DumpBody         bool
	Retries          int // 重试次数，如果设置为-1表示将永远重试
}

// beego的默认配置如下
var defaultSetting = BeegoHTTPSettings{
	UserAgent:        "beegoServer",
	ConnectTimeout:   60 * time.Second,
	ReadWriteTimeout: 60 * time.Second,
	Gzip:             true,
	DumpBody:         true,
}
//beego提供了更改默认配置的函数
func SetDefaultSetting(setting BeegoHTTPSettings) {
	settingMutex.Lock()
	defer settingMutex.Unlock()
	defaultSetting = setting
}

//BeegoRequest支持链式调用
// Setting Change request settings
func (b *BeegoHTTPRequest) Setting(setting BeegoHTTPSettings) *BeegoHTTPRequest {
	b.setting = setting
	return b
}

// SetBasicAuth将请求的AuthorizationHeader设置为使用提供的用户名和密码来使用HTTP Basic Authentication。
func (b *BeegoHTTPRequest) SetBasicAuth(username, password string) *BeegoHTTPRequest {
	b.req.SetBasicAuth(username, password)
	return b
}

// SetEnableCookie 用来设置关闭/开启 cookiejar
func (b *BeegoHTTPRequest) SetEnableCookie(enable bool) *BeegoHTTPRequest {
	b.setting.EnableCookie = enable
	return b
}

// SetUserAgent 设置 User-Agent Header头字段
func (b *BeegoHTTPRequest) SetUserAgent(useragent string) *BeegoHTTPRequest {
	b.setting.UserAgent = useragent
	return b
}

// Debug在执行请求时是否显示调试
func (b *BeegoHTTPRequest) Debug(isdebug bool) *BeegoHTTPRequest {
	b.setting.ShowDebug = isdebug
	return b
}

// 重试次数设置重试次数。默认值为0表示不重试。 -1表示永远重试。其他意味着重试的次数。
func (b *BeegoHTTPRequest) Retries(times int) *BeegoHTTPRequest {
	b.setting.Retries = times
	return b
}

// DumpBody设置是否需要转储Body.
func (b *BeegoHTTPRequest) DumpBody(isdump bool) *BeegoHTTPRequest {
	b.setting.DumpBody = isdump
	return b
}


// SetTimeout 设置链接超时时间和读写超超时
func (b *BeegoHTTPRequest) SetTimeout(connectTimeout, readWriteTimeout time.Duration) *BeegoHTTPRequest {
	b.setting.ConnectTimeout = connectTimeout
	b.setting.ReadWriteTimeout = readWriteTimeout
	return b
}

// SetTLSClientConfig 如果访问https url，则设置连接配置。
func (b *BeegoHTTPRequest) SetTLSClientConfig(config *tls.Config) *BeegoHTTPRequest {
	b.setting.TLSClientConfig = config
	return b
}

// Header 添加头字段
func (b *BeegoHTTPRequest) Header(key, value string) *BeegoHTTPRequest {
	b.req.Header.Set(key, value)
	return b
}

// SetHost 请求主机
func (b *BeegoHTTPRequest) SetHost(host string) *BeegoHTTPRequest {
	b.req.Host = host
	return b
}

// SetProtocolVersion 设置发送请求的协议版本.
// 客户端请求一直使用  HTTP/1.1.
func (b *BeegoHTTPRequest) SetProtocolVersion(vers string) *BeegoHTTPRequest {
	if len(vers) == 0 {
		vers = "HTTP/1.1"
	}

	major, minor, ok := http.ParseHTTPVersion(vers)
	if ok {
		b.req.Proto = vers
		b.req.ProtoMajor = major
		b.req.ProtoMinor = minor
	}

	return b
}

// SetCookie 将cookie添加进request中
func (b *BeegoHTTPRequest) SetCookie(cookie *http.Cookie) *BeegoHTTPRequest {
	b.req.Header.Add("Cookie", cookie.String())
	return b
}

// SetTransport 设置transport
func (b *BeegoHTTPRequest) SetTransport(transport http.RoundTripper) *BeegoHTTPRequest {
	b.setting.Transport = transport
	return b
}


// SetProxy 用于设置http代理
// 例如:
//
//	func(req *http.Request) (*url.URL, error) {
// 		u, _ := url.ParseRequestURI("http://127.0.0.1:8118")
// 		return u, nil
// 	}
func (b *BeegoHTTPRequest) SetProxy(proxy func(*http.Request) (*url.URL, error)) *BeegoHTTPRequest {
	b.setting.Proxy = proxy
	return b
}

// SetCheckRedirect 指定处理重定向的策略.
//
// 如果CheckRedirect为 nil, 客户端使用默认策略
// 连续10次请求后停止
func (b *BeegoHTTPRequest) SetCheckRedirect(redirect func(req *http.Request, via []*http.Request) error) *BeegoHTTPRequest {
	b.setting.CheckRedirect = redirect
	return b
}

// Param用于添加查询参数
// 参数的格式类似于 ?key1=value1&key2=value2...
func (b *BeegoHTTPRequest) Param(key, value string) *BeegoHTTPRequest {
	if param, ok := b.params[key]; ok {
		b.params[key] = append(param, value)
	} else {
		b.params[key] = []string{value}
	}
	return b
}

// PostFile 添加上传文件，第一个参数是 form 表单的字段名,第二个是需要发送的文件名或者文件路径
func (b *BeegoHTTPRequest) PostFile(formname, filename string) *BeegoHTTPRequest {
	b.files[formname] = filename
	return b
}

// Body 添加请求原始的body 支持 string 和 []byte.
func (b *BeegoHTTPRequest) Body(data interface{}) *BeegoHTTPRequest {
	switch t := data.(type) {
	case string:
		bf := bytes.NewBufferString(t)
		b.req.Body = ioutil.NopCloser(bf)
		b.req.ContentLength = int64(len(t))
	case []byte:
		bf := bytes.NewBuffer(t)
		b.req.Body = ioutil.NopCloser(bf)
		b.req.ContentLength = int64(len(t))
	}
	return b
}

// XMLBody 通过XML添加请求原始body编码
func (b *BeegoHTTPRequest) XMLBody(obj interface{}) (*BeegoHTTPRequest, error) {
	if b.req.Body == nil && obj != nil {
		byts, err := xml.Marshal(obj)
		if err != nil {
			return b, err
		}
		b.req.Body = ioutil.NopCloser(bytes.NewReader(byts))
		b.req.ContentLength = int64(len(byts))
		b.req.Header.Set("Content-Type", "application/xml")
	}
	return b, nil
}

// YAMLBody adds request raw body encoding by YAML.
func (b *BeegoHTTPRequest) YAMLBody(obj interface{}) (*BeegoHTTPRequest, error) {
	if b.req.Body == nil && obj != nil {
		byts, err := yaml.Marshal(obj)
		if err != nil {
			return b, err
		}
		b.req.Body = ioutil.NopCloser(bytes.NewReader(byts))
		b.req.ContentLength = int64(len(byts))
		b.req.Header.Set("Content-Type", "application/x+yaml")
	}
	return b, nil
}

// JSONBody adds request raw body encoding by JSON.
func (b *BeegoHTTPRequest) JSONBody(obj interface{}) (*BeegoHTTPRequest, error) {
	if b.req.Body == nil && obj != nil {
		byts, err := json.Marshal(obj)
		if err != nil {
			return b, err
		}
		b.req.Body = ioutil.NopCloser(bytes.NewReader(byts))
		b.req.ContentLength = int64(len(byts))
		b.req.Header.Set("Content-Type", "application/json")
	}
	return b, nil
}
// DoRequest will do the client.Do
func (b *BeegoHTTPRequest) DoRequest() (resp *http.Response, err error) {
    // 获取参数值
	var paramBody string 
	if len(b.params) > 0 {
		var buf bytes.Buffer
		for k, v := range b.params {
			for _, vv := range v {
				buf.WriteString(url.QueryEscape(k))
				buf.WriteByte('=')
				buf.WriteString(url.QueryEscape(vv))
				buf.WriteByte('&')
			}
		}
		paramBody = buf.String()
		paramBody = paramBody[0 : len(paramBody)-1]
	}

	b.buildURL(paramBody) // 根据参数生成请求url
	urlParsed, err := url.Parse(b.url)
	if err != nil {
		return nil, err
	}

	b.req.URL = urlParsed

	trans := b.setting.Transport

	if trans == nil {
		// 采取默认的Transport
		trans = &http.Transport{
			TLSClientConfig:     b.setting.TLSClientConfig,
			Proxy:               b.setting.Proxy,
			Dial:                TimeoutDialer(b.setting.ConnectTimeout, b.setting.ReadWriteTimeout),
			MaxIdleConnsPerHost: 100,
		}
	} else {
		// 如果 b.transport是 *http.Transport则使用该配置
		if t, ok := trans.(*http.Transport); ok {
			if t.TLSClientConfig == nil {
				t.TLSClientConfig = b.setting.TLSClientConfig
			}
			if t.Proxy == nil {
				t.Proxy = b.setting.Proxy
			}
			if t.Dial == nil {
				t.Dial = TimeoutDialer(b.setting.ConnectTimeout, b.setting.ReadWriteTimeout)
			}
		}
	}

    // cookie处理
	var jar http.CookieJar
	if b.setting.EnableCookie {
		if defaultCookieJar == nil { //使用默认的cookiejar
			createDefaultCookie() 
		}
		jar = defaultCookieJar
	}

	client := &http.Client{
		Transport: trans,
		Jar:       jar,
	}
    //UA设置
	if b.setting.UserAgent != "" && b.req.Header.Get("User-Agent") == "" {
		b.req.Header.Set("User-Agent", b.setting.UserAgent)
	}

    // 重定向策略
	if b.setting.CheckRedirect != nil {
		client.CheckRedirect = b.setting.CheckRedirect
	}
    // 
	if b.setting.ShowDebug {
        /*
        DumpRequest以HTTP / 1.x表示形式返回给定的请求。它应该仅由服务器用于调试客户端请求。返回的表示仅是近似值;在将其解析为http.Request时，初始请求的一些细节将丢失。特别是，标题字段名称的顺序和大小写丢失了。多值标头中的值的顺序保持不变。 HTTP / 2请求以HTTP / 1.x格式转储，而不是以其原始二进制表示形式转储。
        如果body为true，则DumpRequest也返回body。为此，它使用req.Body，然后将其替换为产生相同字节的新io.ReadCloser。如果DumpRequest返回错误，则req的状态是未定义的。
        */
		dump, err := httputil.DumpRequest(b.req, b.setting.DumpBody)
		if err != nil {
			log.Println(err.Error())
		}
		b.dump = dump
	}
	// 重连次数
	for i := 0; b.setting.Retries == -1 || i <= b.setting.Retries; i++ {
		resp, err = client.Do(b.req)
		if err == nil {
			break
		}
	}
	return resp, err
}

// 请求发送后可以使用bytes返回以[]byte形式body, 如果没有调用过DoRequest(),bytes方法将会直接调用DoRequest()
// it calls Response inner.
func (b *BeegoHTTPRequest) Bytes() ([]byte, error) {
	if b.body != nil {
		return b.body, nil
	}
    resp, err := b.getResponse() // 直接调用DoRequest()
    //请求后响应体的处理
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, nil
	}
    defer resp.Body.Close()
    // Content-Encoding设置
	if b.setting.Gzip && resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		b.body, err = ioutil.ReadAll(reader)
		return b.body, err
	}
	b.body, err = ioutil.ReadAll(resp.Body)
	return b.body, err
}

func (b *BeegoHTTPRequest) getResponse() (*http.Response, error) {
	if b.resp.StatusCode != 0 {
		return b.resp, nil
	}
	resp, err := b.DoRequest()
	if err != nil {
		return nil, err
	}
	b.resp = resp
	return resp, nil
}


// ToFile 将响应体中的数据存储为文件
// it calls Response inner.
func (b *BeegoHTTPRequest) ToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := b.getResponse()
	if err != nil {
		return err
	}
	if resp.Body == nil {
		return nil
	}
	defer resp.Body.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
```

