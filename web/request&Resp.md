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
// EscapedPath返回u.RawPath，当它是u.Path.Otherwise的有效转义时，EscapedPath会忽略u.RawPath并自行计算转义形式。 String和RequestURI方法使用EscapedPath构造其结果。通常，代码应该调用EscapedPath而不是直接读取u.RawPath。
// 解析URL的协议和host后便开始调用EscapedPath解析Path，通过validEncodedPath判断是否是一个规范的url通过下面的escape去编码，随后会对所有字符进行转码
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

// String()将URL重新组合为有效的URL字符串。
//结果的一般形式是：
//	scheme:opaque?query#fragment
//	scheme://userinfo@host/path?query#fragment
// 如果u.Opaque非空，String使用第一个from;否则它使用第二种形式。
//要获取路径，String使用u.EscapedPath（）
//在第二种形式中，以下规则适用：
//  - 如果u.Scheme为空，则省略scheme。
//  - 如果u.User为nil，则省略userinfo@
//	- 如果u.Host为空，则省略host
//	- 如果u.Scheme和u.Host都是空的而且u.User是nil，整个 scheme://userinfo@host/被省略
//	- 如果u.Host非空并且u.Path以/开头 表单主机/路径不添加自己的/
// 	- 如果u.RawQuery为空，则省略查询
//	- 如果u.Fragment为空，则省略#fragment
func (u *URL) String() string {
	var buf strings.Builder
	if u.Scheme != "" {
		buf.WriteString(u.Scheme)
		buf.WriteByte(':')
	}
	if u.Opaque != "" {
		buf.WriteString(u.Opaque)
	} else {
		if u.Scheme != "" || u.Host != "" || u.User != nil {
			if u.Host != "" || u.Path != "" || u.User != nil {
				buf.WriteString("//")
			}
			if ui := u.User; ui != nil {
				buf.WriteString(ui.String())
				buf.WriteByte('@')
			}
			if h := u.Host; h != "" {
				buf.WriteString(escape(h, encodeHost))
			}
		}
		path := u.EscapedPath()
		if path != "" && path[0] != '/' && u.Host != "" {
			buf.WriteByte('/')
		}
		if buf.Len() == 0 {
			// RFC 3986 §4.2
			// A path segment that contains a colon character (e.g., "this:that")
			// cannot be used as the first segment of a relative-path reference, as
			// it would be mistaken for a scheme name. Such a segment must be
			// preceded by a dot-segment (e.g., "./this:that") to make a relative-
			// path reference.
			if i := strings.IndexByte(path, ':'); i > -1 && strings.IndexByte(path[:i], '/') == -1 {
				buf.WriteString("./")
			}
		}
		buf.WriteString(path)
	}
	if u.ForceQuery || u.RawQuery != "" {
		buf.WriteByte('?')
		buf.WriteString(u.RawQuery)
	}
	if u.Fragment != "" {
		buf.WriteByte('#')
		buf.WriteString(escape(u.Fragment, encodeFragment))
	}
	return buf.String()
}

//ParseQuery解析URL编码的查询字符串并返回列出为每个key指定的值的映射.ParseQuery始终返回包含找到的所有有效查询参数的非零映射;错误描述了遇到的第一个解码错误，如果有的话。
//查询应该是由＆符号或分号分隔的键=值设置列表。没有等号的设置被解释为设置为空值的键
// ParseQuery主要由parseQuery实现
func ParseQuery(query string) (Values, error) {
	m := make(Values)
	err := parseQuery(m, query)
	return m, err
}

// Encode将值编码为按键排序的“URL编码”形式（“bar = baz＆foo = quux”）
func (v Values) Encode() string {
	if v == nil {
		return ""
	}
	var buf strings.Builder
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vs := v[k]
		keyEscaped := QueryEscape(k)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(QueryEscape(v))
		}
	}
	return buf.String()
}

// 根据RFC 3986第5.2节，ResolveReference根据绝对基URI来解析对绝对URI的URI引用。 URI引用可以是相对的或绝对的。 ResolveReference始终返回一个新的URL实例，即使返回的URL与base或reference相同。如果ref是绝对URL，则ResolveReference会忽略base并返回ref的副本。
func (u *URL) ResolveReference(ref *URL) *URL {
	url := *ref
	if ref.Scheme == "" {
		url.Scheme = u.Scheme
	}
	if ref.Scheme != "" || ref.Host != "" || ref.User != nil {
		// The "absoluteURI" or "net_path" cases.
		// We can ignore the error from setPath since we know we provided a
		// validly-escaped path.
		url.setPath(resolvePath(ref.EscapedPath(), ""))
		return &url
	}
	if ref.Opaque != "" {
		url.User = nil
		url.Host = ""
		url.Path = ""
		return &url
	}
	if ref.Path == "" && ref.RawQuery == "" {
		url.RawQuery = u.RawQuery
		if ref.Fragment == "" {
			url.Fragment = u.Fragment
		}
	}
	// The "abs_path" or "rel_path" cases.
	url.Host = u.Host
	url.User = u.User
	url.setPath(resolvePath(u.EscapedPath(), ref.EscapedPath()))
	return &url
}
```

### Header

Header是一个Map 定义如下

type Header map[string][]string

Header有8个非指针方法，添加删除更新等均是由MIMEHeader实现的，MIMEHeader表示映射到值集的键的MIME样式Header。当添加新的key是会调用CanonicalMIMEHeaderKey，CanonicalMIMEHeaderKey返回MIMEHeader key的规范格式。规范化将第一个字母和连字符后的任何字母转换为大写;其余的都转换成小写。例如，“accept-encoding”的规范键是“Accept-Encoding”.MIMEheader key被假定为仅ASCII。如果s包含空格或无效的头字段字节，则返回时不进行修改（主要是由canonicalMIMEHeaderKey方法实现，实现在net/textproto/reader.go中）

### From

Form和PostForm都是采用url.Values结构存储，方便添加，删除，修改，MultipartForm是一个multipart.Form类型multipart.Form是一个解析的多部分表单。它的文件部分存储在内存或磁盘上，可以通过*FileHeader的Open方法访问。它的值部分存储为字符串。两者都由字段名称键入。

定义如下
```go
type Form struct {
	Value map[string][]string
	File  map[string][]*FileHeader //FileHeader描述了多部分请求的文件部分
}

// FileHeader定义如下
type FileHeader struct {
	Filename string  //文件名
	Header   textproto.MIMEHeader
	Size     int64 // 文件大小

	content []byte //文件二进制内容
	tmpfile string 
}

// 打开并返回FileHeader的相关文件
func (fh *FileHeader) Open() (File, error) {
	// 如果当前content有数据
	if b := fh.content; b != nil {
		//NewSectionReader返回一个SectionReader，它在偏移关闭时从rstarting读取，在n个字节后用EOF停止。
		r := io.NewSectionReader(bytes.NewReader(b), 0, int64(len(b)))
		return sectionReadCloser{r}, nil // 类型将[]byte转换为File类型
	}
	return os.Open(fh.tmpfile)
}

// 主要方法
//ReadForm解析整个多部分消息，其部分具有Content-Disposition为“form-data”。它在内存中存储最多maxMemory字节+ 10MB（保留用于非文件部分）。无法存储在内存中的文件部分将以临时文件的形式存储在磁盘上。如果所有非文件部分都无法存储在内存中，则返回ErrMessageTooLarge。
func (r *Reader) ReadForm(maxMemory int64) (*Form, error) {
	return r.readForm(maxMemory)
}

func (r *Reader) readForm(maxMemory int64) (_ *Form, err error) {
	// 初始化，以及结束后的清理
	form := &Form{make(map[string][]string), make(map[string][]*FileHeader)}
	defer func() {
		if err != nil {
			form.RemoveAll() //RemoveAll删除与表单关联的所有临时文件
		}
	}()

	// 为非文件部分预留额外的10 MB
	maxValueBytes := maxMemory + int64(10<<20)
	for {
		p, err := r.NextPart() //NextPart返回multipart中的下一部分或错误。当没有其他部分时，返回错误io.EOF

		//错误处理
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}


		name := p.FormName() //如果p具有“form-data”类型的Content-Disposition，则FormName返回name参数。否则返回空字符串。
		if name == "" {
			continue
		}

		filename := p.FileName() //FileName返回Part'sContent-Disposition标头的filename参数

		var b bytes.Buffer

		if filename == "" { 
			// value，作为字符串存储在内存中
			n, err := io.CopyN(&b, p, maxValueBytes+1) // CopyN将maxValueBytes+1字节（或直到错误）从p复制到b。它返回复制的字节数和复制时遇到的earliesterror。返回时，写入== n当且仅当err == nil时。
			if err != nil && err != io.EOF {
				return nil, err
			}
			maxValueBytes -= n
			if maxValueBytes < 0 {
				return nil, ErrMessageTooLarge
			}
			form.Value[name] = append(form.Value[name], b.String())
			continue
		}

		// 文件，存储在内存或磁盘上
		fh := &FileHeader{
			Filename: filename,
			Header:   p.Header,
		}
		n, err := io.CopyN(&b, p, maxMemory+1)
		if err != nil && err != io.EOF {
			return nil, err
		}
		// 太大，写入磁盘和刷新缓冲区	
		if n > maxMemory {
			file, err := ioutil.TempFile("", "multipart-")
			if err != nil {
				return nil, err
			}
			size, err := io.Copy(file, io.MultiReader(&b, p))
			if cerr := file.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				os.Remove(file.Name())
				return nil, err
			}
			fh.tmpfile = file.Name()
			fh.Size = size
		} else {
			fh.content = b.Bytes()
			fh.Size = int64(len(fh.content))
			maxMemory -= n
			maxValueBytes -= n
		}
		form.File[name] = append(form.File[name], fh)
	}

	return form, nil
}

// ParseForm填充r.Form和r.PostForm
// 对于所有请求，ParseForm从URL解析原始查询并更新r.Form。
//对于POST，PUT和PATCH请求，它还将请求体解析为一个表单，并将结果放入r.PostForm和r.Form中。请求Body参数优先于r.Form中的URL查询字符串值。
// 对于其他HTTP方法，或者当Content-Type不是application / x-www-form-urlencoded时，不读取请求Body，并将r.PostForm初始化为非零空值。
// 如果请求Body的大小尚未受MaxBytesReader的限制，则大小上限为10MB
// ParseMultipartForm自动调用ParseForm
// ParseForm是幂等的(在编程中一个幂等操作的特点是其任意多次执行所产生的影响均与一次执行的影响相同。幂等函数，或幂等方法，是指可以使用相同参数重复执行，并能获得相同结果的函数,这些函数不会影响系统状态，也不用担心重复执行会对系统造成改变。)
func (r *Request) ParseForm() error {
	var err error
	if r.PostForm == nil { //非空判断
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
			r.PostForm, err = parsePostForm(r) 
		}
		if r.PostForm == nil { // 双重检测
			r.PostForm = make(url.Values)
		}
	}
	if r.Form == nil {
		if len(r.PostForm) > 0 {
			r.Form = make(url.Values)
			// 深拷贝，避免只拷贝引用
			copyValues(r.Form, r.PostForm)
		}
		var newValues url.Values
		if r.URL != nil {
			var e error
			newValues, e = url.ParseQuery(r.URL.RawQuery) //解析Path中的查询参数与值
			if err == nil {
				err = e
			}
		}
		if newValues == nil { // 如果没有参数使用空参数
			newValues = make(url.Values)
		}
		if r.Form == nil { // 如果form也没有参数则使用解析好的newValues
			r.Form = newValues
		} else {
			copyValues(r.Form, newValues)
		}
	}
	return err
}

// 解析PostForm实现
func parsePostForm(r *Request) (vs url.Values, err error) {
	// 检查
	if r.Body == nil {
		err = errors.New("missing form body")
		return
	}
	ct := r.Header.Get("Content-Type")
	//RFC 7231，第3.1.1.5节 - 空类型 可以视为application / octet-stream
	if ct == "" {
		ct = "application/octet-stream"
	}
	/*
	ParseMediaType根据RFC 1521解析媒体类型值和任何可选参数。媒体类型是Content-Type和Content-Disposition标头（RFC 2183）中的值。成功时，ParseMediaType返回转换为小写并修剪空白区域和非零映射的媒体类型。如果解析可选参数时出错，则将返回媒体类型以及错误ErrInvalidMediaParameter。返回的映射，参数，从小写属性映射到属性值并保留其大小写。
	*/
	ct, _, err = mime.ParseMediaType(ct)
	switch {
	case ct == "application/x-www-form-urlencoded":
		var reader io.Reader = r.Body
		maxFormSize := int64(1<<63 - 1)//最大form大小
		/*
		MaxBytesReader类似于io.LimitReader，但用于限制传入请求主体的大小。与io.LimitReader相反，MaxBytesReader的结果是ReadCloser，为超出限制的Read返回非EOF错误，并在调用Close方法时关闭底层读取器
		*/
		if _, ok := r.Body.(*maxBytesReader); !ok {
			maxFormSize = int64(10 << 20) // 10 MB 已经是很长的文本了.
			reader = io.LimitReader(r.Body, maxFormSize+1)
		}
		b, e := ioutil.ReadAll(reader)
		if e != nil {
			if err == nil {
				err = e
			}
			break
		}
		if int64(len(b)) > maxFormSize {
			err = errors.New("http: POST too large")
			return
		}
		vs, e = url.ParseQuery(string(b))
		if err == nil {
			err = e
		}
	case ct == "multipart/form-data":
		// handled by ParseMultipartForm (which is calling us, or should be)
		// 待完成 (bradfitz): 这里有太多可能的命令来调用太多的函数。清理它并编写更多tests.request_test.go包含TestParseMultipartFormOrder和其他的开头
	}
	return
}


// ParseMultipartForm将请求主体解析为multipart/form-data。解析整个请求体，并将其文件部分的总maxMemory字节存储在内存中，其余部分存储在临时files.ParseMultipartForm中，如果需要，调用ParseForm。在调用ParseMultipartForm后，后续调用无效。
func (r *Request) ParseMultipartForm(maxMemory int64) error {
	if r.MultipartForm == multipartByReader {
		return errors.New("http: multipart handled by MultipartReader")
	}
	if r.Form == nil {
		err := r.ParseForm()
		if err != nil {
			return err
		}
	}
	if r.MultipartForm != nil {
		return nil
	}

	mr, err := r.multipartReader(false)
	if err != nil {
		return err
	}

	f, err := mr.ReadForm(maxMemory)
	if err != nil {
		return err
	}

	if r.PostForm == nil {
		r.PostForm = make(url.Values)
	}
	for k, v := range f.Value {
		r.Form[k] = append(r.Form[k], v...)
		// r.PostForm should also be populated. See Issue 9305.
		r.PostForm[k] = append(r.PostForm[k], v...)
	}

	r.MultipartForm = f

	return nil
}

//FormValue返回查询的命名组件的第一个值
// POST和PUT正文参数优先于URL查询字符串值。如果需要，.FormValue会调用ParseMultipartForm和ParseForm，并忽略这些函数返回的任何错误。如果key不存在，FormValue将返回空字符串。要访问同一个键的多个值，请调用ParseForm，然后直接检查Request.Form。
func (r *Request) FormValue(key string) string {
	if r.Form == nil {
		r.ParseMultipartForm(defaultMaxMemory)
	}
	if vs := r.Form[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// PostFormValue返回POST，PATCH或PUT请求主体的命名组件的第一个值。 URL查询参数被忽略.PostFormValue在必要时调用ParseMultipartForm和ParseForm并忽略这些函数返回的任何错误。如果key不存在，PostFormValue返回空字符串
func (r *Request) PostFormValue(key string) string {
	if r.PostForm == nil {
		r.ParseMultipartForm(defaultMaxMemory)
	}
	if vs := r.PostForm[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// FormFile返回提供的form键的第一个文件。
// 如有必要，FormFile会调用ParseMultipartForm和ParseForm。
func (r *Request) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	if r.MultipartForm == multipartByReader {
		return nil, nil, errors.New("http: multipart handled by MultipartReader")
	}
	if r.MultipartForm == nil {
		err := r.ParseMultipartForm(defaultMaxMemory)
		if err != nil {
			return nil, nil, err
		}
	}
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		if fhs := r.MultipartForm.File[key]; len(fhs) > 0 {
			f, err := fhs[0].Open()
			return f, fhs[0], err
		}
	}
	return nil, nil, ErrMissingFile
}
```


## 响应

Response的结构如下

```go
type Response struct {
	Status     string // 响应状态码，文本形式 e.g. "200 OK"
	StatusCode int    // 响应状态码，int形式 e.g. 200
	Proto      string // 所使用的协议版本 e.g. "HTTP/1.0"
	ProtoMajor int    // 所使用的协议大版本 e.g. 1
	ProtoMinor int    // 所使用的协议小版本 e.g. 0

	//Header将Header的键映射到值。如果响应具有相同键的多个标题，则可以使用commadelimiters连接它们。 （RFC 7230，第3.2.2节要求多个头在语义上等同于逗号分隔的序列。）当此结构中的其他字段（例如，ContentLength，TransferEncoding，Trailer）复制了Header值时，字段值是认证过的
	// map中的键是规范化的
	Header Header

	// Body代表响应体.
	//在读取Body字段时，响应主体按需流式传输。如果网络连接失败或服务器终止响应，Body.Read调用将返回错误
	//http客户端和传输保证Body始终为非零，即使在没有正文的响应或具有零长度正文的响应时也是如此。关闭Body是调用者的负责。如果Body未读取完成并关闭，则默认HTTP客户端的传输可能不会重用HTTP/1.x“保持活动”TCP连接。
	//如果服务器回复了“分块”Transfer-Encoding，则Body会自动取消。
	Body io.ReadCloser

	// ContentLength记录关联内容的长度。值-1表示长度未知。除非Request.Method是“HEAD”，否则值> = 0表示可以从Body读取给定的字节数。
	ContentLength int64

	// 包含从最外层到最内层的转移编码。值为nil，表示使用“identity”编码。
	TransferEncoding []string

	// 关闭记录读取Body后是否指向连接的Header。ReadResponse和Response.Write都不会关闭连接
	Close bool

	// 未压缩报告响应是否已发送压缩但由http包解压缩。如果为true，则从Body读取将生成未压缩的内容而不是从服务器实际设置的压缩内容，ContentLength设置为-1，并从responseHeader中删除“Content-Length”和“Content-Encoding”字段。要从服务器获取原始响应，请将Transport.DisableCompression设置为true
	Uncompressed bool

	// Trailer将Trailer键映射到与Header格式相同的值
	// Trailer最初只包含零值，一个用于服务器“Trailer”标题值中指定的每个键。这些值不会添加到标题中
	// 不得与Body上的Read调用同时访问预告片。
	// 在Body.Read返回io.EOF之后，Trailer将包含服务器发送的任何预告片值
	Trailer Header

	// 请求是为获取此响应而发送的请求。请求的正文为nil（已被使用）。这仅填充客户端请求。
	Request *Request

	// TLS包含有关收到响应的TLS连接的信息。对于未加密的响应，它是nil。指针在响应之间共享，不应修改
	TLS *tls.ConnectionState
}

// 相关方法
// Cookies()会解析并返回Set-Cookie标头中设置的Cookie
func (r *Response) Cookies() []*Cookie {
	return readSetCookies(r.Header)
}

// Location()返回响应的“位置”标题的URL（如果存在）。相对重定向相对于响应的请求进行解析。如果没有Location头，则返回ErrNoLocation
func (r *Response) Location() (*url.URL, error) {
	lv := r.Header.Get("Location")
	if lv == "" {
		return nil, ErrNoLocation
	}
	if r.Request != nil && r.Request.URL != nil {
		return r.Request.URL.Parse(lv)
	}
	return url.Parse(lv)
}

//ReadResponse从r读取并返回HTTP响应.req参数可选地指定与此Response对应的Request。如果为nil，则假定为GET请求。
// 从resp.Body读取完后，客户端必须调用resp.Body.Close
// 在该调用之后，客户端可以检查resp.Trailer以查找响应Trailer中包含的键/值对。
func ReadResponse(r *bufio.Reader, req *Request) (*Response, error) {
	tp := textproto.NewReader(r)
	resp := &Response{
		Request: req,
	}

	// 解析响应的第一行
	line, err := tp.ReadLine()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	// 读取http协议版本和响应状态
	if i := strings.IndexByte(line, ' '); i == -1 {
		return nil, &badStringError{"malformed HTTP response", line}
	} else {
		resp.Proto = line[:i]
		resp.Status = strings.TrimLeft(line[i+1:], " ")
	}
	// 获取响应状态码
	statusCode := resp.Status
	if i := strings.IndexByte(resp.Status, ' '); i != -1 {
		statusCode = resp.Status[:i]
	}
	// 如果返回的长度不为3，代表非法状态码
	if len(statusCode) != 3 {
		return nil, &badStringError{"malformed HTTP status code", statusCode}
	}
	// 将状态码进行转换
	resp.StatusCode, err = strconv.Atoi(statusCode)
	if err != nil || resp.StatusCode < 0 {
		return nil, &badStringError{"malformed HTTP status code", statusCode}
	}

	var ok bool
	// ParseHTTPVersion解析HTTP版本字符串。e.g “HTTP / 1.0”返回（1,0，true）
	if resp.ProtoMajor, resp.ProtoMinor, ok = ParseHTTPVersion(resp.Proto); !ok {
		return nil, &badStringError{"malformed HTTP version", resp.Proto}
	}

	// 解析响应头
	//ReadMIMEHeader从r读取MIME样式的头。Header是一系列可能连续的键：以空行结尾的值行。返回的映射m将CanonicalMIMEHeaderKey（key）映射到输入中遇到的相同顺序的值序列。
	//例如
	//	My-Key: Value 1
	//	Long-Key: Even
	//	       Longer Value
	//	My-Key: Value 2
	// 根据这个将会返回
	// map[string][]string{
	//	"My-Key": {"Value 1", "Value 2"},
	//	Long-Key": {"Even Longer Value"},
	//}
	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	// 将MIMEHeader进行转换
	resp.Header = Header(mimeHeader)
	// RFC 7234，第5.4节：应该是treatPragma：no-cache，如Cache-Control：no-cache
	fixPragmaCacheControl(resp.Header)

	// 将r转换为*Response 
	err = readTransfer(resp, r)
	if err != nil {
		return nil, err
	}

	return resp, nil
}


// 对应实现
// msg是 *Request 或者 *Response.
func readTransfer(msg interface{}, r *bufio.Reader) (err error) {
	t := &transferReader{RequestMethod: "GET"} //临时对象

	// Unify input
	isResponse := false
	switch rr := msg.(type) {
	case *Response:
		t.Header = rr.Header
		t.StatusCode = rr.StatusCode
		t.ProtoMajor = rr.ProtoMajor
		t.ProtoMinor = rr.ProtoMinor
		t.Close = shouldClose(t.ProtoMajor, t.ProtoMinor, t.Header, true) //确定在发送请求和正文后是挂起，还是接收响应，“header”是请求标头
		isResponse = true
		if rr.Request != nil {
			t.RequestMethod = rr.Request.Method
		}
	case *Request:
		t.Header = rr.Header
		t.RequestMethod = rr.Method
		t.ProtoMajor = rr.ProtoMajor
		t.ProtoMinor = rr.ProtoMinor
		// Transfer semantics for Requests are exactly like those for
		// Responses with status code 200, responding to a GET method
		t.StatusCode = 200
		t.Close = rr.Close
	default:
		panic("unexpected type")
	}

	// 如果没有相关相应值则默认使用 HTTP/1.1
	if t.ProtoMajor == 0 && t.ProtoMinor == 0 {
		t.ProtoMajor, t.ProtoMinor = 1, 1
	}

	// 传输编码，内容长度
	err = t.fixTransferEncoding() //修复Transfer编码，忽略HTTP / 1.0请求上的Transfer-Encoding。
	/*
	RFC 7230 3.3.2说“发送方不得在任何包含Transfer-Encoding标头字段的消息中发送Content-Length标头字段。”但是：“如果收到带有aTransfer-Encoding和Content-Length头字段的消息，则Transfer-Encoding会覆盖Content-Length。这样的消息可能表示尝试执行请求走私（第9.5节）或响应分裂（第9.4节）并且应该作为一个错误来处理。发送者必须在向下游转发这样的消息之前删除所接收的Content-Length字段。“据说，这些消息在非官方出现
	*/
	if err != nil {
		return err
	}
	//使用RFC 7230第3.3节确定预期的body长度。此函数不是方法，因为最终它应该由ReadResponse和ReadRequest共享。
	realLength, err := fixLength(isResponse, t.StatusCode, t.RequestMethod, t.Header, t.TransferEncoding)
	if err != nil {
		return err
	}
	// 如果响应Method为Head则解析 Content-Length字段
	if isResponse && t.RequestMethod == "HEAD" {
		if n, err := parseContentLength(t.Header.get("Content-Length")); err != nil {
			return err
		} else {
			t.ContentLength = n
		}
	} else {
		t.ContentLength = realLength
	}

	// 解析Trailer Header
	t.Trailer, err = fixTrailer(t.Header, t.TransferEncoding)
	if err != nil {
		return err
	}

	// 如果*Response中没有Content-Length或chunked Transfer-Encoding且状态不是1xx，204或304，那么正文是无界的。参见RFC 7230，第3.3节。
	switch msg.(type) {
	case *Response:
		if realLength == -1 &&
			!chunked(t.TransferEncoding) &&
			bodyAllowedForStatus(t.StatusCode) {
			// Unbounded body.
			t.Close = true
		}
	}

	// 准备Body Reader。 ContentLength < 0表示完成后的分块编码或关闭连接，因为尚不支持multipart
	switch {
	case chunked(t.TransferEncoding): //检查chunked是否是编码堆栈的一部分
		// Method为HEAD，或根据响应码不允许有body
		if noResponseBodyExpected(t.RequestMethod) || !bodyAllowedForStatus(t.StatusCode)//bodyAllowedForStatus报告给定的响应状态代码是否允许body。请参阅RFC 7230，第3.3节 {
			t.Body = NoBody
		} else {
			// body将Reader转换为ReadCloser.Close确保已完全读取正文，然后在必要时读取Trailer。
			t.Body = &body{src: internal.NewChunkedReader(r), hdr: msg, r: r, closing: t.Close}
		}
	case realLength == 0: // 如果没有长度则没有body
		t.Body = NoBody
	case realLength > 0: 
		t.Body = &body{src: io.LimitReader(r, realLength), closing: t.Close}
	default:
		// realLength <0，即Header中没有“Content-Length”
		if t.Close {
			// 关闭语义 (i.e. HTTP/1.0)
			t.Body = &body{src: r, closing: t.Close}
		} else {
			// 持续链接 (i.e. HTTP/1.1)
			t.Body = NoBody
		}
	}

	// 统一处理
	switch rr := msg.(type) {
	case *Request:
		rr.Body = t.Body
		rr.ContentLength = t.ContentLength
		rr.TransferEncoding = t.TransferEncoding
		rr.Close = t.Close
		rr.Trailer = t.Trailer
	case *Response:
		rr.Body = t.Body
		rr.ContentLength = t.ContentLength
		rr.TransferEncoding = t.TransferEncoding
		rr.Close = t.Close
		rr.Trailer = t.Trailer
	}

	return nil
}

```

### Cookie

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
