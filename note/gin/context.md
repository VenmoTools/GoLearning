# context

context作为请求的上下文，每次请求都会使用该对象，因此将context放入到sync.pool进行对象复用

## 结构

```go
type Context struct {
    // responseWriter是对http.ResponseWriter的进一步封装
    // 包含了WriteHeader（状态码）WriteHeaderNow()，Write(与ResponseWriter一样，添加了写入状态码操作)
    writermem responseWriter
    // 在ServeHTTP中将会被赋值，用于存储请求信息
    Request   *http.Request
    // ResponseWriter是一个接口继承了responseWriterBase（responseWriter实现了该接口）
	Writer    ResponseWriter
    // param是一个URL参数，由一个键和一个值组成。Params是一个param切片
    Params   Params
    
    // HandlersChain是一个HandleFunc切片用于存储请求处理器函数
    handlers HandlersChain
    // index用于handlers的下标，通过该下标调用相应函数
    index    int8
    // engine实例
	engine *Engine

	// Keys是专门用于每个请求的上下文的键/值对
	Keys map[string]interface{}

	// Errors是附加到使用此上下文的所有处理程序/中间件的错误列表
	Errors errorMsgs

	// Accepted defines a list of manually accepted formats for content negotiation.
	Accepted []string
}

```

## 方法

### reset

reset方法在context使用之前进行重置，不会收到上一个请求的影响，在另一方面来说reset方法也是创建context的方法
```go
func (c *Context) reset() {
	c.Writer = &c.writermem
	c.Params = c.Params[0:0]
	c.handlers = nil
	c.index = -1
	c.Keys = nil
	c.Errors = c.Errors[0:0]
	c.Accepted = nil
}
```

### Next
每次middleware中流程完毕后将会调用Next方法,在middleware中使用

```go
// Next应该只在中间件中使用。它在调用处理程序内的链中执行挂起的处理程序。
// 循环调用handlers中的方法
func (c *Context) Next() {
	c.index++
	for c.index < int8(len(c.handlers)) {
		c.handlers[c.index](c)
		c.index++
	}
}
```

### Abort
```go
// abort可防止调用挂起的处理程序。请注意，这不会停止当前处理程序。
// 假设您有一个授权中间件，用于验证当前请求是否已获得授权。
// 如果授权失败（例如：密码不匹配），请调用Abort以确保不会调用此请求的其余处理程序。
func (c *Context) Abort() {
	c.index = abortIndex
}
```

## 基本元素处理

GetString,GetBool，GetInt，GetInt64，GetFloat64，GetTime，GetDuration，GetStringSlice，GetStringMap，GetStringMapString，GetStringMapStringSlice等方法均采用Get方法

```go

// Set用于专门为此上下文存储新的键/值对。
// c.Keys如果之前没有使用它，它使用懒加载，
func (c *Context) Set(key string, value interface{}) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = value
}

// 获取返回给定键的值，即：（value，true）
// 如果该值不存在，则返回（nil，false)
func (c *Context) Get(key string) (value interface{}, exists bool) {
	value, exists = c.Keys[key]
	return
}

```

## 获取请求参数

### 获取url中的参数

```go
//router.GET("/user/:id", func(c *gin.Context) {
      // 使用GET请求 /user/john
//         id := c.Param("id") // 则id == "john"
//    })
func (c *Context) Param(key string) string {
	return c.Params.ByName(key)
}
```

Query调用GetQuery，GetQuery调用GetQueryArray，

Query与GetQuery不同的是GetQuery多回一个bool值用于表述该key是否存在


```go

// GET /path?id=1234&name=Manu&value=
// 	   c.Query("id") == "1234"
// 	   c.Query("name") == "Manu"
// 	   c.Query("value") == ""
// 	   c.Query("wtf") == ""
func (c *Context) Query(key string) string {
	value, _ := c.GetQuery(key)
	return value
}

//     GET /?name=Manu&lastname=
//     ("Manu", true) == c.GetQuery("name")
//     ("", false) == c.GetQuery("id")
//     ("", true) == c.GetQuery("lastname")
func (c *Context) GetQuery(key string) (string, bool) {
	if values, ok := c.GetQueryArray(key); ok {
		return values[0], ok
	}
	return "", false
}

func (c *Context) GetQueryArray(key string) ([]string, bool) {
	if values, ok := c.Request.URL.Query()[key]; ok && len(values) > 0 {
		return values, true
	}
	return []string{}, false
}
```