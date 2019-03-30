# context
beego的context是对request，ResponseWriter的抽象，客户端发送到服务器称为Input，服务器的响应称为Output,context提供了大量便捷的API（怎么感觉没啥写的。。）

## Input结构
```go
type BeegoInput struct {
	Context       *Context // 刚开始也是一脸蒙蔽。。这个Context是用于存储重置过后的context，防止使用上一个请求的context
	CruSession    session.Store // 存储的session
	pnames        []string
	pvalues       []string 
    data          map[interface{}]interface{} // 在controller或者filter使用context时候像客户端发送的数据比如模板数据等,
	RequestBody   []byte
	RunMethod     string
	RunController reflect.Type // 当前处理的Controller
}

```

### 初始化/重置
当新的接受到新的请求是将会调用Reset方法来初始化/重置Input内容

```go
func (input *BeegoInput) Reset(ctx *Context) {
	input.Context = ctx //重置过后的Context
	input.CruSession = nil // 清空session
	input.pnames = input.pnames[:0] // []
	input.pvalues = input.pvalues[:0] // []
	input.data = nil // 清空Data
	input.RequestBody = []byte{}
}
```
其余的API都比较简单

## Output结构

```go
type BeegoOutput struct {
	Context    *Context
	Status     int
	EnableGzip bool
}
```

### 初始化/重置
当新的接受到新的请求是将会调用Reset方法来初始化/重置Output内容
```go
func (output *BeegoOutput) Reset(ctx *Context) {
	output.Context = ctx
	output.Status = 0
}
```

## context结构

```go
type Context struct {
	Input          *BeegoInput //客户端发送的请求抽象
	Output         *BeegoOutput //服务器响应的请求抽象
	Request        *http.Request // 在ServeHTTP中有个参数是*http.Request该字段用于保存请求数据
	ResponseWriter *Response // 保存ServeHTTP中的ResponseWriter内容
	_xsrfToken     string // _xsrf防护
}
```

## context重置
每当一各接受的时候请求都会调用Reset方法具体可以查看ControllerRegister中的ServeHTTP方法，其中Response结构是对ResponseWriter的一层封装
```go
func (ctx *Context) Reset(rw http.ResponseWriter, r *http.Request) {
	ctx.Request = r
	if ctx.ResponseWriter == nil {
		ctx.ResponseWriter = &Response{}
	}
	ctx.ResponseWriter.reset(rw)
	ctx.Input.Reset(ctx)
	ctx.Output.Reset(ctx)
	ctx._xsrfToken = ""
}
```