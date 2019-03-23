# go服务器
## Server

### 使用

简单的启动
```go
import(
    "net/http"
)

func main(){
    http.ListenAndServe("",nil)
}
```

带有配置的启动

```go
import(
    "net/http"
)

func main() {
	server := http.Server{
		Addr:"127.0.0.1",
		Handler:nil,
	}
	server.ListenAndServe()
}
```

```go
type Server struct {
	Addr    string  // TCP监听的链接地址，如果为空则默认为":http" 
	Handler Handler // 使用的多路复用器, 如果为nil则使用http.DefaultServeMux

	// TLSConfig可选择提供ServeTLS和ListenAndServeTLS使用的TLS配置。请注意，此值由ServeTLS和ListenAndServeTLS克隆，因此无法使用tls.Config.SetSessionTicketKeys等方法修改配置。要使用SetSessionTicketKeys，请使用带有TLS侦听器的Server.Serve。
	TLSConfig *tls.Config

	// ReadTimeout是读取整个请求的最长持续时间，包括正文。因为ReadTimeout不允许处理程序根据每个请求正文的可接受截止日期或每个请求做出请求上传速率，大多数用户更愿意使用ReadHeaderTimeout。使用它们都是有效的。
	ReadTimeout time.Duration

	// ReadHeaderTimeout是允许读取请求头的时间量。读取标题后，连接的读取截止日期将被重置，处理程序可以决定，对body来说太慢的内容。
	ReadHeaderTimeout time.Duration

	// WriteTimeout是超时写入响应之前的最大持续时间。只要读取新请求头，它就会重置。与ReadTimeout一样，它不允许处理程序基于每个请求做出决定。
	WriteTimeout time.Duration

	// IdleTimeout是启用keep-alives时等待下一个请求的最长时间。如果IdleTimeout为零，则使用ReadTimeout的值。如果两者都为零，则使用ReadHeaderTimeout。
	IdleTimeout time.Duration

	// MaxHeaderBytes控制服务器读取的最大字节数，解析请求标头的键和值，包括请求行。它不限制请求体的大小。如果为零，则使用DefaultMaxHeaderBytes。
	MaxHeaderBytes int

	// TLSNextProto可选地指定在发生NPN / ALPN协议升级时接管所提供的TLS连接的所有权的函数。Map的key是协议名称。 Handler参数应该用于处理HTTP请求，如果尚未设置，则会初始化Request的TLS和RemoteAddr。函数返回时，连接自动关闭。如果TLSNextProto不是nil，HTTP / 2不会自动启用。
	TLSNextProto map[string]func(*Server, *tls.Conn, Handler)

	// ConnState指定在客户端连接更改状态时调用的可选回调函数
	ConnState func(net.Conn, ConnState)

	// ErrorLog为接受连接，处理程序的意外行为和基础FileSystem错误的错误指定了一个可选的记录器。如果为nil，则通过日志包的标准记录器完成日志记录。
	ErrorLog *log.Logger

	disableKeepAlives int32     // 原子访问.
	inShutdown        int32     // 原子地访问（非零处于关闭状态）
	nextProtoOnce     sync.Once // guards setupHTTP2_* init
	nextProtoErr      error     //  http2.ConfigureServer的结果（如果使用的话）

	mu         sync.Mutex
	listeners  map[*net.Listener]struct{} // 存储实现Listener接口的对象
	activeConn map[*conn]struct{} // 正在使用的链接
	doneChan   chan struct{}
	onShutdown []func()
}
//公开方法

// 关闭会立即关闭所有活动的net.Listeners以及状态StateNew，StateActive或StateIdle中的所有连接。要正常关闭，请使用Shutdown。
// Close不会尝试关闭（甚至不知道）任何被劫持的连接，例如WebSockets。
// Close返回关闭Server的底层Listener返回的任何错误。
func (srv *Server) Close() error {
	atomic.StoreInt32(&srv.inShutdown, 1) // 更改服务器状态值
	srv.mu.Lock()
	defer srv.mu.Unlock()
	srv.closeDoneChanLocked() 
	err := srv.closeListenersLocked() // 关闭所有的Listener
	for c := range srv.activeConn { // 关闭所有的activeConn
		c.rwc.Close()
		delete(srv.activeConn, c)
	}
	return err
}

//closeListenersLocked函数，定义如下
func (s *Server) closeListenersLocked() error {
	var err error
	for ln := range s.listeners { // 	listeners 是一个map，利用mapkey去重，存储实现Listener接口的对象
		if cerr := (*ln).Close(); cerr != nil && err == nil { // 关闭所有的listener
			err = cerr
		}
		delete(s.listeners, ln)
	}
	return err
}
//shutdownPollInterval是我们在Server.Shutdown期间轮询静态的频率。这在测试期间较低，以加快测试速度。理想情况下，我们可以找到一个不涉及轮询的解决方案，但也没有很高的运行时成本（并且没有涉及任何有争议的互斥体），但这仍然是读者的练习
var shutdownPollInterval = 500 * time.Millisecond

// Shutdown正常关闭服务器而不会中断任何活动连接。关闭工作首先关闭所有打开的侦听器，然后关闭所有空闲连接，然后无限期地等待连接返回空闲然后关闭
// 如果提供的上下文在关闭完成之前到期，则Shutdown返回上下文的错误，否则它将返回关闭Server的基础Listener返回的任何错误。
// 调用Shutdown时，Serve，ListenAndServe和ListenAndServeTLS会立即返回ErrServerClosed。确保程序没有退出并等待Shutdown返回
// Shutdown不会尝试关闭也不会等待WebSockets等被劫持的连接。如果需要，Shutdown的调用者应单独通知关闭的长期连接并等待它们关闭。有关注册关闭通知功能的方法
// 一旦在服务器上调用Shutdown，它就可能无法重用;以后对Serve等方法的调用将返回ErrServerClosed
func (srv *Server) Shutdown(ctx context.Context) error {
	atomic.StoreInt32(&srv.inShutdown, 1)

	srv.mu.Lock()
	lnerr := srv.closeListenersLocked()
	srv.closeDoneChanLocked()
	for _, f := range srv.onShutdown { // 调用注册的onShutdown函数
		go f()
	}
	srv.mu.Unlock()

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if srv.closeIdleConns() {
			return lnerr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// RegisterOnShutdown注册一个函数以在Shutdown上调用。这可用于正常关闭已经过NPN / ALPN协议升级或已被劫持的连接。此函数应启动特定于协议的正常关闭，但不应等待关闭完成。
func (srv *Server) RegisterOnShutdown(f func()) {
	srv.mu.Lock()
	srv.onShutdown = append(srv.onShutdown, f)
	srv.mu.Unlock()
}
//ListenAndServe侦听TCP网络地址srv.Addr，然后调用Serve来处理传入连接上的请求。接受连接配置为启用TCP保持活动。
//如果srv.Addr为空，则使用“:http”。
// ListenAndServe始终返回非零错误。关闭或关闭后，返回的错误是ErrServerClosed。
func (srv *Server) ListenAndServe() error {
	if srv.shuttingDown() { // 判断服务器是否处于关闭状态
		return ErrServerClosed
	}
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr) // 启动tcp服务器
	if err != nil {
		return err
	}
	return srv.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
}

//tcpKeepAliveListener在接受的连接上设置TCP保持活动超时。它由ListenAndServe和ListenAndServeTLS使用，因此dead的TCP连接（例如关闭下载中的笔记本电脑）最终会消失。
type tcpKeepAliveListener struct {
	*net.TCPListener //TCPListener是TCP网络侦听器。客户端通常应使用Listener类型的变量而不是假设TCP。
}

//Serve接受Listener l上的传入连接，为每个连接创建一个新的服务goroutine。服务goroutines读取请求，然后调用srv.Handler来接受它们
//仅当侦听器返回* tls.Conn连接并且在TLS Config.NextProtos中将它们配置为“h2”时，才会启用HTTP / 2支持。
// Serve总是返回一个非零错误并关闭 l
// 关闭或关闭后，返回的错误是ErrServerClosed
func (srv *Server) Serve(l net.Listener) error {
	if fn := testHookServerServe; fn != nil {
		fn(srv, l) // 用unwrapped监听器调用钩子
	}

	l = &onceCloseListener{Listener: l}
	defer l.Close()

	if err := srv.setupHTTP2_Serve(); err != nil {
		return err
	}

	if !srv.trackListener(&l, true) {
		return ErrServerClosed
	}
	defer srv.trackListener(&l, false)

	var tempDelay time.Duration     // how long to sleep on accept failure
	baseCtx := context.Background() // base is always background, per Issue 16220
	ctx := context.WithValue(baseCtx, ServerContextKey, srv)
	for {
		rw, e := l.Accept()
		if e != nil {
			select {
			case <-srv.getDoneChan():
				return ErrServerClosed
			default:
			}
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				srv.logf("http: Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		c := srv.newConn(rw)
		c.setState(c.rwc, StateNew) // before Serve can return
		go c.serve(ctx)
	}
}


// ServeTLS accepts incoming connections on the Listener l, creating a
// new service goroutine for each. The service goroutines perform TLS
// setup and then read requests, calling srv.Handler to reply to them.
//
// Files containing a certificate and matching private key for the
// server must be provided if neither the Server's
// TLSConfig.Certificates nor TLSConfig.GetCertificate are populated.
// If the certificate is signed by a certificate authority, the
// certFile should be the concatenation of the server's certificate,
// any intermediates, and the CA's certificate.
//
// ServeTLS always returns a non-nil error. After Shutdown or Close, the
// returned error is ErrServerClosed.
func (srv *Server) ServeTLS(l net.Listener, certFile, keyFile string) error {
	// Setup HTTP/2 before srv.Serve, to initialize srv.TLSConfig
	// before we clone it and create the TLS Listener.
	if err := srv.setupHTTP2_ServeTLS(); err != nil {
		return err
	}

	config := cloneTLSConfig(srv.TLSConfig)
	if !strSliceContains(config.NextProtos, "http/1.1") {
		config.NextProtos = append(config.NextProtos, "http/1.1")
	}

	configHasCert := len(config.Certificates) > 0 || config.GetCertificate != nil
	if !configHasCert || certFile != "" || keyFile != "" {
		var err error
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return err
		}
	}

	tlsListener := tls.NewListener(l, config)
	return srv.Serve(tlsListener)
}

// SetKeepAlivesEnabled controls whether HTTP keep-alives are enabled.
// By default, keep-alives are always enabled. Only very
// resource-constrained environments or servers in the process of
// shutting down should disable them.
func (srv *Server) SetKeepAlivesEnabled(v bool) {
	if v {
		atomic.StoreInt32(&srv.disableKeepAlives, 0)
		return
	}
	atomic.StoreInt32(&srv.disableKeepAlives, 1)

	// Close idle HTTP/1 conns:
	srv.closeIdleConns()

	// TODO: Issue 26303: close HTTP/2 conns as soon as they become idle.
}

// ListenAndServe listens on the TCP network address addr and then calls
// Serve with handler to handle requests on incoming connections.
// Accepted connections are configured to enable TCP keep-alives.
//
// The handler is typically nil, in which case the DefaultServeMux is used.
//
// ListenAndServe always returns a non-nil error.
func ListenAndServe(addr string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

// ListenAndServeTLS acts identically to ListenAndServe, except that it
// expects HTTPS connections. Additionally, files containing a certificate and
// matching private key for the server must be provided. If the certificate
// is signed by a certificate authority, the certFile should be the concatenation
// of the server's certificate, any intermediates, and the CA's certificate.
func ListenAndServeTLS(addr, certFile, keyFile string, handler Handler) error {
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServeTLS(certFile, keyFile)
}

```

## 处理函数
服务器接受请求后需要寻找处理器来处理请求，只要符合ServeHTTP(http.ResponseWriter,*http.Request)函数签名便是一个处理器

当ListenAndServe函数的第二个参数为nil时默认使用DefaultServeMux,DefaultServeMux即是ServeMux结构的实例，也是Handler的实例，DefaultServeMux也是一个处理器，它根据请求的URL将请求重定向到不同的处理器

### ServeMux
ServeMux是一个HTTP请求多路复用器。它将每个传入请求的URL与已注册模式的列表进行匹配，并调用与该URL最匹配的模式的处理程序。

模式名称固定，有根路径，如“/favicon.ico”，或有根的子树，如“/images/”（请注意尾部斜杠）。较长的模式优先于较短的模式，因此如果有“/images/”和“/images/thumbnails/”注册的处理程序，后面的处理程序将被调用以开始“/images/thumbnails/”的路径和前者将收到“/images/”子树中任何其他路径的请求。请注意，由于以斜杠结尾的模式命名为带根子树，因此模式“/”匹配所有与其他已注册模式不匹配的路径，而不仅仅是具有Path ==“/”的URL。

如果已注册子树并且收到命名子树根但没有其斜杠的请求，则ServeMux会将该请求重定向到子树根（添加尾部斜杠）。可以通过单独注册没有尾部斜杠的路径来覆盖此行为。例如，注册“/ images /”会导致ServeMux将“/ images”请求重定向到“/images/”，除非“/images”已单独注册。

模式可以选择以主机名开头，仅限制与该主机上的URL匹配。特定于主机的模式优先于一般模式，因此处理程序可以注册两个模式“/codesearch”和“codesearch.google.com/”，而无需接管“http://www.google.com/”的请求”。

ServeMux还负责清理URL请求路径和主机头，split端口号并重定向包含的任何请求。例如.或..，重复斜杠，使URL更明确

`假设有indexHandler绑定Path为"/"，imageHandler绑定path为“/images”,注意尾部没有斜杠，当请求Path为"/images/thumbnails/"则将会使用indexHandler处理器进行处理，因为绑定imageHandler的Path为/image而不是/image/，如果绑定的Path不是以"/"结尾，只会与完全相同的Path取匹配，如果Path以"/"结尾，请求的Path只有前缀部分与注册的相同，也会认为两个Path匹配`

ServeMux结构如下
```go
type ServeMux struct {
	mu    sync.RWMutex //读写锁
	m     map[string]muxEntry // key存储path，muxEntry包含对应模式和响应的处理函数
	hosts bool // 模式中是否包含主机名
}

type muxEntry struct {
	h       Handler
	pattern string
}
// 根据给定的path在handler Map中查找，返回符合特定的或者最长的模式
func (mux *ServeMux) match(path string) (h Handler, pattern string) {
	// 首先检查完全匹配
	v, ok := mux.m[path]
	if ok {
		return v.h, v.pattern
	}

	// 检查最长的有效匹配
	var n = 0
	for k, v := range mux.m {
		if !pathMatch(k, path) { //判断path是否匹配
			continue
		}
		if h == nil || len(k) > n {
			n = len(k)
			h = v.h
			pattern = v.pattern
		}
	}
	return
}
//Handler返回用于给定请求的处理程序，请参阅r.Method，r.Host和r.URL.Path。它总是返回一个非零处理程序。如果路径不是规范形式，则处理程序将是内部生成的处理程序，重定向到规范路径。如果主机包含端口，则在匹配处理程序时将忽略该端口。
//对于CONNECT请求，路径和主机未经更改使用。
// Handler还返回与请求匹配的已注册模式，或者在内部生成的重定向的情况下，返回跟随重定向后匹配的模式
// 如果没有适用于请求的已注册处理程序，
func (mux *ServeMux) Handler(r *Request) (h Handler, pattern string) {

	// CONNECT请求不是规范化的
	if r.Method == "CONNECT" {
		// 如果r.URL.Path是/tree并且其处理程序未注册，则/tree  - > /tree/ redirect应用于CONNECT请求，但路径规范化不适用.
		if u, ok := mux.redirectToPathSlash(r.URL.Host, r.URL.Path, r.URL); ok {
			return RedirectHandler(u.String(), StatusMovedPermanently), u.Path
		}

		return mux.handler(r.Host, r.URL.Path)
	}

	//在传递给mux.handler之前，所有其他请求都strip了任何端口并清除了路径.
	host := stripHostPort(r.Host)
	path := cleanPath(r.URL.Path)

	// 如果给定路径是/tree且其处理程序未注册，则重定向到/tree/。
	if u, ok := mux.redirectToPathSlash(host, path, r.URL); ok {
		return RedirectHandler(u.String(), StatusMovedPermanently), u.Path
	}

	if path != r.URL.Path {
		_, pattern = mux.handler(host, path)
		url := *r.URL
		url.Path = path
		return RedirectHandler(url.String(), StatusMovedPermanently), pattern
	}

	return mux.handler(host, r.URL.Path)
}
// handler是Handler的主要实现。除了CONNECT方法之外，已知路径是规范形式。
func (mux *ServeMux) handler(host, path string) (h Handler, pattern string) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	// 特定于主机的模式优先于通用模式
	if mux.hosts {
		h, pattern = mux.match(host + path)
	}
	if h == nil {
		h, pattern = mux.match(path)
	}
	if h == nil {
		h, pattern = NotFoundHandler(), ""
	}
	return
}
//ServeHTTP将请求分派给处理程序，该处理程序的模式与请求URL最匹配
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(StatusBadRequest)
		return
	}
	h, _ := mux.Handler(r) // 调用Handler方法返回对应的Handler
	h.ServeHTTP(w, r) // 调用ServeHTTP处理请求
}

//Handle注册给定模式的处理程序。如果模式已存在处理程序，则Handler panic
func (mux *ServeMux) Handle(pattern string, handler Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
    // 检查
	if pattern == "" {
		panic("http: invalid pattern")
	}
	if handler == nil {
		panic("http: nil handler")
	}
	if _, exist := mux.m[pattern]; exist {
		panic("http: multiple registrations for " + pattern)
	}

	if mux.m == nil {
		mux.m = make(map[string]muxEntry)
    }
    // 注册
	mux.m[pattern] = muxEntry{h: handler, pattern: pattern}

	if pattern[0] != '/' {
		mux.hosts = true
	}
}

// HandleFunc为给定模式注册处理函数。
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	mux.Handle(pattern, HandlerFunc(handler))
}
//Handle在DefaultServeMux中注册给定模式的处理程序
// var defaultServeMux ServeMux 
// var DefaultServeMux = &defaultServeMux 全局DefaultServeMux
func Handle(pattern string, handler Handler) { DefaultServeMux.Handle(pattern, handler) }
```

### 单个处理器

```go
type TestHander struct {
}

func (t *TestHander) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello")
}

func main() {
	testHander := TestHander{}
	server := http.Server{
		Addr:    "127.0.0.1",
		Handler: &testHander,
	}
	server.ListenAndServe()
}
```

### 多个处理器

```go
type TestHander struct {
}

func (t *TestHander) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello")
}

type OtherTestHander struct {
}

func (t *OtherTestHander) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello i am other one")
}

func main() {
	testHander := TestHander{}
	ohterHander := OtherTestHander{}

	server := http.Server{
		Addr:    "127.0.0.1",
    }
    // 因为默认使用DefaultServeMux，并且DefaultServeMux已经实例化直接注册即可
	http.Handle("/hello",&testHander)
	http.Handle("/other",&ohterHander)
	server.ListenAndServe()
}
```

### 处理函数

```go
func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello")
}

type OtherTestHander struct {
}

func (t *OtherTestHander) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello i am other one")
}

func main() {
	ohterHander := OtherTestHander{}

	server := http.Server{
		Addr: "127.0.0.1",
	}
	http.HandleFunc("/hello",hello)
	http.Handle("/other", &ohterHander)
	server.ListenAndServe()
}

```

### 串联多个处理函数
例如日志，安全检查和错误处理这样的操作通常成为横切关注点，为了防止代码重复操作和代码依赖问题可以使用串联分割代码中的横切关注点

```go

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello")
}

func log(h http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		name := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
		fmt.Println("Handler function called ->", name)
		h(writer, request)
	}
}

func main() {
	server := http.Server{
		Addr: "127.0.0.1:8080",
	}
	http.HandleFunc("/hello", log(hello))
	err := server.ListenAndServe()
	fmt.Println(err)
}

```

串联多个函数可以让程序执行更多动作，这种方法也称为管道处理

### 串联处理器

```go

func log(h http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		name := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
		fmt.Println("Handler function called ->", name)
		h.ServeHTTP(writer, request)
	})
}

type HelloHandler struct {
}

func (h HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "hello")
}

func protect(h http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("checking name...")
		fmt.Println("checking password...")
		h.ServeHTTP(writer, request)
	})
}

func main() {
	hello := HelloHandler{}
	server := http.Server{
		Addr: "127.0.0.1:8080",
	}
	http.Handle("/hello", protect(log(hello)))
	err := server.ListenAndServe()
	fmt.Println(err)
}
```

## 使用其他多路复用器
