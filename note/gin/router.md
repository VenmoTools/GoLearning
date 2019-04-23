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

### 路由注册步骤

1. 使用Use(middleware ...HandlerFunc) IRoutes添加middware，最终会保存在gin.go中的HandlersChain切片中
2. 使用Handle，Any，GET，POST，DELETE，PATCH，PUT，OPTIONS，HEAD方式注册对应的路由和处理函数，以上均调用handle方法，handle方法首先会根据传进来的相对路径进行合并成绝对路径（calculateAbsolutePath函数）然后通过combineHandlers来对传入的多个handler进行比较，最后进行整合
3. 调用engine的addRoute方法注册路由，首先会从engine中查找对应的method，如果没有找到则新建一个root并添加到原tree中，最后调用tree的addRoute方法添加（处理请求时也会使用tree来查找）