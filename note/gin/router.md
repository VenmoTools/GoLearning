# gin路由

## 接口

```go
// IRouter defines all router handle interface includes single and group router.
// IRouter定义了所有单路由，组路由接口
type IRouter interface {
	IRoutes
	Group(string, ...HandlerFunc) *RouterGroup
}

// IRoutes defines all router handle interface.
type IRoutes interface {
	Use(...HandlerFunc) IRoutes

	Handle(string, string, ...HandlerFunc) IRoutes
	Any(string, ...HandlerFunc) IRoutes
	GET(string, ...HandlerFunc) IRoutes
	POST(string, ...HandlerFunc) IRoutes
	DELETE(string, ...HandlerFunc) IRoutes
	PATCH(string, ...HandlerFunc) IRoutes
	PUT(string, ...HandlerFunc) IRoutes
	OPTIONS(string, ...HandlerFunc) IRoutes
	HEAD(string, ...HandlerFunc) IRoutes

	StaticFile(string, string) IRoutes
	Static(string, string) IRoutes
	StaticFS(string, http.FileSystem) IRoutes
}


```

## 路由结构

```go
// RouterGroup在用于配置路由器，RouterGroup与前缀和处理程序数组（中间件）相关联。
// engine组合了RouterGroup因此engine也实现了以上接口
type RouterGroup struct {
    // HandlersChain用于存放middleware切片
    Handlers HandlersChain
    // 根路径
    basePath string
	engine   *Engine
	root     bool
}
```
## 分组路由
Group方法将会创建一个新的router，添加PATH前缀相同的PATH或者是具有相同的中间件
```go
engine := gin.Deafult()
user := engine.Group("/admin")
{
	// 通过http://127.0.0.1/admin/login 访问
	user.GET("/login",LoginHander)
	// 通过http://127.0.0.1/admin/register 访问
	user.GET("/register",RegisterHandler)
}
// 含有file前缀的路径都将会添加用户认证中间件
file := engine.Group("/file",UserAuthorMiddleWare())
{
	file.GET("/getfile",FileDownlaodHandler)
	file.POST("/delete",DeleteFileHandler)
}

```
实现原理

调用Group()时将会创建一个新的RouterGroup，将传递的path作为basePath
例如传递的`user := engine.Group("/user")`,之后所有的router添加是都会以/user作为跟路径（原basePath为 ‘/’），然后通过engine的addRoute添加路由
```go
// 
func (group *RouterGroup) Group(relativePath string, handlers ...HandlerFunc) *RouterGroup {
	// 创建一个新的RouterGroup
	return &RouterGroup{
		Handlers: group.combineHandlers(handlers),
		// 将传递的path作为跟路径，然后复用IRouter添加路由
		basePath: group.calculateAbsolutePath(relativePath),
		engine:   group.engine,
	}
}
// 
func (group *RouterGroup) combineHandlers(handlers HandlersChain) HandlersChain {
	// 获取已经注册的路由以及将要注册的路由数量
	// 如果一次性注册路由数量大于 (1 << 7 - 1) / 2 = 63个 将会panic
	finalSize := len(group.Handlers) + len(handlers)
	if finalSize >= int(abortIndex) {
		panic("too many handlers")
	}
	// handlerFunc组进行扩展
	mergedHandlers := make(HandlersChain, finalSize)
	// 深拷贝，拷贝所有的已注册的HandlerFunc
	copy(mergedHandlers, group.Handlers)
	// 深拷贝，从已注册以后的HandlerFunc位置拷贝将要注册的handler
	copy(mergedHandlers[len(group.Handlers):], handlers)
	return mergedHandlers
}

// 根据当前路径计算绝对路径
func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return joinPaths(group.basePath, relativePath)
}

func joinPaths(absolutePath, relativePath string) string {
	if relativePath == "" {
		return absolutePath
	}
	// Join函数可以将任意数量的路径元素放入一个单一路径里，会根据需要添加斜杠。结果是经过简化的，所有的空字符串元素会被忽略
	finalPath := path.Join(absolutePath, relativePath)

	appendSlash := lastChar(relativePath) == '/' && lastChar(finalPath) != '/'
	if appendSlash {
		return finalPath + "/"
	}
	return finalPath
}


func (engine *Engine) addRoute(method, path string, handlers HandlersChain) {
	assert1(path[0] == '/', "path must begin with '/'")
	assert1(method != "", "HTTP method can not be empty")
	assert1(len(handlers) > 0, "there must be at least one handler")

	debugPrintRoute(method, path, handlers)
	root := engine.trees.get(method)
	if root == nil {
		root = new(node)
		engine.trees = append(engine.trees, methodTree{method: method, root: root})
	}
	root.addRoute(path, handlers)
}

```




### 路由注册步骤

1. 使用Use(middleware ...HandlerFunc) IRoutes添加middware，最终会保存在gin.go中的HandlersChain切片中
2. 使用Handle，Any，GET，POST，DELETE，PATCH，PUT，OPTIONS，HEAD方式注册对应的路由和处理函数，以上均调用handle方法，handle方法首先会根据传进来的相对路径进行合并成绝对路径（calculateAbsolutePath函数）然后通过combineHandlers来对传入的多个handler进行比较，最后进行整合
3. 调用engine的addRoute方法注册路由，首先会从engine中查找对应的method，如果没有找到则新建一个root并添加到原tree中，最后调用tree的addRoute方法添加（处理请求时也会使用tree来查找）