# 请求与响应
## 请求

请求结构如下
```go
type Request struct {
    //Method指定HTTP方法（GET，POST，PUT等）。对于客户端请求，空字符串表示GET
	//Go的HTTP客户端不支持使用CONNECT方法发送请求。有关详细信息，请参阅传输文档
	// 请求方法的定义在net/http/method.go中
	Method string

    // URL指定要求的URI（用于服务器请求）或要访问的URL（用于客户端请求）。
    // 对于服务器请求，URL从RequestURI中提供的URI中解析，存储在RequestURI中。对于大多数请求，Path和RawQuery以外的字段将为空。
    // 对于客户端请求，URL的主机指定要连接的服务器，而Request的主机字段可选地指定要在HTTP请求中发送的主机头值
	URL *url.URL

    // 传入服务器请求的协议版本。
    // 对于客户端请求，将忽略这些字段。 HTTP客户端代码始终使用HTTP/1.1或HTTP/2。
	Proto      string // "HTTP/1.0"
	ProtoMajor int    // 1
	ProtoMinor int    // 0

	// Header包含服务器接收或客户端发送的请求标头字段。
	//如果服务器收到带标题行的请求
	//	Host: example.com
	//	accept-encoding: gzip, deflate
	//	Accept-Language: en-us
	//	fOO: Bar
	//	foo: two
	//
	// 对应的结构为
	//
	//	Header = map[string][]string{
	//		"Accept-Encoding": {"gzip, deflate"},
	//		"Accept-Language": {"en-us"},
	//		"Foo": {"Bar", "two"},
	//	}
	//
	// 对于传入请求，Host Header将提升为Request.Host字段并从Header映射中删除。
	//
	// HTTP定义标头名称不区分大小写。请求解析器通过使用CanonicalHeaderKey实现此操作，使第一个字符和连字符大写和其余小写后面的任何字符。
	//
	// 对于客户端请求，某些标头（如Content-Length和Connection）会在需要时自动写入，Header中的值可能会被忽略。请参阅Request.Write方法的文档。
	Header Header

	// Body为请求体
	//
	// 对于客户端请求，nil正文表示请求没有正文，例如GET请求。 HTTP客户端的传输负责调用Close方法.
	//
	// 对于服务器请求，Request Body始终为非零，但在没有正文时立即返回EOF。服务器将关闭请求正文。 ServeHTTP处理器不需要。
	Body io.ReadCloser

	// GetBody定义了一个可选的func来返回Body的新副本。当重定向需要多次读取主体时，它用于客户端请求。使用GetBody仍然需要设置Body.
	//
	// 对于服务器请求这是没用的
	GetBody func() (io.ReadCloser, error)

	// ContentLength记录关联内容的长度。值-1表示长度未知。值> = 0表示可以从Body读取给定的字节数。对于客户端请求，具有非零主体的值为0也被视为未知。
	ContentLength int64

	// TransferEncoding列出从最外层到最内层的传输编码。空列表表示“身份”编码。 TransferEncoding通常可以忽略;发送和接收请求时，会根据需要自动添加和删除分块编码
	TransferEncoding []string

    // Close表示在回复此请求后（对于服务器）或发送此请求并读取其响应（对于客户端）之后是否关闭连接
    // 对于服务器请求，HTTP服务器自动处理此请求，处理程序不需要此字段
    // 对于客户端请求，设置此字段可防止在对同一主机的请求之间重复使用TCP连接，就像设置Transport.DisableKeepAlives一样
	Close bool

    // 对于服务器请求，Host指定要在其上查找URL的主机。根据RFC 7230的5.4节，这可以是“主机”标头的值，也可以是URL本身中给出的主机名。它可能是“host：port”的形式。对于国际域名，Host可以是Punycode或Unicode格式。如果需要，使用golang.org/x/net/idna将其转换为任一格式
	// 防止DNS重新绑定攻击，服务器处理程序应验证Host标头是否具有Handler认为自己具有权威性的值。包含的ServeMux支持注册到特定主机名的模式，从而保护其注册的处理程序。
	// 对于客户端请求，Host可以选择覆盖要发送的Host头。如果为空，则Request.Write方法使用URL.Host的值。主机可能包含国际域名。
	Host string

    // From包含已解析的表单数据，包括URL字段的查询参数和POST或PUT表单数据。
    // 此字段仅在调用ParseForm后可用
    // HTTP客户端忽略Form并改为使用Body
	Form url.Values

	// PostForm包含来自POST，PATCH或PUT主体参数的解析表单数据。
	// 该字段仅在调用ParseForm后可用。
    // HTTP客户端忽略PostForm并使用Body。
	PostForm url.Values

    // MultipartForm是解析的多部分表单，包括文件上传
    // 此字段仅在调用ParseMultipartForm后可用。
    // HTTP客户端忽略MultipartForm并改为使用Body。
	MultipartForm *multipart.Form

	// Trailer指定在请求者之后发送的其他Header
	// 对于服务器请求，Trailer Map最初仅包含具有零值的Trailer密钥。 （客户端声明它将在以后发送哪些预告片。）当处理程序从Body读取时，它不能引用Trailer。从Body返回EOF读取后，可以再次读取Trailer，如果客户端发送了Trailer，则会包含非零值。
	//
	// 对于客户端请求，必须将Trailer初始化为包含以后发送的Trailer密钥的Map。值可以是nil或其他值。 ContentLength必须为0或-1，以发送分块请求。发送HTTP请求后，可以在读取请求主体时更新映射值。一旦Body返回EOF，调用者就不能改变Trailer。
	//
	// 很少有HTTP客户端，服务器或代理支持HTTPTrailer。
	Trailer Header

	// RemoteAddr允许HTTP服务器和其他软件记录发送请求的网络地址，通常用于记录。 ReadRequest未填写此字段，并且没有已定义的格式。在调用处理程序之前，此程序包中的HTTP服务器将RemoteAddr设置为“IP：port”地址。
	// HTTP客户端忽略此字段。
	RemoteAddr string

	// RequestURI是客户端发送到服务器的Request-Line（RFC 7230，第3.1.1节）的未修改请求目标。通常应该使用URL字段。在HTTP客户端请求中设置此字段是错误的。
	RequestURI string

	// TLS允许HTTP服务器和其他软件记录有关收到请求的TLS连接的信息。 ReadRequest未填写此字段。
    // 此包中的HTTP服务器在调用处理程序之前为启用TLS的连接设置字段;否则它将字段保留为nil。HTTP客户端将忽略此字段。
	TLS *tls.ConnectionState

    // 取消是一个可选通道，其闭包表示客户端请求应被视为已取消。并非所有的RoundTripper实现都支持Cancel。
    // 对于服务器请求，此字段不适用
    // 不推荐使用：改为使用Context和WithContext方法。如果同时设置了Request的Cancel字段和上下文，则不确定是否遵循Cancel。
	Cancel <-chan struct{}

	// 响应是重定向响应，它导致创建此请求。此字段仅在客户端重定向期间填充。
	Response *Response

	// ctx是客户端或服务器上下文。它只能通过使用WithContext复制整个请求来修改。它是未导出的，以防止人们使用Context错误并改变同一请求的调用者持有的上下文。
	ctx context.Context
}
```

#### Method

所对应的定义放在net/http/Method.go文件中

相关定义如下

```go
const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH" // RFC 5789
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
)
```

使用

```go
....
if request.Method == http.MethodGet{
	....
}
...
```

### URL

URl类型在net/url/url.go文件中，相关定义如下

```go
// URL表示已解析的URL（从技术上讲，是URI引用）
// 表示的一般形式是
// [scheme:][//[userinfo@]host][/]path[?query][#fragment]
// 在scheme之后不以斜杠开头的URL被解释为：
// scheme:opaque[?query][#fragment]
// 请注意，Path字段以解码形式存储：/％47％6f％2f变为/Go/。结果是，无法判断Path中的哪些斜杠是原始URL中的斜杠，哪些是％2f 。这种区别很少很重要，但是当它出现时，代码不能直接使用Path。
// Parse函数在它返回的URL中设置Path和RawPath，而URL的String方法通过调用EscapedPath方法使用RawPath（如果它是Path的有效编码）
type URL struct {
	Scheme     string
	Opaque     string    // 编码的 opaque data
	User       *Userinfo // 用户名和密码信息，Userinfo类型是URL的用户名和密码详细信息的不可变封装。保证现有的Userinfo值具有用户名集（可能为空，如RFC 2396所允许的），以及可选的密码。相关定义在net/url/url.go文件
	Host       string    // 主机地址 或者 主机:端口
	Path       string    //路径（相对路径可能省略前导斜杠
	RawPath    string    // 编码路径（请参阅EscapedPath方法）
	ForceQuery bool      // 即使RawQuery为空，也附加一个查询（'？'）
	RawQuery   string    // 编码的查询值，没有'？'
	Fragment   string    // fragment 相关, 没有 '#'
}
```

相关方法

```go
// Parse将rawurl解析为URL结构
// rawurl可以是相对的（路径，没有主机）或绝对的（从scheme开始）。尝试解析没有scheme的主机名和路径是无效的，但由于解析歧义，可能不一定会返回错误
func Parse(rawurl string) (*URL, error) {
	// 去除 #frag
	u, frag := split(rawurl, "#", true)
	url, err := parse(u, false)
	if err != nil {
		return nil, &Error{"parse", u, err}
	}
	if frag == "" {
		return url, nil
	}
	if url.Fragment, err = unescape(frag, encodeFragment); err != nil {
		return nil, &Error{"parse", rawurl, err}
	}
	return url, nil
}

//ParseRequestURI将rawurl解析为URL结构。它假定在​​HTTP请求中接收到rawurl，因此rawurl仅被解释为绝对URI或绝对路径。
// 假设字符串rawurl没有#fragment后缀。 （Web浏览器在将URL发送到Web服务器之前剥离#fragment。）
// ParseRequestURI主要实现是通过parse函数完成的
func ParseRequestURI(rawurl string) (*URL, error) {
	url, err := parse(rawurl, true)
	if err != nil {
		return nil, &Error{"parse", rawurl, err}
	}
	return url, nil
}

// EscapedPath返回转义形式的u.Path。通常，任何路径都有多种可能的转义形式。
EscapedPath返回u.RawPath，当它是u.Path.Otherwise的有效转义时，EscapedPath会忽略u.RawPath并自行计算转义形式。 String和RequestURI方法使用EscapedPath构造其结果。通常，代码应该调用EscapedPath而不是直接读取u.RawPath。
func (u *URL) EscapedPath() string {
	if u.RawPath != "" && validEncodedPath(u.RawPath) {
		p, err := unescape(u.RawPath, encodePath)
		if err == nil && p == u.Path {
			return u.RawPath
		}
	}
	if u.Path == "*" {
		return "*" // don't escape (Issue 11202)
	}
	return escape(u.Path, encodePath)
}


```

