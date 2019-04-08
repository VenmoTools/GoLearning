# HTTPClient

## 发送请求流程

发送GET请求，代码如下
```go
func main() {
	// 创建一个HTTPClient客户端
	client := http.Client{}
	// 新建一个请求
	request,err := http.NewRequest("GET","http://www.baidu.com",nil)
	if err != nil {
		fmt.Println(err)
	}
	// 发送请求
	res,err := client.Do(request)
	if err != nil {
		fmt.Println(err)
	}
	// 读取内容
	data,_:= ioutil.ReadAll(res.Body)

	fmt.Println(string(data))
}
```

1. 创建一个客户端，如果不是默认使用HTTP的DefaultClient
2. 使用NewRequest新建一个请求，该函数首先会判断传入的method，如果为空默认使用GET请求，随后使用validMethod来验证请求是否合法，使用url.Parse来解析传入的url，处理传入的body，首先尝试转换为ReadCloser如果无法转换则包装成NopCloser，然后调用removeEmptyPort//todo，新建Request然后根据传入的body进行解析，主要是bytes.Buffer，bytes.Reader，strings.Reader，设置ContentLength和GetBody的函数如果没有指定body则使用NoBody
3. 使用Do方法发送请求，其中有一个testHookClientDoResult函数，该函数在请求结束后调用，随后检查请求URL如果没有通过直接调用closebody并返回错误
4. 初始化变量，主要内容有deadline（通过deadline()方法生成，根据设置超时时间），reqs(*Request类型的切片),resp()

## Client

Client结构如下

```go
// 客户端是HTTP客户端。它的nil值（DefaultClient）是一个可用的DefaultTransport客户端
// 客户端的传输通常具有内部状态（缓存的TCP连接），因此应该重用客户端而不是根据需要创建客户端。客户可以安全地同时使用多个goroutine。
// 客户端比RoundTripper（例如Transport）更高级，并且还处理HTTP细节，例如cookie和重定向。

// 在关注重定向后，客户端将转发初始请求中设置的所有Header，但以下情况除外：
//      将敏感的Header（如“Authorization”，“WWW-Authenticate”和“Cookie”）转发给不受信任的目标时。在重定向到不是子域匹配或完全匹配初始域的域时，将忽略这些Header。例如，从“foo.com”到“foo.com”或“sub.foo.com”的重定向将转发敏感Header，但不会重定向到“bar.com”。
//      使用非零cookie转发“Cookie”Header时。由于每个重定向可能会改变cookie jar的状态，重定向可能会改变初始请求中的cookie集。当转发“Cookie”Header时，任何变异的cookie都将被省略，期望Jar将插入带有更新值的变异cookie（假设原点匹配）。如果Jar为nil，则初始cookie转发而不更改
type Client struct {
	// Transport指定单个HTTP请求的生成机制。如果为nil，则使用DefaultTransport。
	Transport RoundTripper

    // CheckRedirect指定处理重定向的策略
    // 如果CheckRedirect不是nil，客户端会在执行HTTP重定向之前调用它。参数req和via是即将发出的请求和已经提出的请求，最早的。如果CheckRedirect返回错误，则客户端的Get方法返回先前的Response（其Body已关闭）和CheckRedirect的错误（包含在url.Error中）而不是发出Request req。如果是特殊情况，如果CheckRedirect返回ErrUseLastResponse，则最近的响应以其bodyunclosed返回，并返回nil。

    //如果CheckRedirect为nil，则客户端使用其默认策略，即在连续10次请求后停止
	CheckRedirect func(req *Request, via []*Request) error

    // Jar指定的cookie jar
    // Jar用于将相关的cookie插入到每个出站请求中，并使用每个入站响应的cookie值进行更新。对于客户端遵循的每个重定向，都会使用jar
    // 如果Jar为零，只有在请求中明确设置了cookie，才会发送cookie
	Jar CookieJar

    // Timeout指定此客户端发出的请求的时间限制。超时包括连接时间，任何重定向和读取响应体。 Get，Head，Post或Do返回后计时器仍然运行，并将中断Response.Body的读取
    // Timeout为零意味着没有超时
    // 客户端取消对底层传输的请求，就像请求的上下文结束一样
    // 为了兼容性，如果找到客户端还将在Transport上使用已弃用的CancelRequest方法。新的RoundTripper实现应该使用Request的Context来取消而不是实现CancelRequest
	Timeout time.Duration
}

// 相关方法
// 发送HTTP请求并返回HTTP响应，遵循客户端上配置的策略（例如重定向，cookie，auth）
// 如果由客户端策略（例如CheckRedirect）或无法说HTTP（例如网络连接问题）引起错误，则返回错误。非2xx状态代码不会导致错误
// 如果返回的错误为nil，则响应将包含用户应关闭的非零主体。如果Body未关闭，则客户端的基础RoundTripper（通常为Transport）可能无法重新使用到服务器的持久TCP连接以用于后续“保持活动”请求。
// 请求Body（如果非零）将由底层传输关闭，即使出现错误也是如此。
// 出错时，可以忽略任何响应。只有在CheckRedirect失败时才会出现带有非零错误的非零响应，即使这样，返回的Response.Body也已关闭。
//通常使用Get，Post或PostForm代替Do

//如果服务器回复重定向，则客户端首先使用CheckRedirect函数来确定是否应遵循重定向。如果允许，301,302或303重定向会导致后续请求使用HTTP方法GET（如果原始请求是HEAD则为HEAD），没有正文。如果定义了Request.GetBody函数，则307或308重定向会保留原始HTTP方法和正文。 NewRequest函数自动为常见的标准库体类型设置GetBody。
//任何返回的错误都是* url.Error类型。如果请求超时或被取消，url.Error值的Timeout方法将返回true。
func (c *Client) Do(req *Request) (*Response, error) {
	return c.do(req)
}


func (c *Client) do(req *Request) (retres *Response, reterr error) {
	if testHookClientDoResult != nil {
		defer func() { testHookClientDoResult(retres, reterr) }()
    }
    // 判断URL
	if req.URL == nil {
		req.closeBody()
		return nil, &url.Error{
			Op:  urlErrorOp(req.Method),
			Err: errors.New("http: nil Request.URL"),
		}
	}

	var (
		deadline      = c.deadline() // 根据客户端设置返回超时时间
		reqs          []*Request
		resp          *Response
		copyHeaders   = c.makeHeadersCopier(req) // makeHeadersCopier创建了一个从最初的Request，ireq复制头文件的函数。对于每个重定向，必须调用此函数，以便它可以将标头复制到即将发生的请求中。
		reqBodyClosed = false // 是否关闭了当前的req.Body

		// 重定向行为
		redirectMethod string //重定向方式
		includeBody    bool // 是否包含body
    )
    // 屏蔽用户信息的error
	uerr := func(err error) error {
		//body 可能已被c.send()关闭
		if !reqBodyClosed {
			req.closeBody()
		}
		var urlStr string
        // url可能还有密码信息，stripPassword将密码替换为****
		if resp != nil && resp.Request != nil { // 如果响应不为空则从请求中提取，否则从响应中提取
			urlStr = stripPassword(resp.Request.URL)
		} else {
			urlStr = stripPassword(req.URL)
		}
		return &url.Error{
			Op:  urlErrorOp(reqs[0].Method),
			URL: urlStr,
			Err: err,
		}
	}
	for {
        // 对于除第一个请求之外的所有请求，创建下一个请求跃点并替换req
		if len(reqs) > 0 {
            // 尝试获取Location，当浏览器接受到头信息中的 Location: xxxx 后，就会自动跳转到 xxxx 指向的URL地址
            loc := resp.Header.Get("Location")
			if loc == "" { // 如果没有
				resp.closeBody()
				return nil, uerr(fmt.Errorf("%d response missing Location header", resp.StatusCode))
            }
            // 尝试解析跳转的URL
			u, err := req.URL.Parse(loc)
			if err != nil {
				resp.closeBody()
				return nil, uerr(fmt.Errorf("failed to parse Location header %q: %v", loc, err))
			}
            host := ""
            // 如果请求中的Host和req.URL中的Host不一致
			if req.Host != "" && req.Host != req.URL.Host {
				// 如果调用者指定了自定义主机Header并且重定向位置是相对的，则通过重定向保留主机Header
				if u, _ := url.Parse(loc); u != nil && !u.IsAbs() {
					host = req.Host
				}
			}
			ireq := reqs[0] // 获取第一个Request
			req = &Request{
				Method:   redirectMethod, // 重定向的Method
				Response: resp,
				URL:      u, //解析好的重定向URL
				Header:   make(Header),
				Host:     host,
				Cancel:   ireq.Cancel,
				ctx:      ireq.ctx,
            }
            // 如果包含body并且GetBody函数不为nil
			if includeBody && ireq.GetBody != nil {
				req.Body, err = ireq.GetBody()
				if err != nil {
					resp.closeBody()
					return nil, uerr(err)
				}
				req.ContentLength = ireq.ContentLength
			}

			// 在设置Referer之前复制原始Header，以防用户在第一次请求时设置Referer。如果真的想要覆盖，可以在的CheckRedirect函数中执行
			copyHeaders(req)

			// 添加最新的 Referer header
            // 请求新网址，如果不是https-> http
            // 如果reqs[len(reqs)-1].URL 的Referer是https而req.URL 的Referer是http，则refererForURL返回没有任何身份验证信息的引用者或空字符串
			if ref := refererForURL(reqs[len(reqs)-1].URL, req.URL); ref != "" {
				req.Header.Set("Referer", ref)
            }
            // checkRedirect调用用户配置的CheckRedirect函数或默认值
			err = c.checkRedirect(req, reqs)

			// Sentinel err 让用户选择上一个响应，而不关闭其body. See Issue 10069.
			if err == ErrUseLastResponse {
				return resp, nil
			}

			// 关闭上一个响应的正文。但至少读取一些body，所以如果它很小，底层的TCP连接将被重用。无需检查错误：如果失败，则无论如何都不会重复使用它。
			const maxBodySlurpSize = 2 << 10
			if resp.ContentLength == -1 || resp.ContentLength <= maxBodySlurpSize {
				io.CopyN(ioutil.Discard, resp.Body, maxBodySlurpSize)
			}
			resp.Body.Close()

			if err != nil {
				// Go 1兼容性的特殊情况：如果CheckRedirect功能失败，则返回响应和错误
                // See https://golang.org/issue/3795
                // body已经被关闭了
				ue := uerr(err)
				ue.(*url.Error).URL = loc
				return resp, ue
			}
		}

		reqs = append(reqs, req)
		var err error
		var didTimeout func() bool
		if resp, didTimeout, err = c.send(req, deadline); err != nil {
			// c.send()总是关闭 req.Body
            reqBodyClosed = true
            // 判断是否超时
			if !deadline.IsZero() && didTimeout() {
				err = &httpError{
					// TODO: 在周期的早期：s / Client.Timeout超出/超时或上下文取消/
					err:     err.Error() + " (Client.Timeout exceeded while awaiting headers)",
					timeout: true,
				}
			}
			return nil, uerr(err)
		}

		var shouldRedirect bool
		redirectMethod, shouldRedirect, includeBody = redirectBehavior(req.Method, resp, reqs[0])
		if !shouldRedirect {
			return resp, nil
		}

		req.closeBody()
	}
}


// makeHeadersCopier创建了一个从最初的Request，ireq复制头文件的函数。对于每个重定向，必须调用此函数，以便它可以将标头复制到即将发生的请求中。
func (c *Client) makeHeadersCopier(ireq *Request) func(*Request) {
	// 要复制的Header来自最初的请求。
	// 使用一个关闭的回调来保持对这些原始Header的引用
	var (
		ireqhdr  = ireq.Header.clone()
		icookies map[string][]*Cookie
    )
    // 从Jar中获取Cookie
	if c.Jar != nil && ireq.Header.Get("Cookie") != "" {
		icookies = make(map[string][]*Cookie)
		for _, c := range ireq.Cookies() {
			icookies[c.Name] = append(icookies[c.Name], c)
		}
	}

	preq := ireq // 先前的请求
	return func(req *Request) {
		// 如果Jar存在并且通过请求标头提供了一些初始cookie，那么可能需要在遵循重定向时更改初始cookie，因为每个重定向可能最终修改预先存在的cookie。
		//
		// 由于已在请求标头中设置的cookie不包含有关原始域和路径的信息，因此以下逻辑假定任何新的设置cookie都覆盖原始cookie，而不管域或路径如何
		//
		// See https://golang.org/issue/17494
		if c.Jar != nil && icookies != nil {
			var changed bool
			resp := req.Response // 导致即将进行重定向的响应
			for _, c := range resp.Cookies() { // 获取响应Cookie
				if _, ok := icookies[c.Name]; ok {//如果包含相同的Cookie名 则删除
					delete(icookies, c.Name)
					changed = true //cookie发生改变
				}
			}
			if changed { 
				ireqhdr.Del("Cookie")   // 删除旧cookie
				var ss []string
				for _, cs := range icookies { // 更新cookie
					for _, c := range cs {
						ss = append(ss, c.Name+"="+c.Value)
					}
				}
				sort.Strings(ss) // 确保准确的Header
				ireqhdr.Set("Cookie", strings.Join(ss, "; "))
			}
		}

		// 复制初始请求的Header值(至少是安全的).
		for k, vv := range ireqhdr {
            /*
            允许从“foo.com”向“sub.foo.com”发送auth/cookie header
            请注意，不会自动将所有Cookie发送到子域。此功能仅用于在初始传出客户端请求上明确设置的Cookie。通过CookieJar机制自动添加的Cookie会继续遵循Set-Cookie设置的每个Cookie的范围。但对于直接设置Cookie标头的传出请求，不知道它们的范围，因此假设为* .domain.com。*/
			if shouldCopyHeaderOnRedirect(k, preq.URL, req.URL) {
				req.Header[k] = vv
			}
		}

		preq = req // 使用当前请求更新先前的请求
	}
}


// 只有当err != nil时，didTimeout才是非零的。
func (c *Client) send(req *Request, deadline time.Time) (resp *Response, didTimeout func() bool, err error) {
    // 判断需要添加cookie
	if c.Jar != nil {
		for _, cookie := range c.Jar.Cookies(req.URL) {
			req.AddCookie(cookie)
		}
    }
    // 调用send函数
	resp, didTimeout, err = send(req, c.transport(), deadline)
	if err != nil {
		return nil, didTimeout, err
	}
	if c.Jar != nil {
		if rc := resp.Cookies(); len(rc) > 0 {
			c.Jar.SetCookies(req.URL, rc)
		}
	}
	return resp, nil, nil
}


// 发送HTTP请求
// 调用者在完成读取后应该关闭resp.Body。
func send(ireq *Request, rt RoundTripper, deadline time.Time) (resp *Response, didTimeout func() bool, err error) {
	req := ireq // req是原始请求或修改后的fork

    // 检查参数
	if rt == nil {
		req.closeBody()
		return nil, alwaysFalse, errors.New("http: no Client.Transport or DefaultTransport")
	}

	if req.URL == nil {
		req.closeBody()
		return nil, alwaysFalse, errors.New("http: nil Request.URL")
	}

	if req.RequestURI != "" {
		req.closeBody()
		return nil, alwaysFalse, errors.New("http: Request.RequestURI can't be set in client requests.")
	}

	// forkReq在第一次调用时将req转换为ireq的浅层复制
	forkReq := func() {
		if ireq == req {
			req = new(Request)
			*req = *ireq // shallow clone
		}
	}

	// send（Get，Post等）的大多数调用者都不需要Headers，而是将其保留为未初始化状态。不过，已经保证Transport初始化。
	if req.Header == nil {
		forkReq()
		req.Header = make(Header)
	}

    // 如果URL包含账号密码信息，添加到Header中
	if u := req.URL.User; u != nil && req.Header.Get("Authorization") == "" {
		username := u.Username()
		password, _ := u.Password()
		forkReq()
		req.Header = cloneHeader(ireq.Header)
		req.Header.Set("Authorization", "Basic "+basicAuth(username, password))
	}

	if !deadline.IsZero() {
		forkReq()
    }
    // 如果截止时间非零，则setRequestCancel设置req的Cancel字段。 RoundTripper的类型用于确定是否应使用传统的CancelRequest行为
	stopTimer, didTimeout := setRequestCancel(req, rt, deadline)

    // 调用RoundTrip
	resp, err = rt.RoundTrip(req)
	if err != nil {
		stopTimer()
		if resp != nil {
			log.Printf("RoundTripper returned a response & error; ignoring response")
		}
		if tlsErr, ok := err.(tls.RecordHeaderError); ok {
			// 如果得到一个错误的TLS记录，请检查响应是否看起来像HTTP并提供更有用的错误。
			// See golang.org/issue/11111.
			if string(tlsErr.RecordHeader[:]) == "HTTP/" {
				err = errors.New("http: server gave HTTP response to HTTPS client")
			}
		}
		return nil, didTimeout, err
	}
	if !deadline.IsZero() {
		resp.Body = &cancelTimerBody{
			stop:          stopTimer,
			rc:            resp.Body,
			reqDidTimeout: didTimeout,
		}
	}
	return resp, nil, nil
}
```
#### Transport 
RoundTripper是一个接口，表示执行单个HTTP事务的能力，获取给定请求的响应。RoundTripper必须安全，才能由多个goroutine同时使用，定义如下

```go
type RoundTripper interface {
    // RoundTrip执行单个HTTP事务，为提供的Request返回Response
    // RoundTrip不应尝试解释响应。特别是，如果RoundTrip获得响应，则必须返回err == nil，而不管响应的HTTP状态代码如何,应该为未能获得响应而保留非零错误。同样，RoundTrip不应尝试处理更高级别的协议详细信息，例如重定向，身份验证或Cookie
    //除了使用和关闭Request的Body之外，RoundTrip不应该修改请求。 RoundTrip可以在单独的goroutine中读取请求的字段。在响应机构关闭之前，调用者者不应改变或重复使用请求。
    // RoundTrip必须始终关闭正文，包括错误，但是根据实现可能会在单独的goroutine中执行，即使在RoundTrip返回之后也是如此。这意味着希望为后续请求重用主体的呼叫者必须安排在执行此操作之前等待Close呼叫
	// 必须初始化请求的URL和标题字段
	RoundTrip(*Request) (*Response, error)
}

// DefaultTransport结构如下
var DefaultTransport RoundTripper = &Transport{
	Proxy: ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

// Transport结构如下
type Transport struct {
	idleMu     sync.Mutex
	wantIdle   bool                                // user has requested to close all idle conns
	idleConn   map[connectMethodKey][]*persistConn // most recently used at end
	idleConnCh map[connectMethodKey]chan *persistConn
	idleLRU    connLRU

	reqMu       sync.Mutex
	reqCanceler map[*Request]func(error)

	altMu    sync.Mutex   // guards changing altProto only
	altProto atomic.Value // of nil or map[string]RoundTripper, key is URI scheme

	connCountMu          sync.Mutex
	connPerHostCount     map[connectMethodKey]int
	connPerHostAvailable map[connectMethodKey]chan struct{}

	// Proxy specifies a function to return a proxy for a given
	// Request. If the function returns a non-nil error, the
	// request is aborted with the provided error.
	//
	// The proxy type is determined by the URL scheme. "http",
	// "https", and "socks5" are supported. If the scheme is empty,
	// "http" is assumed.
	//
	// If Proxy is nil or returns a nil *URL, no proxy is used.
	Proxy func(*Request) (*url.URL, error)

	// DialContext specifies the dial function for creating unencrypted TCP connections.
	// If DialContext is nil (and the deprecated Dial below is also nil),
	// then the transport dials using package net.
	//
	// DialContext runs concurrently with calls to RoundTrip.
	// A RoundTrip call that initiates a dial may end up using
	// an connection dialed previously when the earlier connection
	// becomes idle before the later DialContext completes.
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)

	// Dial specifies the dial function for creating unencrypted TCP connections.
	//
	// Dial runs concurrently with calls to RoundTrip.
	// A RoundTrip call that initiates a dial may end up using
	// an connection dialed previously when the earlier connection
	// becomes idle before the later Dial completes.
	//
	// Deprecated: Use DialContext instead, which allows the transport
	// to cancel dials as soon as they are no longer needed.
	// If both are set, DialContext takes priority.
	Dial func(network, addr string) (net.Conn, error)

	// DialTLS specifies an optional dial function for creating
	// TLS connections for non-proxied HTTPS requests.
	//
	// If DialTLS is nil, Dial and TLSClientConfig are used.
	//
	// If DialTLS is set, the Dial hook is not used for HTTPS
	// requests and the TLSClientConfig and TLSHandshakeTimeout
	// are ignored. The returned net.Conn is assumed to already be
	// past the TLS handshake.
	DialTLS func(network, addr string) (net.Conn, error)

	// TLSClientConfig specifies the TLS configuration to use with
	// tls.Client.
	// If nil, the default configuration is used.
	// If non-nil, HTTP/2 support may not be enabled by default.
	TLSClientConfig *tls.Config

	// TLSHandshakeTimeout specifies the maximum amount of time waiting to
	// wait for a TLS handshake. Zero means no timeout.
	TLSHandshakeTimeout time.Duration

	// DisableKeepAlives, if true, disables HTTP keep-alives and
	// will only use the connection to the server for a single
	// HTTP request.
	//
	// This is unrelated to the similarly named TCP keep-alives.
	DisableKeepAlives bool

	// DisableCompression, if true, prevents the Transport from
	// requesting compression with an "Accept-Encoding: gzip"
	// request header when the Request contains no existing
	// Accept-Encoding value. If the Transport requests gzip on
	// its own and gets a gzipped response, it's transparently
	// decoded in the Response.Body. However, if the user
	// explicitly requested gzip it is not automatically
	// uncompressed.
	DisableCompression bool

	// MaxIdleConns controls the maximum number of idle (keep-alive)
	// connections across all hosts. Zero means no limit.
	MaxIdleConns int

	// MaxIdleConnsPerHost, if non-zero, controls the maximum idle
	// (keep-alive) connections to keep per-host. If zero,
	// DefaultMaxIdleConnsPerHost is used.
	MaxIdleConnsPerHost int

	// MaxConnsPerHost optionally limits the total number of
	// connections per host, including connections in the dialing,
	// active, and idle states. On limit violation, dials will block.
	//
	// Zero means no limit.
	//
	// For HTTP/2, this currently only controls the number of new
	// connections being created at a time, instead of the total
	// number. In practice, hosts using HTTP/2 only have about one
	// idle connection, though.
	MaxConnsPerHost int

	// IdleConnTimeout is the maximum amount of time an idle
	// (keep-alive) connection will remain idle before closing
	// itself.
	// Zero means no limit.
	IdleConnTimeout time.Duration

	// ResponseHeaderTimeout, if non-zero, specifies the amount of
	// time to wait for a server's response headers after fully
	// writing the request (including its body, if any). This
	// time does not include the time to read the response body.
	ResponseHeaderTimeout time.Duration

	// ExpectContinueTimeout, if non-zero, specifies the amount of
	// time to wait for a server's first response headers after fully
	// writing the request headers if the request has an
	// "Expect: 100-continue" header. Zero means no timeout and
	// causes the body to be sent immediately, without
	// waiting for the server to approve.
	// This time does not include the time to send the request header.
	ExpectContinueTimeout time.Duration

	// TLSNextProto specifies how the Transport switches to an
	// alternate protocol (such as HTTP/2) after a TLS NPN/ALPN
	// protocol negotiation. If Transport dials an TLS connection
	// with a non-empty protocol name and TLSNextProto contains a
	// map entry for that key (such as "h2"), then the func is
	// called with the request's authority (such as "example.com"
	// or "example.com:1234") and the TLS connection. The function
	// must return a RoundTripper that then handles the request.
	// If TLSNextProto is not nil, HTTP/2 support is not enabled
	// automatically.
	TLSNextProto map[string]func(authority string, c *tls.Conn) RoundTripper

	// ProxyConnectHeader optionally specifies headers to send to
	// proxies during CONNECT requests.
	ProxyConnectHeader Header

	// MaxResponseHeaderBytes specifies a limit on how many
	// response bytes are allowed in the server's response
	// header.
	//
	// Zero means to use a default limit.
	MaxResponseHeaderBytes int64

	// nextProtoOnce guards initialization of TLSNextProto and
	// h2transport (via onceSetNextProtoDefaults)
	nextProtoOnce sync.Once
	h2transport   h2Transport // non-nil if http2 wired up
}
// 接口的实现方法在net/http/roundtrip.go文件中，但是具体内容实现在net/http/transport.go文件中
// roundTrip通过HTTP实现RoundTripper。
func (t *Transport) roundTrip(req *Request) (*Response, error) {
	t.nextProtoOnce.Do(t.onceSetNextProtoDefaults)
	ctx := req.Context()
	trace := httptrace.ContextClientTrace(ctx)

	if req.URL == nil {
		req.closeBody()
		return nil, errors.New("http: nil Request.URL")
	}
	if req.Header == nil {
		req.closeBody()
		return nil, errors.New("http: nil Request.Header")
	}
	scheme := req.URL.Scheme
	isHTTP := scheme == "http" || scheme == "https"
	if isHTTP {
		for k, vv := range req.Header {
			if !httpguts.ValidHeaderFieldName(k) {
				return nil, fmt.Errorf("net/http: invalid header field name %q", k)
			}
			for _, v := range vv {
				if !httpguts.ValidHeaderFieldValue(v) {
					return nil, fmt.Errorf("net/http: invalid header field value %q for key %v", v, k)
				}
			}
		}
	}

	altProto, _ := t.altProto.Load().(map[string]RoundTripper)
	if altRT := altProto[scheme]; altRT != nil {
		if resp, err := altRT.RoundTrip(req); err != ErrSkipAltProtocol {
			return resp, err
		}
	}
	if !isHTTP {
		req.closeBody()
		return nil, &badStringError{"unsupported protocol scheme", scheme}
	}
	if req.Method != "" && !validMethod(req.Method) {
		return nil, fmt.Errorf("net/http: invalid method %q", req.Method)
	}
	if req.URL.Host == "" {
		req.closeBody()
		return nil, errors.New("http: no Host in request URL")
	}

	for {
		select {
		case <-ctx.Done():
			req.closeBody()
			return nil, ctx.Err()
		default:
		}

		// treq gets modified by roundTrip, so we need to recreate for each retry.
		treq := &transportRequest{Request: req, trace: trace}
		cm, err := t.connectMethodForRequest(treq)
		if err != nil {
			req.closeBody()
			return nil, err
		}

		// Get the cached or newly-created connection to either the
		// host (for http or https), the http proxy, or the http proxy
		// pre-CONNECTed to https server. In any case, we'll be ready
		// to send it requests.
		pconn, err := t.getConn(treq, cm)
		if err != nil {
			t.setReqCanceler(req, nil)
			req.closeBody()
			return nil, err
		}

		var resp *Response
		if pconn.alt != nil {
			// HTTP/2 path.
			t.decHostConnCount(cm.key()) // don't count cached http2 conns toward conns per host
			t.setReqCanceler(req, nil)   // not cancelable with CancelRequest
			resp, err = pconn.alt.RoundTrip(req)
		} else {
			resp, err = pconn.roundTrip(treq)
		}
		if err == nil {
			return resp, nil
		}
		if !pconn.shouldRetryRequest(req, err) {
			// Issue 16465: return underlying net.Conn.Read error from peek,
			// as we've historically done.
			if e, ok := err.(transportReadFromServerError); ok {
				err = e.err
			}
			return nil, err
		}
		testHookRoundTripRetried()

		// Rewind the body if we're able to.  (HTTP/2 does this itself so we only
		// need to do it for HTTP/1.1 connections.)
		if req.GetBody != nil && pconn.alt == nil {
			newReq := *req
			var err error
			newReq.Body, err = req.GetBody()
			if err != nil {
				return nil, err
			}
			req = &newReq
		}
	}
}
```


## Cookie

Cookie的结构如下

```go
type Cookie struct {
	Name  string
	Value string

	Path       string    // optional
	Domain     string    // optional
	Expires    time.Time // optional
	RawExpires string    // for reading cookies only

	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	// MaxAge>0 means Max-Age attribute present and given in seconds
	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite SameSite
	Raw      string
	Unparsed []string // Raw text of unparsed attribute-value pairs
}

```