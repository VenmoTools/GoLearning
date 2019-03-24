# Context
Context用来简化在多个go routine传递上下文数据、(手动/超时)中止routine树等操作,官方http包使用context传递请求的上下文数据，gRpc使用context来终止某个请求产生的routine树。由于它使用简单，现在基本成了编写go基础库的通用规范，使用时清楚context在什么时候cancel，什么行为会触发cancel。

Context在不同的函数或者goroutine中传递，每次传递时使用WithCancel，WithTimeout，WithValue，WithDeadline，等函数创建一个上下文，上下文中包含了当前运行时的环境或者参数，可以通过调用生成的取消函数来停止context


## 接口
### Context
```go
//对服务器的传入请求应创建一个Context，对服务器的传出调用应接受一个Context。它们之间的函数调用链必须传递Context，可选地将其替换为使用WithCancel，WithDeadline，WithTimeout或WithValue创建的派生Context。取消上下文后，也会取消从中派生的所有上下文。
// WithCancel，WithDeadline和WithTimeout函数接受Context（父）并返回派生的Context（子）和CancelFunc。调用CancelFunc会取消子项及其子项，删除父项对子项的引用，并停止任何关联的计时器。未能调用CancelFunc会泄漏子项及其他子项，直到取消父项或计时器触发。 go vet工具检查CancelFuncs是否在所有控制流路径上使用。
// 使用上下文的程序应遵循这些规则，以使各接口之间的接口保持一致，并启用静态分析工具来检查上下文传递
//      1.不要将Context存储在结构类型中;相反，将Context明确传递给需要它的每个函数。 Context应该是第一个参数，通常命名为ctx
// 例如 
// func DoSomething(ctx context.Context, arg Arg) error {
// 	 ... use ctx ...
// }

//      2.即使函数允许，也不要传递nil Context。如果您不确定要使用哪个Context，请传递context.TODO。

//      3.仅将Context值用于转换进程和API的请求范围数据，而不是将可选参数传递给函数

//      4.可以将相同的Context传递给在不同goroutine中运行的函数; 多个goroutine可以同时安全使用Context。

// Context的方法可以由多个goroutine同时调用。

// 在请求处理的过程中，会调用各层的函数，每层的函数会创建自己的routine，是一个routine树。所以，context也应该反映并实现成一棵树。
type Context interface {
	// Deadline返回应该取消代表此上下文完成的工作的时间。如果没有设置Deadline，Deadline返回ok == false。对Deadline的连续调用返回相同的结果。
	Deadline() (deadline time.Time, ok bool)

	// 当代表此上下文完成的工作应该被取消时，Done返回一个已关闭的通道。如果永远不能取消此上下文，则Done可能返回nil。对Done的连续调用返回相同的值
	// 当取消被调用时，WithCancel将会关闭Done; WithDeadline将会在Deadline到期时关闭Done，当超时超时时，WithTimeout1将会关闭Done
	//
	// Done用于select语句
	//
	//  // Stream使用DoSomething生成值并将它们发送到out，直到DoSomething返回错误或ctx.Done关闭。
	//  func Stream(ctx context.Context, out chan<- Value) error {
	//  	for {
	//  		v, err := DoSomething(ctx)
	//  		if err != nil {
	//  			return err
	//  		}
	//  		select {
	//  		case <-ctx.Done():
	//  			return ctx.Err()
	//  		case out <- v:
	//  		}
	//  	}
	//  }
	//
	// See https://blog.golang.org/pipelines for more examples of how to use
	// 取消的Done channel
	Done() <-chan struct{}

    // 如果Done尚未关闭，则Err返回nil
    // 如果Done关闭，Err会返回一个非零错误，原因如下
    // 如果Context被取消返回Canceled;如果context的Deadline已过期，则返回DeadlineExceeded
    // 在Err返回非nil错误后，对Err的连续调用返回相同的错误。
	Err() error

	// Value返回与key的此上下文关联的值，如果没有值与key关联，则返回nil。使用相同的键连续调用Value会返回相同的结果。
	//
	// 仅将上下文值用于转换进程和API边界的请求范围数据，而不是将可选参数传递给函数
	//
	// key标识Context中的特定值。希望在Context中存储值的函数通常在全局变量中分配一个键，然后使用该键作为context.WithValue和Context.Value的参数。 key可以是支持相等的任何类型;包应该将键定义为私有类型以避免冲突。
	//
	// Packages that define a Context key should provide type-safe accessors
	// for the values stored using that key:
	//
	// 	// 包用户定义存储在上下文中的用户类型。
	// 	package user
	//
	// 	import "context"
	//
	// 	// User是存储在Contexts.type User struct {...}中的值的类型
	//
	// 	// key是此程序包中定义的键的私有类型.
	// 	// 这可以防止与其他包中定义的键冲突
	// 	type key int
	//
	// 	// userKey是上下文中user.User值的关键字。它是未被出口的;客户端使用user.NewContext和user.FromContext而不是直接使用此key。
	// 	var userKey key
	//
	// 	// NewContext返回一个带有值u的新Context
	// 	func NewContext(ctx context.Context, u *User) context.Context {
	// 		return context.WithValue(ctx, userKey, u)
	// 	}
	//
	// 	// FromContext返回存储在ctx中的User值（如果有）
	// 	func FromContext(ctx context.Context) (*User, bool) {
	// 		u, ok := ctx.Value(userKey).(*User)
	// 		return u, ok
	// 	}
	Value(key interface{}) interface{}
}
```

### canceler
```go
//canceler是可以直接取消的上下文类型。实现是*cancelCtx和*timerCtx。
type canceler interface {
	cancel(removeFromParent bool, err error)
	Done() <-chan struct{}
}
```

## 变量
### Error
```go
// Canceled是Context.Err在取消上下文时返回的错误。
var Canceled = errors.New("context canceled")

// DeadlineExceeded是Context.Err在上下文到期过后返回的错误
var DeadlineExceeded error = deadlineExceededError{}

type deadlineExceededError struct{}

func (deadlineExceededError) Error() string   { return "context deadline exceeded" }
func (deadlineExceededError) Timeout() bool   { return true }
func (deadlineExceededError) Temporary() bool { return true }
```

### 私有定义
```go
var (
	background = new(emptyCtx)
	todo       = new(emptyCtx)
)
```

### 函数定义
```go
// CancelFunc告诉操作放弃其工作.CancelFunc不等待工作停止。第一次调用后，对CancelFunc的后续调用不执行任何操作。
// 调用CancelFunc对象将撤销对应的Context对象，这样父结点的所在的环境中，获得了撤销子节点context的权利，当触发某些条件时，可以调用CancelFunc对象来终止子结点树的所有routine。在子节点的routine中，需要用类似下面的代码来判断何时退出routine： 
// select {
//    case <-cxt.Done():
        // do some cleaning and return
// }
// 根据cxt.Done()判断是否结束。当顶层的Request请求处理结束，或者外部取消了这次请求，就可以cancel掉顶层context，从而使整个请求的routine树得以退出。
type CancelFunc func()
```

## context树
### 创建Context树
```go
// Background返回一个非零的空Context。它永远不会被取消，没有Value，也没有Deadline
// context.Background函数的返回值是一个空的context，经常作为树的根结点，它一般由接收请求的第一个routine创建
// 
func Background() Context {
	return background
}
// TODO返回一个非零的空Context。当不清楚使用哪个Context或者它还不确定时应该使用context.TODO，(因为周围的函数尚未扩展为接受Context参数)，意味着计划将来要添加 context。静态分析工具可识别TODO，以确定上下文是否在程序中正确传递。
func TODO() Context {
	return todo
}
// 创建子节点函数
// WithCancel返回带有新Done通道的父副本。返回的上下文的完成通道在调用返回的取消函数或父上下文的完成通道关闭时关闭，以先最先产生的为准
// 取消此上下文会释放与其关联的资源，因此代码应在此上下文中运行的操作完成后立即调用cancel，只有创建它的函数才能调用取消函数来取消此 context
// 强烈建议不要传递取消函数。这可能导致取消函数的调用者没有意识到取消 context 的下游影响。可能存在源自此的其他 context，这可能导致程序以意外的方式运行。简而言之，永远不要传递取消函数。
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	c := newCancelCtx(parent)
	propagateCancel(parent, &c)
	return &c, func() { c.cancel(true, Canceled) }
}
// WithTimeout返回WithDeadline(parent, time.Now().Add(timeout)) 
// 取消此上下文会释放与之关联的资源，因此代码应在此上下文中运行的操作完成后立即调用cancel：
// 此函数类似于 context.WithDeadline。不同之处在于它将持续时间作为参数输入。此函数返回派生 context，如果调用取消函数或超出超时持续时间，则会取消该派生 context
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

// WithValue返回父级的副本，其中与key关联的值为val。
// 仅将上下文值用于转换进程和API的请求范围数据，而不是将可选参数传递给函数。
// 提供的key必须是可比较的，不应该是字符串类型或任何其他内置类型，以避免使用上下文的包之间的冲突。 WithValue的用户应该为key定义他们自己的类型。为了避免在分配接口{}时分配，上下文键通常具有具体类型struct {}。或者，导出的上下文关键变量的静态类型应该是指针或接口。
func WithValue(parent Context, key, val interface{}) Context {
	if key == nil {
		panic("nil key")
	}
	if !reflect.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &valueCtx{parent, key, val}
}

// WithDeadline返回父上下文的副本，其截止日期调整为不迟于d。如果父母的截止日期早于d
// WithDeadline（parent，d）在语义上等同于parent。返回的上下文的完成通道在截止时间到期，调用返回的取消函数时或父上下文的完成通道关闭时关闭，以先产生的为准
// 当deadline过期而取消该 context 时，获此 context 的所有函数都会收到通知去停止运行并返回。
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
	if cur, ok := parent.Deadline(); ok && cur.Before(d) {
		// The current deadline is already sooner than the new one.
		return WithCancel(parent)
	}
	c := &timerCtx{
		cancelCtx: newCancelCtx(parent),
		deadline:  d,
	}
	propagateCancel(parent, c)
	dur := time.Until(d)
	if dur <= 0 {
		c.cancel(true, DeadlineExceeded) // deadline has already passed
		return c, func() { c.cancel(true, Canceled) }
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.timer = time.AfterFunc(dur, func() {
			c.cancel(true, DeadlineExceeded)
		})
	}
	return c, func() { c.cancel(true, Canceled) }
}

```

## 树的相关操作
### parentCancelCtx
parentCancelCtx沿着父节点，直到找到*cancelCtx。此函数获取context包中的每个具体类型如何表示其父类。
```go

func parentCancelCtx(parent Context) (*cancelCtx, bool) {
	for {
		switch c := parent.(type) {
		case *cancelCtx:
			return c, true
		case *timerCtx:
			return &c.cancelCtx, true
		case *valueCtx:
			parent = c.Context
		default:
			return nil, false
		}
	}
}
```

### propagateCancel
propagateCancel排列在parent的时候取消child。
```go
func propagateCancel(parent Context, child canceler) {
	if parent.Done() == nil {
		return // parent永远不会被取消
	}
	if p, ok := parentCancelCtx(parent); ok {
		p.mu.Lock()
		if p.err != nil {
			// parent has already been canceled
			child.cancel(false, p.err)
		} else {
			if p.children == nil {
				p.children = make(map[canceler]struct{})
			}
			p.children[child] = struct{}{}
		}
		p.mu.Unlock()
	} else {
		go func() {
			select {
			case <-parent.Done():
				child.cancel(false, parent.Err())
			case <-child.Done():
			}
		}()
	}
}
```
### removeChild
removeChild从其父级中删除context
```go
func removeChild(parent Context, child canceler) {
	p, ok := parentCancelCtx(parent)
	if !ok {
		return
	}
	p.mu.Lock()
	if p.children != nil {
		delete(p.children, child)
	}
	p.mu.Unlock()
}

```
# 4种Ctx类型

## emptyCtx
```go
// Background()和TODO()的底层均是emptyCtx
// emptyCtx永远不会被取消，没有值，也没有DeadLine。它不是struct {}，因为此类型的vars必须具有不同的地址
type emptyCtx int

func (*emptyCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (*emptyCtx) Done() <-chan struct{} {
	return nil
}

func (*emptyCtx) Err() error {
	return nil
}

func (*emptyCtx) Value(key interface{}) interface{} {
	return nil
}

func (e *emptyCtx) String() string {
	switch e {
	case background:
		return "context.Background"
	case todo:
		return "context.TODO"
	}
	return "unknown empty Context"
}
```

## cancelCtx

可以取消cancelCtx。取消后，它还会取消任何实现canceler的子项。

cancelCtx结构体中children保存它的所有子canceler， 当外部触发cancel时，会调用children中的所有cancel()来终止所有的cancelCtx。done用来标识是否已被cancel。当外部触发cancel、或者父Context的channel关闭时，此done也会关闭。
```go

type cancelCtx struct {
	Context
	mu       sync.Mutex            // protects following fields
	done     chan struct{}         // 延迟加载，将会被第一个cancel调用关闭
	children map[canceler]struct{} // 第一次调用cancel将被设为nil，派生的context将会存储这里
	err      error                 // 第一次调用cancel将被设为非nil
}

func (c *cancelCtx) Done() <-chan struct{} {
	c.mu.Lock()
	if c.done == nil { 
		c.done = make(chan struct{})
	}
	d := c.done // 防止并发时获取的chan不一致
	c.mu.Unlock()
	return d // 双通道将会转为单通道
}

func (c *cancelCtx) Err() error {
	c.mu.Lock()
	err := c.err // 防止并发时获取的err不一致
	c.mu.Unlock()
	return err
}

func (c *cancelCtx) String() string {
	return fmt.Sprintf("%v.WithCancel", c.Context)
}

// cancel关闭c.done，cancel每个c的子节点，如果removeFromParent为true，则从其父节点的子节点中删除c
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
	if err == nil {
		panic("context: internal error: missing cancel error")
	}
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return // 已经被cancel
	}
	c.err = err
	if c.done == nil {
        /*  
            var closedchan = make(chan struct{})
            func init() {
                close(closedchan)
            }
        */
		c.done = closedchan //closedchan是一个可重复使用的封闭chan
	} else {
		close(c.done)
	}
	for child := range c.children {
		// NOTE: 在持有parent's锁定的同时获取child's锁定。
		child.cancel(false, err)
	}
	c.children = nil
	c.mu.Unlock()

	if removeFromParent {
		removeChild(c.Context, c)
	}
}

// newCancelCtx returns an initialized cancelCtx.
func newCancelCtx(parent Context) cancelCtx {
	return cancelCtx{Context: parent}
}
```

## timerCtx
timerCtx携带一个计时器和一个Deadline。它嵌入了一个cancelCtx来实现Done和Err。它通过停止计时器然后委托给cancelCtx.cancel来实现取消。

timerCtx结构体中deadline保存了超时的时间，当超过这个时间，会触发cancel。
```go

type timerCtx struct {
	cancelCtx
	timer *time.Timer // Under cancelCtx.mu.

	deadline time.Time
}

func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
	return c.deadline, true
}

func (c *timerCtx) String() string {
	return fmt.Sprintf("%v.WithDeadline(%s [%s])", c.cancelCtx.Context, c.deadline, time.Until(c.deadline))
}

func (c *timerCtx) cancel(removeFromParent bool, err error) {
	c.cancelCtx.cancel(false, err)
	if removeFromParent {
		// Remove this timerCtx from its parent cancelCtx's children.
		removeChild(c.cancelCtx.Context, c)
	}
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}
```

## valueCtx

valueCtx带有键值对。它为该key实现Value，并将所有其他调用给valueCtx嵌入。

context上下文数据的存储就像一个树，每个结点只存储一个key/value对。WithValue()保存一个key/value对，它将父context嵌入到新的子context，并在节点中保存了key/value数据。Value()查询key对应的value数据，会从当前context中查询，如果查不到，会递归查询父context中的数据。context中的上下文数据并不是全局的，它只查询本节点及父节点们的数据，不能查询兄弟节点的数据。

```go
type valueCtx struct {
	Context
	key, val interface{}
}

func (c *valueCtx) String() string {
	return fmt.Sprintf("%v.WithValue(%#v, %#v)", c.Context, c.key, c.val)
}

func (c *valueCtx) Value(key interface{}) interface{} {
	if c.key == key {
		return c.val
	}
	return c.Context.Value(key)
}

```

# 使用

## 创建流程
使用时先调用Background()来生成一个顶层节点类型为emptyCtx(首次创建时是使用Background()创建，随后使用propagateCancel()方法创建，例如上层节点是WithCancel，那会将派生的context存储在cancelCtx的children中）

根据这个创建的emptyCtx创建节点函数其字节点，（WithCancel产生cancelCtx节点，WithDeadline，WithTimeout目前的截止日期早于新的截止日期 则会创建ancelCtx节点，否则创建timerCtx节点，WithValue产生valueCtx节点)，四个函数均会返回新添加节点的新emptyCtx和一个CancelFunc，这样在当前的环境(例如函数调用，或者其他goroutine)中就创建了一个上下文

上游持有一个CancelFunc，在需要取消一个操作的时候可以调用该函数，相应的下游中ctx.Done()将会通知操作取消

当调用CancelFunc时会调用各cancelCtx中的cancel方法（以WithCancel为例），其中会传入两个参数一个时，是否需要从root中删除该节点，还有一个就是异常(Canceled，DeadlineExceeded),调用时会将done通道设置为closeChan以代表改context被取消随后关闭他所派生的所有子context（children map[canceler]struct{}字段），然后从root中删除改context


## 最佳实践
1. context.Background 只应用在最高等级，作为所有派生 context 的根。
2. context.TODO 应用在不确定要使用什么的地方，或者当前函数以后会更新以便使用 context。
3. context 取消是建议性的，这些函数可能需要一些时间来清理和退出。
4. context.Value 应该很少使用，它不应该被用来传递可选参数。这使得 API 隐式的并且可以引起错误。取而代之的是，这些值应该作为参数传递。
5. 不要将 context 存储在结构中，在函数中显式传递它们，最好是作为第一个参数。
6. 永远不要传递不存在的 context 。相反，如果您不确定使用什么，使用一个 ToDo context。
7. Context 结构没有取消方法，因为只有派生 context 的函数才应该取消 context。

<i>来自https://studygolang.com/articles/13866?fr=sidebar</i>

## 实例
Github: https://github.com/pagnihotry/golang_samples/blob/master/go_context_sample.go
```go
package main

import (
    "context"
    "fmt"
    "math/rand"
    "time"
)

//Slow function
func sleepRandom(fromFunction string, ch chan int) {
    //defer cleanup
    defer func() { fmt.Println(fromFunction, "sleepRandom complete") }()
    //Perform a slow task
    //For illustration purpose,
    //Sleep here for random ms
    seed := time.Now().UnixNano()
    r := rand.New(rand.NewSource(seed))
    randomNumber := r.Intn(100)
    sleeptime := randomNumber + 100
    fmt.Println(fromFunction, "Starting sleep for", sleeptime, "ms")
    time.Sleep(time.Duration(sleeptime) * time.Millisecond)
    fmt.Println(fromFunction, "Waking up, slept for ", sleeptime, "ms")
    //write on the channel if it was passed in
    if ch != nil {
        ch <- sleeptime
    }
}

//Function that does slow processing with a context
//Note that context is the first argument
func sleepRandomContext(ctx context.Context, ch chan bool) {
    //Cleanup tasks
    //There are no contexts being created here
    //Hence, no canceling needed
    defer func() {
        fmt.Println("sleepRandomContext complete")
        ch <- true
    }()
    //Make a channel
    sleeptimeChan := make(chan int)
    //Start slow processing in a goroutine
    //Send a channel for communication
    go sleepRandom("sleepRandomContext", sleeptimeChan)
    //Use a select statement to exit out if context expires
    select {
    case <-ctx.Done():
        //If context is cancelled, this case is selected
        //This can happen if the timeout doWorkContext expires or
        //doWorkContext calls cancelFunction or main calls cancelFunction
        //Free up resources that may no longer be needed because of aborting the work
        //Signal all the goroutines that should stop work (use channels)
        //Usually, you would send something on channel,
        //wait for goroutines to exit and then return
        //Or, use wait groups instead of channels for synchronization
        fmt.Println("sleepRandomContext: Time to return")
    case sleeptime := <-sleeptimeChan:
        //This case is selected when processing finishes before the context is cancelled
        fmt.Println("Slept for ", sleeptime, "ms")
    }
}

//A helper function, this can, in the real world do various things.
//In this example, it is just calling one function.
//Here, this could have just lived in main
func doWorkContext(ctx context.Context) {
    //Derive a timeout context from context with cancel
    //Timeout in 150 ms
    //All the contexts derived from this will returns in 150 ms
    ctxWithTimeout, cancelFunction := context.WithTimeout(ctx, time.Duration(150)*time.Millisecond)
    //Cancel to release resources once the function is complete
    defer func() {
        fmt.Println("doWorkContext complete")
        cancelFunction()
    }()
    //Make channel and call context function
    //Can use wait groups as well for this particular case
    //As we do not use the return value sent on channel
    ch := make(chan bool)
    go sleepRandomContext(ctxWithTimeout, ch)
    //Use a select statement to exit out if context expires
    select {
    case <-ctx.Done():
        //This case is selected when the passed in context notifies to stop work
        //In this example, it will be notified when main calls cancelFunction
        fmt.Println("doWorkContext: Time to return")
    case <-ch:
        //This case is selected when processing finishes before the context is cancelled
        fmt.Println("sleepRandomContext returned")
    }
}
func main() {
    //Make a background context
    ctx := context.Background()
    //Derive a context with cancel
    ctxWithCancel, cancelFunction := context.WithCancel(ctx)
    //defer canceling so that all the resources are freed up
    //For this and the derived contexts
    defer func() {
        fmt.Println("Main Defer: canceling context")
        cancelFunction()
    }()
    //Cancel context after a random time
    //This cancels the request after a random timeout
    //If this happens, all the contexts derived from this should return
    go func() {
        sleepRandom("Main", nil)
        cancelFunction()
        fmt.Println("Main Sleep complete. canceling context")
    }()
    //Do work
    doWorkContext(ctxWithCancel)
}

```

