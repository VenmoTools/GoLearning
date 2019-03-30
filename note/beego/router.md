# beego多路复用器
在beego/app.go文件中有一个为BeeApp的结构，该结构主要是对多路复用器以及其他的一层封装

router是beego实现的一个多路复用器，提供了很多注册处理器方式

## 变量

```go

const (
	BeforeStatic = iota
	BeforeRouter
	BeforeExec
	AfterExec
	FinishRouter
)

const (
	routerTypeBeego = iota
	routerTypeRESTFul
	routerTypeHandler
)

var (
    // http请求支持的method映射
	HTTPMETHOD = map[string]bool{
		"GET":       true,
		"POST":      true,
		"PUT":       true,
		"DELETE":    true,
		"PATCH":     true,
		"OPTIONS":   true,
		"HEAD":      true,
		"TRACE":     true,
		"CONNECT":   true,
		"MKCOL":     true,
		"COPY":      true,
		"MOVE":      true,
		"PROPFIND":  true,
		"PROPPATCH": true,
		"LOCK":      true,
		"UNLOCK":    true,
	}
    
	// beego.Controller's 所有的方法集合
	// 这些放回不能用作AutoRouter
	exceptMethod = []string{
        "Init", "Prepare", "Finish", "Render", "RenderString",
		"RenderBytes", "Redirect", "Abort", "StopRun", "UrlFor", "ServeJSON", "ServeJSONP",
		"ServeYAML", "ServeXML", "Input", "ParseForm", "GetString", "GetStrings", "GetInt", "GetBool",
		"GetFloat", "GetFile", "SaveToFile", "StartSession", "SetSession", "GetSession",
		"DelSession", "SessionRegenerateID", "DestroySession", "IsAjax", "GetSecureCookie",
		"SetSecureCookie", "XsrfToken", "CheckXsrfCookie", "XsrfFormHtml",
		"GetControllerAndAction", "ServeFormatted"}


	urlPlaceholder = "{{placeholder}}"
    // DefaultAccessLogFilter will skip the accesslog if return true
    // 默认的log过滤，如果返回true则跳过成功的log
	DefaultAccessLogFilter FilterHandler = &logFilter{}
)

```
## 结构
```go
// ControllerInfo用于保存controller的信息
type ControllerInfo struct {
	pattern        string   // path
	controllerType reflect.Type // 
	methods        map[string]string // 请求所支持的方法 例如只能发送post或者get
	handler        http.Handler //
	runFunction    FilterFunc //
	routerType     int // 
	initialize     func() ControllerInterface
	methodParams   []*param.MethodParam
}

// ControllerRegister为一个多路复用器
type ControllerRegister struct {
	routers      map[string]*Tree // beego实现了一个树结构用于存储path的路径
	enablePolicy bool //启用代理
	policies     map[string]*Tree
	enableFilter bool //启用过滤
	filters      [FinishRouter + 1][]*FilterRouter
	pool         sync.Pool // 用于存储beego.Context，对象复用
}
```

### ServeHTTP方法
ServeHTTP将请求分派给处理程序

基本流程如下：
1. 从pool中取得Context并传递ResponseWriter和Request进行重置重置(初始化）
2. 根据config中的配置内容进行设置(EnableGzip，RunMode,根据RouterCaseSensitive处理请求url)
3. 如果客户端请求的Method不符合在跳转到 [18]
4. 在处理static file之前调用BeforeStaticFilterRouter跳转到 [18]
5. 调用serverStaticRouter，来处理static file
6. 判断Response是否已经其他处理器已经写入，则不会调用其他的处理器，如果已经写入将调转到 [18]
7. 如果请求不为GET和HEAD，会根据配置是否保存请求Body，最后调用ParseFormOrMulitForm来处理请求
8. 如果配置中启用session将会初始化session
9. 调用BeforeRouterFilterRouter
10. 如果用户在filter中RunController和RunMethod否则调用FindRouter来获取
11. 如果没有找到请求path所对应的Handler(通过标志位findRouter)则会exception("404", context)来跳转到404页面
12. 尝试获取splat，如果获取到则使用SetParam来添加
13. 检查policies，如果通过跳转到 [18]
14. 根据routerInfo.routerType判断使用的处理器，如果是routerTypeRESTFul，如果请求的path和handler符合的Method则调用runFunction函数，否则会返回405错误，如果是routerTypeHandler，则会调用handler的ServeHTTP方法，如果都不是则获取MethodPost或MethodDelete并判断处理函数是否接受，如果没有设置判断有无"*"
15. 通过isRunnable判断是否已经处理了该请求，如果routerInfo已经获取到并且有响应的初始化函数，将会routerInfo.initialize()进行初始化，否则将runRouter尝试强转为ControllerInterface(请求处理器)，随后调用Init方法初始化，调用Prepare，如果启用了XSRF则将会通过CheckXSRFCookie()检查cookie，调用URLMapping,随后根据请求的Method调用对应的方法，例如Post方法会调用exeController.Post()如果是自定义的通过HandlerFunc找到对应的处理函数然后通过反射调用，如果methodParams不为空则调用handleParamResponse最后如果配置了AutoRender会调用Render()随后调用Finish完成并释放资源
16. 调用AfterExecFilterRouter
17. 调用FinishRouterFilterRouter
18. 到此步骤都是使用goto Admin来到的，Admin主要用于记录qps，以及请求时间等内容(如果在配置文件中启用的话)
 
ServeHTTP源码如下
```go
func (p *ControllerRegister) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	startTime := time.Now() //处理开始时间
	var (
		runRouter    reflect.Type
		findRouter   bool
		runMethod    string
		methodParams []*param.MethodParam
		routerInfo   *ControllerInfo
		isRunnable   bool
	)

	context := p.pool.Get().(*beecontext.Context) // 获取Context
	// 初始化context.Context, BeegoInput and BeegoOutput
	context.Reset(rw, r)

	defer p.pool.Put(context) //处理完后归还对象
	if BConfig.RecoverFunc != nil {
		defer BConfig.RecoverFunc(context) // panic处理函数
	}
	
	context.Output.EnableGzip = BConfig.EnableGzip

	if BConfig.RunMode == DEV {
		context.Output.Header("Server", BConfig.ServerName)
	}

	var urlPath = r.URL.Path

	if !BConfig.RouterCaseSensitive {
		urlPath = strings.ToLower(urlPath)
	}

	// filter wrong http method
	if !HTTPMETHOD[r.Method] {
		http.Error(rw, "Method Not Allowed", 405)
		goto Admin
	}

	// filter for static file
	if len(p.filters[BeforeStatic]) > 0 && p.execFilter(context, urlPath, BeforeStatic) {
		goto Admin
	}

	serverStaticRouter(context)

	if context.ResponseWriter.Started {
		findRouter = true
		goto Admin
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if BConfig.CopyRequestBody && !context.Input.IsUpload() {
			context.Input.CopyBody(BConfig.MaxMemory)
		}
		context.Input.ParseFormOrMulitForm(BConfig.MaxMemory)
	}

	// session init
	if BConfig.WebConfig.Session.SessionOn {
		var err error
		context.Input.CruSession, err = GlobalSessions.SessionStart(rw, r)
		if err != nil {
			logs.Error(err)
			exception("503", context)
			goto Admin
		}
		defer func() {
			if context.Input.CruSession != nil {
				context.Input.CruSession.SessionRelease(rw)
			}
		}()
	}
	if len(p.filters[BeforeRouter]) > 0 && p.execFilter(context, urlPath, BeforeRouter) {
		goto Admin
	}
	// User can define RunController and RunMethod in filter
	if context.Input.RunController != nil && context.Input.RunMethod != "" {
		findRouter = true
		runMethod = context.Input.RunMethod
		runRouter = context.Input.RunController
	} else {
		routerInfo, findRouter = p.FindRouter(context)
	}

	//if no matches to url, throw a not found exception
	if !findRouter {
		exception("404", context)
		goto Admin
	}
	if splat := context.Input.Param(":splat"); splat != "" {
		for k, v := range strings.Split(splat, "/") {
			context.Input.SetParam(strconv.Itoa(k), v)
		}
	}

	//execute middleware filters
	if len(p.filters[BeforeExec]) > 0 && p.execFilter(context, urlPath, BeforeExec) {
		goto Admin
	}

	//check policies
	if p.execPolicy(context, urlPath) {
		goto Admin
	}

	if routerInfo != nil {
		//store router pattern into context
		context.Input.SetData("RouterPattern", routerInfo.pattern)
		if routerInfo.routerType == routerTypeRESTFul {
			if _, ok := routerInfo.methods[r.Method]; ok {
				isRunnable = true
				routerInfo.runFunction(context)
			} else {
				exception("405", context)
				goto Admin
			}
		} else if routerInfo.routerType == routerTypeHandler {
			isRunnable = true
			routerInfo.handler.ServeHTTP(rw, r)
		} else {
			runRouter = routerInfo.controllerType
			methodParams = routerInfo.methodParams
			method := r.Method
			if r.Method == http.MethodPost && context.Input.Query("_method") == http.MethodPost {
				method = http.MethodPut
			}
			if r.Method == http.MethodPost && context.Input.Query("_method") == http.MethodDelete {
				method = http.MethodDelete
			}
			if m, ok := routerInfo.methods[method]; ok {
				runMethod = m
			} else if m, ok = routerInfo.methods["*"]; ok {
				runMethod = m
			} else {
				runMethod = method
			}
		}
	}

	// also defined runRouter & runMethod from filter
	if !isRunnable {
		//Invoke the request handler
		var execController ControllerInterface
		if routerInfo != nil && routerInfo.initialize != nil {
			execController = routerInfo.initialize()
		} else {
			vc := reflect.New(runRouter)
			var ok bool
			execController, ok = vc.Interface().(ControllerInterface)
			if !ok {
				panic("controller is not ControllerInterface")
			}
		}

		//call the controller init function
		execController.Init(context, runRouter.Name(), runMethod, execController)

		//call prepare function
		execController.Prepare()

		//if XSRF is Enable then check cookie where there has any cookie in the  request's cookie _csrf
		if BConfig.WebConfig.EnableXSRF {
			execController.XSRFToken()
			if r.Method == http.MethodPost || r.Method == http.MethodDelete || r.Method == http.MethodPut ||
				(r.Method == http.MethodPost && (context.Input.Query("_method") == http.MethodDelete || context.Input.Query("_method") == http.MethodPut)) {
				execController.CheckXSRFCookie()
			}
		}

		execController.URLMapping()

		if !context.ResponseWriter.Started {
			//exec main logic
			switch runMethod {
			case http.MethodGet:
				execController.Get()
			case http.MethodPost:
				execController.Post()
			case http.MethodDelete:
				execController.Delete()
			case http.MethodPut:
				execController.Put()
			case http.MethodHead:
				execController.Head()
			case http.MethodPatch:
				execController.Patch()
			case http.MethodOptions:
				execController.Options()
			default:
				if !execController.HandlerFunc(runMethod) {
					vc := reflect.ValueOf(execController)
					method := vc.MethodByName(runMethod)
					in := param.ConvertParams(methodParams, method.Type(), context)
					out := method.Call(in)

					//For backward compatibility we only handle response if we had incoming methodParams
					if methodParams != nil {
						p.handleParamResponse(context, execController, out)
					}
				}
			}

			//render template
			if !context.ResponseWriter.Started && context.Output.Status == 0 {
				if BConfig.WebConfig.AutoRender {
					if err := execController.Render(); err != nil {
						logs.Error(err)
					}
				}
			}
		}

		// finish all runRouter. release resource
		execController.Finish()
	}

	//execute middleware filters
	if len(p.filters[AfterExec]) > 0 && p.execFilter(context, urlPath, AfterExec) {
		goto Admin
	}

	if len(p.filters[FinishRouter]) > 0 && p.execFilter(context, urlPath, FinishRouter) {
		goto Admin
	}

Admin:
	//admin module record QPS

	statusCode := context.ResponseWriter.Status
	if statusCode == 0 {
		statusCode = 200
	}

	logAccess(context, &startTime, statusCode)

	if BConfig.Listen.EnableAdmin {
		timeDur := time.Since(startTime)
		pattern := ""
		if routerInfo != nil {
			pattern = routerInfo.pattern
		}

		if FilterMonitorFunc(r.Method, r.URL.Path, timeDur, pattern, statusCode) {
			if runRouter != nil {
				go toolbox.StatisticsMap.AddStatistics(r.Method, r.URL.Path, runRouter.Name(), timeDur)
			} else {
				go toolbox.StatisticsMap.AddStatistics(r.Method, r.URL.Path, "", timeDur)
			}
		}
	}

	if BConfig.RunMode == DEV && !BConfig.Log.AccessLogs {
		var devInfo string
		timeDur := time.Since(startTime)
		iswin := (runtime.GOOS == "windows")
		statusColor := logs.ColorByStatus(iswin, statusCode)
		methodColor := logs.ColorByMethod(iswin, r.Method)
		resetColor := logs.ColorByMethod(iswin, "")
		if findRouter {
			if routerInfo != nil {
				devInfo = fmt.Sprintf("|%15s|%s %3d %s|%13s|%8s|%s %-7s %s %-3s   r:%s", context.Input.IP(), statusColor, statusCode,
					resetColor, timeDur.String(), "match", methodColor, r.Method, resetColor, r.URL.Path,
					routerInfo.pattern)
			} else {
				devInfo = fmt.Sprintf("|%15s|%s %3d %s|%13s|%8s|%s %-7s %s %-3s", context.Input.IP(), statusColor, statusCode, resetColor,
					timeDur.String(), "match", methodColor, r.Method, resetColor, r.URL.Path)
			}
		} else {
			devInfo = fmt.Sprintf("|%15s|%s %3d %s|%13s|%8s|%s %-7s %s %-3s", context.Input.IP(), statusColor, statusCode, resetColor,
				timeDur.String(), "nomatch", methodColor, r.Method, resetColor, r.URL.Path)
		}
		if iswin {
			logs.W32Debug(devInfo)
		} else {
			logs.Debug(devInfo)
		}
	}
	// Call WriteHeader if status code has been set changed
	if context.Output.Status != 0 {
		context.ResponseWriter.WriteHeader(context.Output.Status)
	}
}
```

## 注册方式
### 手工注册
手工注册需要在用户创建的router.go文件中添加Init方法，在Init方法中使用Beego.Router函数添加，Router是App结构的一个字段，App初始化也会在Init函数中进行，因此使用的时候已经是实例化过的，底层调用的是addWithMethodParams方法添加步骤如下

1. 获取ControllerInterface的反射结构
2. 针对传递的mappingMethods进行处理获取可接受的请求Method和对应的方法名，然后通过反射验证方法名是否存在
3. 实例化ControllerInfo设置该处理器对应的path，method，routerType，controllerType等信息，最后设置initialize初始化函数，该函数首先根据controllerType生成一个反射结构并尝试转换为ControllerInterface，然后将注册的Controller的Field均赋值给刚才产生的execController
4. 如果没有指定请求method或者请求Method为*将会循环HTTPMETHOD添加其方法(addToRouter函数)，否则按照指定的请求Method添加。addToRouter函数内容：首先尝试从ControllerRegister.routers获取method，如果存在调用AddRouter来添加字节点，如果不存在则调用NewTree实例化并添加最后注册到routers中


```go
// Add controller handler and pattern rules to ControllerRegister.
// usage:
//	default methods is the same name as method
//	Add("/user",&UserController{})
//	Add("/api/list",&RestController{},"*:ListFood")
//	Add("/api/create",&RestController{},"post:CreateFood")
//	Add("/api/update",&RestController{},"put:UpdateFood")
//	Add("/api/delete",&RestController{},"delete:DeleteFood")
//	Add("/api",&RestController{},"get,post:ApiFunc"
//	Add("/simple",&SimpleController{},"get:GetFunc;post:PostFunc")
func (p *ControllerRegister) Add(pattern string, c ControllerInterface, mappingMethods ...string) {
	p.addWithMethodParams(pattern, c, nil, mappingMethods...)
}
// 
func (p *ControllerRegister) addWithMethodParams(pattern string, c ControllerInterface, methodParams []*param.MethodParam, mappingMethods ...string) {
    // 反射获取控制器值
    reflectVal := reflect.ValueOf(c)
    // 判断是否是指针，如果是指针返回指针指向的值，否则返回该值
	t := reflect.Indirect(reflectVal).Type()
    methods := make(map[string]string)
    // 获取映射的请求方法
	if len(mappingMethods) > 0 {
        // Add("/simple",&SimpleController{},"get:GetFunc;post:PostFunc")
        // 将会切分成get:GetFunc和post:PostFunc
		semi := strings.Split(mappingMethods[0], ";")
		for _, v := range semi {
            // 获取请求方法以及对应的处理器
			colon := strings.Split(v, ":")
			if len(colon) != 2 {
				panic("method mapping format is invalid")
            }
            // 处理"get,post:ApiFunc"获取请求方法
			comma := strings.Split(colon[0], ",")
			for _, m := range comma {
                // 转为大写，判断是否要求注册的请求方法被支持 （HTTPMETHOD映射）
				if m == "*" || HTTPMETHOD[strings.ToUpper(m)] {
                    // 反射获取控制器的reflect
					if val := reflectVal.MethodByName(colon[1]); val.IsValid() {
                        // 将请求方法与method的reflect进行映射
						methods[strings.ToUpper(m)] = colon[1]
					} else {
						panic("'" + colon[1] + "' method doesn't exist in the controller " + t.Name())
					}
				} else {
					panic(v + " is an invalid method mapping. Method doesn't exist " + m)
				}
			}
		}
	}
    // 创建控制器信息
	route := &ControllerInfo{}
	route.pattern = pattern //注册path信息
	route.methods = methods //注册结果经过处理后的方法与method映射
	route.routerType = routerTypeBeego // 类型为routerTypeBeego
	route.controllerType = t // 注册controoler值
	route.initialize = func() ControllerInterface { // 
        vc := reflect.New(route.controllerType)
        // 判断是否实现了ControllerInterface
		execController, ok := vc.Interface().(ControllerInterface)
		if !ok {
			panic("controller is not ControllerInterface")
		}
        // 获得type和val信息
		elemVal := reflect.ValueOf(c).Elem()
		elemType := reflect.TypeOf(c).Elem()
		execElem := reflect.ValueOf(execController).Elem()
        // 获取controller的字段
		numOfFields := elemVal.NumField()
		for i := 0; i < numOfFields; i++ {
			fieldType := elemType.Field(i)
			elemField := execElem.FieldByName(fieldType.Name)
			if elemField.CanSet() {
				fieldVal := elemVal.Field(i)
				elemField.Set(fieldVal)
			}
		}

		return execController
	}

    route.methodParams = methodParams
    // 如果没有注册到处理方法
	if len(methods) == 0 {
        // 默认使用所有的方法
		for m := range HTTPMETHOD {
			p.addToRouter(m, pattern, route)
		}
	} else {
		for k := range methods {
            // 如果映射中还有*默认注册所有方法
			if k == "*" {
				for m := range HTTPMETHOD {
					p.addToRouter(m, pattern, route)
				}
			} else {
				p.addToRouter(k, pattern, route)
			}
		}
	}
}

func (p *ControllerRegister) addToRouter(method, pattern string, r *ControllerInfo) {
	if !BConfig.RouterCaseSensitive {
		pattern = strings.ToLower(pattern)
    }
    // 注册请求方式，如果含有重复的key则建立一个tree用于存放相同key
	if t, ok := p.routers[method]; ok {
		t.AddRouter(pattern, r)
	} else {
		t := NewTree()
		t.AddRouter(pattern, r)
		p.routers[method] = t
	}
}


// AddRouter call addseg function
// 向tree中添加 方法
func (t *Tree) AddRouter(pattern string, runObject interface{}) {
	t.addseg(splitPath(pattern), runObject, nil, "")
}

// "/" -> []
// "/admin" -> ["admin"]
// "/admin/" -> ["admin"]
// "/admin/users" -> ["admin", "users"]
func splitPath(key string) []string {
	key = strings.Trim(key, "/ ")
	if key == "" {
		return []string{}
	}
	return strings.Split(key, "/")
}

// "/"
// "admin" ->
func (t *Tree) addseg(segments []string, route interface{}, wildcards []string, reg string) {
    // 如果是根路径 /
	if len(segments) == 0 {
        // 
		if reg != "" {
			t.leaves = append(t.leaves, &leafInfo{runObject: route, wildcards: wildcards, regexps: regexp.MustCompile("^" + reg + "$")})
		} else {
			t.leaves = append(t.leaves, &leafInfo{runObject: route, wildcards: wildcards})
		}
	} else {
		seg := segments[0]
		iswild, params, regexpStr := splitSegment(seg)
		// if it's ? meaning can igone this, so add one more rule for it
		if len(params) > 0 && params[0] == ":" {
			t.addseg(segments[1:], route, wildcards, reg)
			params = params[1:]
		}
		//Rule: /login/*/access match /login/2009/11/access
		//if already has *, and when loop the access, should as a regexpStr
		if !iswild && utils.InSlice(":splat", wildcards) {
			iswild = true
			regexpStr = seg
		}
		//Rule: /user/:id/*
		if seg == "*" && len(wildcards) > 0 && reg == "" {
			regexpStr = "(.+)"
		}
		if iswild {
			if t.wildcard == nil {
				t.wildcard = NewTree()
			}
			if regexpStr != "" {
				if reg == "" {
					rr := ""
					for _, w := range wildcards {
						if w == ":splat" {
							rr = rr + "(.+)/"
						} else {
							rr = rr + "([^/]+)/"
						}
					}
					regexpStr = rr + regexpStr
				} else {
					regexpStr = "/" + regexpStr
				}
			} else if reg != "" {
				if seg == "*.*" {
					regexpStr = "/([^.]+).(.+)"
					params = params[1:]
				} else {
					for range params {
						regexpStr = "/([^/]+)" + regexpStr
					}
				}
			} else {
				if seg == "*.*" {
					params = params[1:]
				}
			}
			t.wildcard.addseg(segments[1:], route, append(wildcards, params...), reg+regexpStr)
		} else {
			var subTree *Tree
			for _, sub := range t.fixrouters {
				if sub.prefix == seg {
					subTree = sub
					break
				}
			}
			if subTree == nil {
				subTree = NewTree()
				subTree.prefix = seg
				t.fixrouters = append(t.fixrouters, subTree)
			}
			subTree.addseg(segments[1:], route, wildcards, reg)
		}
	}
}

```

### 添加处理函数
手工注册的方式使用的是结构体作为参数添加的，beego也提供了使用函数的方式添加处理器

```go
// FilterFunc将会在controller的handler执行之前调用
type FilterFunc func(*context.Context)
```

提供了常用请求的快捷api
```go

// Get add get method
// usage:
//    Get("/", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func (p *ControllerRegister) Get(pattern string, f FilterFunc) {
	p.AddMethod("get", pattern, f)
}

// Post add post method
// usage:
//    Post("/api", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func (p *ControllerRegister) Post(pattern string, f FilterFunc) {
	p.AddMethod("post", pattern, f)
}

// Put add put method
// usage:
//    Put("/api/:id", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func (p *ControllerRegister) Put(pattern string, f FilterFunc) {
	p.AddMethod("put", pattern, f)
}

// Delete add delete method
// usage:
//    Delete("/api/:id", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func (p *ControllerRegister) Delete(pattern string, f FilterFunc) {
	p.AddMethod("delete", pattern, f)
}

// Head add head method
// usage:
//    Head("/api/:id", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func (p *ControllerRegister) Head(pattern string, f FilterFunc) {
	p.AddMethod("head", pattern, f)
}

// Patch add patch method
// usage:
//    Patch("/api/:id", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func (p *ControllerRegister) Patch(pattern string, f FilterFunc) {
	p.AddMethod("patch", pattern, f)
}

// Options add options method
// usage:
//    Options("/api/:id", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func (p *ControllerRegister) Options(pattern string, f FilterFunc) {
	p.AddMethod("options", pattern, f)
}

// Any add all method
// usage:
//    Any("/api/:id", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func (p *ControllerRegister) Any(pattern string, f FilterFunc) {
	p.AddMethod("*", pattern, f)
}
```
注册的流程与注册Controller的流程基本一致，如果使用函数的方式routerType将会是routerTypeRESTFul
```
// AddMethod add http method router
// usage:
//    AddMethod("get","/api/:id", func(ctx *context.Context){
//          ctx.Output.Body("hello world")
//    })
func (p *ControllerRegister) AddMethod(method, pattern string, f FilterFunc) {
	method = strings.ToUpper(method)
	if method != "*" && !HTTPMETHOD[method] {
		panic("not support http method: " + method)
	}
	route := &ControllerInfo{}
	route.pattern = pattern
	route.routerType = routerTypeRESTFul
	route.runFunction = f
	methods := make(map[string]string)
	if method == "*" {
		for val := range HTTPMETHOD {
			methods[val] = val
		}
	} else {
		methods[method] = method
	}
	route.methods = methods
	for k := range methods {
		if k == "*" {
			for m := range HTTPMETHOD {
				p.addToRouter(m, pattern, route)
			}
		} else {
			p.addToRouter(k, pattern, route)
		}
	}
}
```

### 使用Handler注册
beego页提供了Handler的方式注册，只要实现ServeHTTP(ResponseWriter, *Request)方法的都可以添加
```go
// Handler add user defined Handler
func (p *ControllerRegister) Handler(pattern string, h http.Handler, options ...interface{}) {
	route := &ControllerInfo{}
	route.pattern = pattern
	route.routerType = routerTypeHandler
	route.handler = h
	if len(options) > 0 {
		if _, ok := options[0].(bool); ok {
			pattern = path.Join(pattern, "?:all(.*)")
		}
	}
	for m := range HTTPMETHOD {
		p.addToRouter(m, pattern, route)
	}
}

```

### Filter

Filter会根据指定的pos（调用位置）在不同的阶段进行使用，也可作为单独的处理函数，在注解路由中将会使用到，可以使用InsertFilter来添加

```go
const (
	BeforeStatic = iota //在staticfile处理之前
	BeforeRouter // 在router处理之前
	BeforeExec // 在controller执行之前
	AfterExec // 在controller执行之后
	FinishRouter // 在router处理之后
)

// 结构如下
type FilterRouter struct {
	filterFunc     FilterFunc // 处理函数
	tree           *Tree //用于存储子路径 /admin/other或者/admin/other/aother
	pattern        string // 父路径 /admin/
	returnOnOutput bool // 
	resetParams    bool // 
}
```


### 注解注册
注解路由利用的init特性，首先根据注册的Controller查找到该文件，提取注释和对应的处理器函数名称，然后添加到GlobalControllerRouter中，然后遍历GlobalControllerRouter的key使用InsertFilter来注册处理函数，如果是开发者模式，则会正在router文件夹中创建auto.go,关于注解提取的实现在beego/parser.go文件中关于parser.go解析可参考[注解路由](note/beego/parser.md)

```go
// 如果启用开发模式，将会根据Controller创建router/auto.go文件
// 使用：
// type BankAccount struct{
//   beego.Controller
// }
//
// register the function
// func (b *BankAccount)Mapping(){
//  b.Mapping("ShowAccount" , b.ShowAccount)
//  b.Mapping("ModifyAccount", b.ModifyAccount)
//}
//
// //@router /account/:id  [get]
// func (b *BankAccount) ShowAccount(){
//    //logic
// }
//
//
// //@router /account/:id  [post]
// func (b *BankAccount) ModifyAccount(){
//    //logic
// } 
// 	Include(&BankAccount{}, &OrderController{},&RefundController{},&ReceiptController{})
func (p *ControllerRegister) Include(cList ...ControllerInterface) {
	if BConfig.RunMode == DEV {
		skip := make(map[string]bool, 10)
		for _, c := range cList { // 循环处理Controller
			reflectVal := reflect.ValueOf(c)
			// 返回对应的值或者指针指向的值
			t := reflect.Indirect(reflectVal).Type()
			// 获取go path
			wgopath := utils.GetGOPATHs()
			if len(wgopath) == 0 {
				panic("you are in dev mode. So please set gopath")
			}
			pkgpath := ""
			// 获取所有路径名
			for _, wg := range wgopath {
				// 在评估任何符号链接后，EvalSymlinks返回路径名。如果path是相对的，则结果将相对于当前目录，除非其中一个组件是绝对符号链接.EvalSymlinks在结果上调用Clean
				wg, _ = filepath.EvalSymlinks(filepath.Join(wg, "src", t.PkgPath()))
				if utils.FileExists(wg) {
					pkgpath = wg
					break
				}
			}
			if pkgpath != "" {
				if _, ok := skip[pkgpath]; !ok {
					skip[pkgpath] = true
					parserPkg(pkgpath, t.PkgPath())
				}
			}
		}
	}
	for _, c := range cList {
		reflectVal := reflect.ValueOf(c)
		t := reflect.Indirect(reflectVal).Type()
		key := t.PkgPath() + ":" + t.Name()
		// GlobalControllerRouter用于存储controller的comments
		// comments= pkgpath+controller
		// Include前需要注册相应的Controller以及对应的处理函数
		// 如果没有注册将不会使用
		if comm, ok := GlobalControllerRouter[key]; ok {
			for _, a := range comm {
				// 遍历所有的Filters并向ControllerRegister注册
				for _, f := range a.Filters {
					p.InsertFilter(f.Pattern, f.Pos, f.Filter, f.ReturnOnOutput, f.ResetParams)
				}
				// 从ControllerComments添加path规则，Controller和请求方法等信息
				p.addWithMethodParams(a.Router, c, a.MethodParams, strings.Join(a.AllowHTTPMethods, ",")+":"+a.Method)
			}
		}
	}
}
```

# 控制器
## 控制器结构
```go
type ControllerInterface interface {
	Init(ct *context.Context, controllerName, actionName string, app interface{})
	Prepare()
	Get()
	Post()
	Delete()
	Put()
	Head()
	Patch()
	Options()
	Finish()
	Render() error
	XSRFToken() string
	CheckXSRFCookie() bool
	HandlerFunc(fn string) bool
	URLMapping()
}

// Controller定义了一些基础的请求处理器，包括http context, template， view, session，xsrf
type Controller struct {
    // context data
    // 上下文，每个处理器可以使用该上下文获取当前请求的信息
	Ctx  *context.Context
	Data map[interface{}]interface{} // 数据映射（用于模板）

    // route controller info
	controllerName string // router相关
	actionName     string
	methodMapping  map[string]func() //method:routertree
	gotofunc       string
	AppController  interface{}

	// template data
	TplName        string // 使用的模板名
	ViewPath       string // 
	Layout         string //
	LayoutSections map[string]string // the key is the section name and the value is the template name
	TplPrefix      string
	TplExt         string
	EnableRender   bool

	// xsrf data
	_xsrfToken string
	XSRFExpire int
	EnableXSRF bool

    // session
	CruSession session.Store
}


// ControllerFilter store the filter for controller
type ControllerFilter struct {
	Pattern        string
	Pos            int
	Filter         FilterFunc
	ReturnOnOutput bool
	ResetParams    bool
}

// ControllerFilterComments store the comment for controller level filter
type ControllerFilterComments struct {
	Pattern        string
	Pos            int
	Filter         string // NOQA
	ReturnOnOutput bool
	ResetParams    bool
}
// ControllerImportComments store the import comment for controller needed
type ControllerImportComments struct {
	ImportPath  string
	ImportAlias string
}

// ControllerComments store the comment for the controller method
type ControllerComments struct {
	Method           string
	Router           string
	Filters          []*ControllerFilter
	ImportComments   []*ControllerImportComments
	FilterComments   []*ControllerFilterComments
	AllowHTTPMethods []string
	Params           []map[string]string
	MethodParams     []*param.MethodParam
}

type ControllerCommentsSlice []ControllerComments

```

### 对应方法
```go
// 用于初始化Controller，参数传递是在ServeHTTP中进行
func (c *Controller) Init(ctx *context.Context, controllerName, actionName string, app interface{}) {
	c.Layout = ""
	c.TplName = ""
	c.controllerName = controllerName
	c.actionName = actionName
	c.Ctx = ctx
	c.TplExt = "tpl"
	c.AppController = app
	c.EnableRender = true
	c.EnableXSRF = true
	c.Data = ctx.Input.Data()
	c.methodMapping = make(map[string]func())
}

// 在Init函数执行完毕后将会调用Prepare函数，一般用于初始化后的准备工作
func (c *Controller) Prepare() {}

// 在处理完一个请求之后将调用Finish释放资源
func (c *Controller) Finish() {}

// Get adds a request function to handle GET request.
func (c *Controller) Get() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Post adds a request function to handle POST request.
func (c *Controller) Post() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Delete adds a request function to handle DELETE request.
func (c *Controller) Delete() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Put adds a request function to handle PUT request.
func (c *Controller) Put() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Head adds a request function to handle HEAD request.
func (c *Controller) Head() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Patch adds a request function to handle PATCH request.
func (c *Controller) Patch() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// Options adds a request function to handle OPTIONS request.
func (c *Controller) Options() {
	http.Error(c.Ctx.ResponseWriter, "Method Not Allowed", 405)
}

// HandlerFunc call function with the name
// HandlerFunc将会在methodMapping查找方法名，如果调用后返回true
func (c *Controller) HandlerFunc(fnname string) bool {
	if v, ok := c.methodMapping[fnname]; ok {
		v()
		return true
	}
	return false
}

// URLMapping register the internal Controller router.
// 可以在URLMapping方法中嗲用Mapping来添加Controller已经对应的path
func (c *Controller) URLMapping() {}

// 添加处理器与path的映射
func (c *Controller) Mapping(method string, fn func()) {
	c.methodMapping[method] = fn
}

// Render将会渲染模板并按照text/html类型发送数据
func (c *Controller) Render() error {
	if !c.EnableRender {
		return nil
	}
	rb, err := c.RenderBytes()
	if err != nil {
		return err
	}

	if c.Ctx.ResponseWriter.Header().Get("Content-Type") == "" {
		c.Ctx.Output.Header("Content-Type", "text/html; charset=utf-8")
	}

	return c.Ctx.Output.Body(rb)
}

// RenderString返回渲染后的内容，不会发送数据
func (c *Controller) RenderString() (string, error) {
	b, e := c.RenderBytes()
	return string(b), e
}

// RenderBytes返回渲染内容后的byte数组，不会发送数据
func (c *Controller) RenderBytes() ([]byte, error) {
	buf, err := c.renderTemplate()
	//if the controller has set layout, then first get the tplName's content set the content to the layout
	if err == nil && c.Layout != "" {
		c.Data["LayoutContent"] = template.HTML(buf.String())

		if c.LayoutSections != nil {
			for sectionName, sectionTpl := range c.LayoutSections {
				if sectionTpl == "" {
					c.Data[sectionName] = ""
					continue
				}
				buf.Reset()
				err = ExecuteViewPathTemplate(&buf, sectionTpl, c.viewPath(), c.Data)
				if err != nil {
					return nil, err
				}
				c.Data[sectionName] = template.HTML(buf.String())
			}
		}

		buf.Reset()
		ExecuteViewPathTemplate(&buf, c.Layout, c.viewPath(), c.Data)
	}
	return buf.Bytes(), err
}

func (c *Controller) renderTemplate() (bytes.Buffer, error) {
	var buf bytes.Buffer
	if c.TplName == "" {
		c.TplName = strings.ToLower(c.controllerName) + "/" + strings.ToLower(c.actionName) + "." + c.TplExt
	}
	if c.TplPrefix != "" {
		c.TplName = c.TplPrefix + c.TplName
	}
	if BConfig.RunMode == DEV {
		buildFiles := []string{c.TplName}
		if c.Layout != "" {
			buildFiles = append(buildFiles, c.Layout)
			if c.LayoutSections != nil {
				for _, sectionTpl := range c.LayoutSections {
					if sectionTpl == "" {
						continue
					}
					buildFiles = append(buildFiles, sectionTpl)
				}
			}
		}
		BuildTemplate(c.viewPath(), buildFiles...)
	}
	return buf, ExecuteViewPathTemplate(&buf, c.TplName, c.viewPath(), c.Data)
}

func (c *Controller) viewPath() string {
	if c.ViewPath == "" {
		return BConfig.WebConfig.ViewsPath
	}
	return c.ViewPath
}

// Redirect重定向，指定路径和响应码
func (c *Controller) Redirect(url string, code int) {
	logAccess(c.Ctx, nil, code)
	c.Ctx.Redirect(code, url)
}

// SetData 根据Handler中Accept所能接受的类型添加数据
func (c *Controller) SetData(data interface{}) {
	accept := c.Ctx.Input.Header("Accept")
	switch accept {
	case context.ApplicationYAML:
		c.Data["yaml"] = data
	case context.ApplicationXML, context.TextXML:
		c.Data["xml"] = data
	default:
		c.Data["json"] = data
	}
}

// Abort stops controller handler and show the error data if code is defined in ErrorMap or code string.
// Abort将会停止controller如果定义了ErrorMap或者错误提示将会显示错误
func (c *Controller) Abort(code string) {
	status, err := strconv.Atoi(code)
	if err != nil {
		status = 200
	}
	c.CustomAbort(status, code)
}

// CustomAbort跟Abort蕾丝，但是同时支持状态码和body
func (c *Controller) CustomAbort(status int, body string) {
	// first panic from ErrorMaps, it is user defined error functions.
	if _, ok := ErrorMaps[body]; ok {
		c.Ctx.Output.Status = status
		panic(body)
	}
	// last panic user string
	c.Ctx.ResponseWriter.WriteHeader(status)
	c.Ctx.ResponseWriter.Write([]byte(body))
	panic(ErrAbort)
}

// StopRun将会panic USERSTOPRUN的错误，如果定了recover，将会进入recover函数
func (c *Controller) StopRun() {
	panic(ErrAbort)
}

// URLFor does another controller handler in this request function.
// it goes to this controller method if endpoint is not clear.

func (c *Controller) URLFor(endpoint string, values ...interface{}) string {
	if len(endpoint) == 0 {
		return ""
	}
	if endpoint[0] == '.' {
		return URLFor(reflect.Indirect(reflect.ValueOf(c.AppController)).Type().Name()+endpoint, values...)
	}
	return URLFor(endpoint, values...)
}

// 返回json响应
func (c *Controller) ServeJSON(encoding ...bool) {
	var (
		hasIndent   = BConfig.RunMode != PROD
		hasEncoding = len(encoding) > 0 && encoding[0]
	)

	c.Ctx.Output.JSON(c.Data["json"], hasIndent, hasEncoding)
}

// 发送jsonp响应
func (c *Controller) ServeJSONP() {
	hasIndent := BConfig.RunMode != PROD
	c.Ctx.Output.JSONP(c.Data["jsonp"], hasIndent)
}

// 发送xml响应
func (c *Controller) ServeXML() {
	hasIndent := BConfig.RunMode != PROD
	c.Ctx.Output.XML(c.Data["xml"], hasIndent)
}

// 发送yaml响应
func (c *Controller) ServeYAML() {
	c.Ctx.Output.YAML(c.Data["yaml"])
}

// ServeFormatted会根据header中的Accept处理YAML, XML 或 JSON
func (c *Controller) ServeFormatted(encoding ...bool) {
	hasIndent := BConfig.RunMode != PROD
	hasEncoding := len(encoding) > 0 && encoding[0]
	c.Ctx.Output.ServeFormatted(c.Data, hasIndent, hasEncoding)
}

// 返回请求参数的映射关系(POST或者PUT方法)
func (c *Controller) Input() url.Values {
	if c.Ctx.Request.Form == nil {
		c.Ctx.Request.ParseForm()
	}
	return c.Ctx.Request.Form
}

// 请求参数映射专为obj结构体
func (c *Controller) ParseForm(obj interface{}) error {
	return ParseForm(c.Input(), obj)
}

// GetString returns the input value by key string or the default value while it's present and input is blank
// 
func (c *Controller) GetString(key string, def ...string) string {
	if v := c.Ctx.Input.Query(key); v != "" {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

// GetStrings returns the input string slice by key string or the default value while it's present and input is blank
// it's designed for multi-value input field such as checkbox(input[type=checkbox]), multi-selection.
func (c *Controller) GetStrings(key string, def ...[]string) []string {
	var defv []string
	if len(def) > 0 {
		defv = def[0]
	}

	if f := c.Input(); f == nil {
		return defv
	} else if vs := f[key]; len(vs) > 0 {
		return vs
	}

	return defv
}

// GetInt returns input as an int or the default value while it's present and input is blank
func (c *Controller) GetInt(key string, def ...int) (int, error) {
	strv := c.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.Atoi(strv)
}

// GetInt8 return input as an int8 or the default value while it's present and input is blank
func (c *Controller) GetInt8(key string, def ...int8) (int8, error) {
	strv := c.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	i64, err := strconv.ParseInt(strv, 10, 8)
	return int8(i64), err
}

// GetUint8 return input as an uint8 or the default value while it's present and input is blank
func (c *Controller) GetUint8(key string, def ...uint8) (uint8, error) {
	strv := c.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	u64, err := strconv.ParseUint(strv, 10, 8)
	return uint8(u64), err
}

// GetInt16 returns input as an int16 or the default value while it's present and input is blank
func (c *Controller) GetInt16(key string, def ...int16) (int16, error) {
	strv := c.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	i64, err := strconv.ParseInt(strv, 10, 16)
	return int16(i64), err
}

// GetUint16 returns input as an uint16 or the default value while it's present and input is blank
func (c *Controller) GetUint16(key string, def ...uint16) (uint16, error) {
	strv := c.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	u64, err := strconv.ParseUint(strv, 10, 16)
	return uint16(u64), err
}

// GetInt32 returns input as an int32 or the default value while it's present and input is blank
func (c *Controller) GetInt32(key string, def ...int32) (int32, error) {
	strv := c.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	i64, err := strconv.ParseInt(strv, 10, 32)
	return int32(i64), err
}

// GetUint32 returns input as an uint32 or the default value while it's present and input is blank
func (c *Controller) GetUint32(key string, def ...uint32) (uint32, error) {
	strv := c.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	u64, err := strconv.ParseUint(strv, 10, 32)
	return uint32(u64), err
}

// GetInt64 returns input value as int64 or the default value while it's present and input is blank.
func (c *Controller) GetInt64(key string, def ...int64) (int64, error) {
	strv := c.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseInt(strv, 10, 64)
}

// GetUint64 returns input value as uint64 or the default value while it's present and input is blank.
func (c *Controller) GetUint64(key string, def ...uint64) (uint64, error) {
	strv := c.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseUint(strv, 10, 64)
}

// GetBool returns input value as bool or the default value while it's present and input is blank.
func (c *Controller) GetBool(key string, def ...bool) (bool, error) {
	strv := c.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseBool(strv)
}

// GetFloat returns input value as float64 or the default value while it's present and input is blank.
func (c *Controller) GetFloat(key string, def ...float64) (float64, error) {
	strv := c.Ctx.Input.Query(key)
	if len(strv) == 0 && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseFloat(strv, 64)
}

// GetFile returns the file data in file upload field named as key.
// it returns the first one of multi-uploaded files.
func (c *Controller) GetFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return c.Ctx.Request.FormFile(key)
}


// GetFiles return multi-upload files
// files, err:=c.GetFiles("myfiles")
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusNoContent)
//		return
//	}
// for i, _ := range files {
//	//for each fileheader, get a handle to the actual file
//	file, err := files[i].Open()
//	defer file.Close()
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//	//create destination file making sure the path is writeable.
//	dst, err := os.Create("upload/" + files[i].Filename)
//	defer dst.Close()
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
//	//copy the uploaded file to the destination file
//	if _, err := io.Copy(dst, file); err != nil {
//		http.Error(w, err.Error(), http.StatusInternalServerError)
//		return
//	}
// }
func (c *Controller) GetFiles(key string) ([]*multipart.FileHeader, error) {
	if files, ok := c.Ctx.Request.MultipartForm.File[key]; ok {
		return files, nil
	}
	return nil, http.ErrMissingFile
}

// SaveToFile saves uploaded file to new path.
// it only operates the first one of mutil-upload form file field.
func (c *Controller) SaveToFile(fromfile, tofile string) error {
	file, _, err := c.Ctx.Request.FormFile(fromfile)
	if err != nil {
		return err
	}
	defer file.Close()
	f, err := os.OpenFile(tofile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	io.Copy(f, file)
	return nil
}

// StartSession starts session and load old session data info this controller.
func (c *Controller) StartSession() session.Store {
	if c.CruSession == nil {
		c.CruSession = c.Ctx.Input.CruSession
	}
	return c.CruSession
}

// SetSession puts value into session.
func (c *Controller) SetSession(name interface{}, value interface{}) {
	if c.CruSession == nil {
		c.StartSession()
	}
	c.CruSession.Set(name, value)
}

// GetSession gets value from session.
func (c *Controller) GetSession(name interface{}) interface{} {
	if c.CruSession == nil {
		c.StartSession()
	}
	return c.CruSession.Get(name)
}

// DelSession removes value from session.
func (c *Controller) DelSession(name interface{}) {
	if c.CruSession == nil {
		c.StartSession()
	}
	c.CruSession.Delete(name)
}

// SessionRegenerateID regenerates session id for this session.
// the session data have no changes.
func (c *Controller) SessionRegenerateID() {
	if c.CruSession != nil {
		c.CruSession.SessionRelease(c.Ctx.ResponseWriter)
	}
	c.CruSession = GlobalSessions.SessionRegenerateID(c.Ctx.ResponseWriter, c.Ctx.Request)
	c.Ctx.Input.CruSession = c.CruSession
}

// DestroySession cleans session data and session cookie.
func (c *Controller) DestroySession() {
	c.Ctx.Input.CruSession.Flush()
	c.Ctx.Input.CruSession = nil
	GlobalSessions.SessionDestroy(c.Ctx.ResponseWriter, c.Ctx.Request)
}

// IsAjax returns this request is ajax or not.
func (c *Controller) IsAjax() bool {
	return c.Ctx.Input.IsAjax()
}

// GetSecureCookie returns decoded cookie value from encoded browser cookie values.
func (c *Controller) GetSecureCookie(Secret, key string) (string, bool) {
	return c.Ctx.GetSecureCookie(Secret, key)
}

// SetSecureCookie puts value into cookie after encoded the value.
func (c *Controller) SetSecureCookie(Secret, name, value string, others ...interface{}) {
	c.Ctx.SetSecureCookie(Secret, name, value, others...)
}

// XSRFToken creates a CSRF token string and returns.
func (c *Controller) XSRFToken() string {
	if c._xsrfToken == "" {
		expire := int64(BConfig.WebConfig.XSRFExpire)
		if c.XSRFExpire > 0 {
			expire = int64(c.XSRFExpire)
		}
		c._xsrfToken = c.Ctx.XSRFToken(BConfig.WebConfig.XSRFKey, expire)
	}
	return c._xsrfToken
}

// CheckXSRFCookie checks xsrf token in this request is valid or not.
// the token can provided in request header "X-Xsrftoken" and "X-CsrfToken"
// or in form field value named as "_xsrf".
func (c *Controller) CheckXSRFCookie() bool {
	if !c.EnableXSRF {
		return true
	}
	return c.Ctx.CheckXSRFCookie()
}

// XSRFFormHTML writes an input field contains xsrf token value.
func (c *Controller) XSRFFormHTML() string {
	return `<input type="hidden" name="_xsrf" value="` +
		c.XSRFToken() + `" />`
}

// GetControllerAndAction gets the executing controller name and action name.
func (c *Controller) GetControllerAndAction() (string, string) {
	return c.controllerName, c.actionName
}

```