# beego多路复用器

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
    // these beego.Controller's methods shouldn't reflect to AutoRouter
    // beego.Controller's 所有的方法集合
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
## 控制器信息
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

```

## 注册控制器

```go
type ControllerRegister struct {
	routers      map[string]*Tree // beego实现了一个树结构用于存储path的路径
	enablePolicy bool //启用代理
	policies     map[string]*Tree
	enableFilter bool //启用过滤
	filters      [FinishRouter + 1][]*FilterRouter
	pool         sync.Pool // 用于存储beego.Context，对象复用
}

```
### 添加控制器

#### 手工注册
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

#### 注解注册
```go

// Include only when the Runmode is dev will generate router file in the router/auto.go from the controller
// Include(&BankAccount{}, &OrderController{},&RefundController{},&ReceiptController{})
func (p *ControllerRegister) Include(cList ...ControllerInterface) {
	if BConfig.RunMode == DEV {
		skip := make(map[string]bool, 10)
		for _, c := range cList {
			reflectVal := reflect.ValueOf(c)
			t := reflect.Indirect(reflectVal).Type()
			wgopath := utils.GetGOPATHs()
			if len(wgopath) == 0 {
				panic("you are in dev mode. So please set gopath")
			}
			pkgpath := ""
			for _, wg := range wgopath {
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
		if comm, ok := GlobalControllerRouter[key]; ok {
			for _, a := range comm {
				for _, f := range a.Filters {
					p.InsertFilter(f.Pattern, f.Pos, f.Filter, f.ReturnOnOutput, f.ResetParams)
				}

				p.addWithMethodParams(a.Router, c, a.MethodParams, strings.Join(a.AllowHTTPMethods, ",")+":"+a.Method)
			}
		}
	}
}

```

## tree结构

```go
// Tree has three elements: FixRouter/wildcard/leaves
// fixRouter stores Fixed Router
// wildcard stores params
// leaves store the endpoint information
type Tree struct {
	//prefix set for static router
	prefix string
	//search fix route first
	fixrouters []*Tree
	//if set, failure to match fixrouters search then search wildcard
	wildcard *Tree
	//if set, failure to match wildcard search
	leaves []*leafInfo
}
\
```