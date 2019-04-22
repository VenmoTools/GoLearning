# gin启动流程

Gin启动流程比beego流程简易很多

启动步骤如下
1. 通过engine.New方法实例化Engine实例，包括template，context，路由树等
2. 使用Use方法注册middleware
3. 通过Run方法 直接调用http.ListenAndServe启动服务器


## 主要结构

```go
// Engine 框架实例，包含多路复用器，middleware，以及配置
// 创建Engine实例可以使用New或者Default函数，Default函数默认会使用Logger()和Recvory()
type Engine struct {
	// RouterGroup在内部用于配置路由器，RouterGroup与前缀和处理程序数组（中间件）相关联
	RouterGroup
	// 如果当前路由无法匹配，则启用自动重定向，但存在带（不带）尾部斜杠的路径的处理程序。例如，如果请求/ foo /但仅存在/ foo的路由，则客户端将重定向到/ foo，其中对于GET请求，http状态代码为301;对于所有其他请求方法，则为307。
	// 默认开启
	RedirectTrailingSlash bool

	// 如果启用，路由器将尝试修复当前请求路径，如果没有为其注册handler。删除第一个多余的路径元素，如../或//。然后，路由器对已处理的路径进行查找（不区分大小写的）。如果可以找到此路由的handler，则路由器将重定向到更正的路径，其中GET请求的状态代码为301，所有其他请求方法的状态代码为307。
	// 例如/ FOO和/..//Foo可以重定向到/ foo。
	// 默认关闭
	RedirectFixedPath bool

	// 如果启用，路由器将检查当前路由是否允许另一种method，如果当前请求无法路由。则使用“Method not allow”和HTTP状态代码405来响应。如果不允许其他方法，则将请求委托给NotFound处理程序
	// 默认关闭
	HandleMethodNotAllowed bool
	// 默认开启
	ForwardedByClientIP    bool

	// #726 #755 如果启用 将会在hander发送以
	// 'X-AppEngine...'开头的header 以便更好地与PaaS集成。
	AppEngine bool

	// 如果启用，url.RawPath将用于查找参数。
	UseRawPath bool

	// 如果为true，则路径值将被取消转义。
    //如果UseRawPath为false（默认情况下），UnescapePathValues实际上是正确的，因为将使用url.Path，它已经未转义
	UnescapePathValues bool

	// 给予http.Request的ParseMultipartForm方法调用的'maxMemory'参数值。
	// 默认32 MB
	MaxMultipartMemory int64
	// 模板渲染的引用符号默认是双花括号( {{.Header}})
	delims           render.Delims
	secureJsonPrefix string
	HTMLRender       render.HTMLRender
	FuncMap          template.FuncMap
	allNoRoute       HandlersChain
	allNoMethod      HandlersChain
	noRoute          HandlersChain
	noMethod         HandlersChain

	// 用于对context对象的复用
	pool             sync.Pool
	// trees
	trees            methodTrees
}
```

## 初始化
```go
func New() *Engine {
	debugPrintWARNINGNew()
	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
		FuncMap:                template.FuncMap{},
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      false,
		HandleMethodNotAllowed: false,
		ForwardedByClientIP:    true,
		AppEngine:              defaultAppEngine,
		UseRawPath:             false,
		UnescapePathValues:     true,
		MaxMultipartMemory:     defaultMultipartMemory,
		trees:                  make(methodTrees, 0, 9),
		delims:                 render.Delims{Left: "{{", Right: "}}"},
		secureJsonPrefix:       "while(1);",
	}

	engine.RouterGroup.engine = engine
	engine.pool.New = func() interface{} {
		// 实例化context
		return engine.allocateContext()
	}
	return engine
}

```

# 请求处理

1. 当请求过来后首先会调用ServeHTTP方法，从pool中取出context对象，根据ResponseWriter对writermem中的responseWriter进行初始化，重置context中的内容，调用handleHTTPRequest处理请求
2. 如果在engine开启了UseRawPath，处理的url为Request.URL.RawPath
3. 在trees中查找符合的请求Method root，然后使用getValue返回所有符合条件的HandlerFunc(handlers起始是一个HandlerFunc切片)
4. 如果找到handler，进行赋值，返回200状态码
5. 如果请求不为connect并且不为跟路径并且在engine开启了RedirectTrailingSlash，则调用 redirectTrailingSlash，如果开启了RedirectFixedPath则调用redirectFixedPath，进行重定向
6. 当重定向无法处理的时候并且开启了HandleMethodNotAllowed，将在路由树中查找请求的method，如果找到则使用serveError返回StatusMethodNotAllowed错误
7. 以上的都没有处理成功则返回404错误

## ServeHTTP
engine实现了ServeHTTP接口也是属于HTTP Handler

```go
// ServeHTTP conforms to the http.Handler interface.
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    // sync.Pool 用于对象的复用
	c := engine.pool.Get().(*Context)
	
    // 初始化 responseWriter，responseWriter是对http.ResponseWriter的一层封装
	c.writermem.reset(w)

    c.Request = req
    // 重置Context内容
	c.reset()

    // handleHTTPRequest来处理请求
	engine.handleHTTPRequest(c)

	engine.pool.Put(c)
}
```


## handleHTTPRequest

```go
func (engine *Engine) handleHTTPRequest(c *Context) {
    //获取请求Method和Path
	httpMethod := c.Request.Method
	rPath := c.Request.URL.Path
	unescape := false
	// 如果开启了UseRawPath
	if engine.UseRawPath && len(c.Request.URL.RawPath) > 0 {
		rPath = c.Request.URL.RawPath
		unescape = engine.UnescapePathValues
	}

    //找到符合HTTP Method的tree root
	t := engine.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method != httpMethod {
			continue
        }
		root := t[i].root
        // Find route in tree
		// 在树中找到路由信息
		// getValue返回使用给定路径（k​​ey）注册的handler。通配符的值将保存到map中。
		// 如果找不到handler，则建议进行TSR（尾部斜杠重定向）
		handlers, params, tsr := root.getValue(rPath, c.Params, unescape)
		if handlers != nil {
			// handler赋值
			c.handlers = handlers
			c.Params = params
			// Next应该只在中间件中使用。它在调用处理程序内的链中执行挂起的处理程序
			c.Next()
			// 返回200状态码
			c.writermem.WriteHeaderNow()
			return
		}
		// 
		if httpMethod != "CONNECT" && rPath != "/" {
			if tsr && engine.RedirectTrailingSlash {
				redirectTrailingSlash(c)
				return
			}
			if engine.RedirectFixedPath && redirectFixedPath(c, root, engine.RedirectFixedPath) {
				return
			}
		}
		break
	}
	// 如果启用了HandleMethodNotAllowed
	if engine.HandleMethodNotAllowed {
		// 循环在路由树中查找符合的Method
		for _, tree := range engine.trees {
			if tree.method == httpMethod {
				continue
			}
			// 如果找到则使用serveError返回StatusMethodNotAllowed错误
			if handlers, _, _ := tree.root.getValue(rPath, nil, unescape); handlers != nil {
				c.handlers = engine.allNoMethod
				serveError(c, http.StatusMethodNotAllowed, default405Body)
				return
			}
		}
	}
	// 如果没有找到符合的路由则使用allNoRoute，处理错误
	c.handlers = engine.allNoRoute
	serveError(c, http.StatusNotFound, default404Body)
}

```