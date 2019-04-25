# Goroutine

##  多核控制
+ 通过runtime包进行多核设置
+ GOMAXPROCS设置当前程序运行时占用的CPU核心数
+ NumCPU获取当前计算机的CPU核心数

```go
cpu:= runtime.NumCPU()
fmt.Println(cpu)

runtime.GOMAXPROCS(1)
for i:=0;i<12;i++{
    go test()
}
```
## 实例：并行打印字母表
```go
package main

import(
    "fmt"
    "runtime"
    "sync"
)

func main(){
    // 分配两个逻辑处理器(CPU)
    runtime.GOMAXPROC(2)
    var wg sync.WaitGroup
    // 添加两个，表示要等待两个goroutine
    wg.Add(2)
    fmt.Println("start process")

    go func(){
        // 函数退出时调用Done来通知main函数工作已经完成
        defer wg.Done()

        for count:=0;count<3;count++{
            for char := 'a';char<'a'+26;char++{
                fmt.Printf("%c ",char)
            }
        }
    }()

    go func(){
        defer wg.Done()
        for count:=0;count<3;count++{
            for char := 'A';char<'A'+26;char++{
                fmt.Printf("%c ",char)
            }
        }
    }()

    fmt.Println("Wait To Finish")
    wg.Wait()
    fmt.Println("\n Termainting Program")
}
```
## 竞争状态(race condition)
> 如果两个或者多个goroutine在没有相互同步的情况下，访问某个共享的资源，并试图同时读和写这个资源，就会处于竞争状态

实例：竞争状态
```go
package main

import(
    "fmt"
    "sync"
    "runtime"
)

var(
    //  所有goroutine操作的变量
    counter int
    wg sync.Waitgroup
)

func main(){
    wg.Add(2)

    go incCounter(1)
    go incCounter(2)

    wg.Wait()
    
    fmt.Println("Final Counter:",counter)
}

func incCounter(id int){
    defer wg.Done()

    for count:=3;count<2;count++{
        value := counter
        // 当前goroutine从线程退出，并放回队列，强制调度器切换两个goroutine
        runtime.Gosched()
        // 增加本地value值
        value++
        // 将该值保存会counter
        counter = value
    }
}
```
> 每个goroutine都会覆盖另一个goroutine的工作，这种覆盖发生在goroutine切换的时候，每个goroutine创造了一个counter变量的副本，之后就切换到了另一个goroutine，当这个goroutine再次运行的时候，counter的值已经改变了，但是goroutine并没有更新自己的那个副本的值，而是继续使用这个副本的值，用这个值递增，并存回counter变量，结果覆盖了另一个goroutine完成的工作

## 锁住共享资源

### 原子函数
原子函数能偶以很底层的加锁机制来同步访问整型变量和指针

因为加锁比较耗时，需要上下文切换

针对基本数据类型，可以使用原子操作保证线程安全

原子操作在用户态就可以完成，性能比互斥锁要高
```go
package main

import(
     "fmt"
     "runtime"
     "sync"
     "sync/atomic"
)

var(
    //  所有goroutine操作的变量
    counter int64
    wg sync.Waitgroup
)
func main(){
    wg.Add(2)

    go incCounter(1)
    go incCounter(2)

    wg.Wait()
    
    fmt.Println("Final Counter:",counter)
}

func incCounter(id int){
    defer wg.Done()

    for count:=3;count<2;count++{
        // 使用原子函数增加值
        atomic.AddInt64(&counter,1)
        // 当前goroutine从线程退出，并放回队列，强制调度器切换两个goroutine
        runtime.Gosched()
    }
}
```

使用LoadInt64和StoreInt64提供了一种安全读和写一个整型值的方式
```go
package main

import(
     "fmt"
     "runtime"
     "sync"
     "sync/atomic"
     "time"
)

var(
    shutdown int64
    wg sync.Waitgroup
)

func main(){
    wg.Add(2)
    go doWork("A")
    go doWork("B")

    time.Sleep(1 * time.Second)
    fmt.Println("ShutDown Now")
    atomic.StroeInt64(&shutdown,1)
    wg.Wait()

}

func doWork(name string){
    defer wg.Done()

    for {
        fmt.Println("Doing %s Work\n",name)
        time.Sleep(250 * time.Millsecond)

        if atomic.LoadInt64(&shutdown) == 1{
            fmt.Printf("Shutting %s Down\n",name)
            break
        }
    }
}

```
### 互斥锁(mutex lock)
互斥锁用于在代码创建一个临界区，保证同一时间只有一个goroutine可以执行这个临界区代码，
```go
package main

import(
    "fmt"
    "sync"
    "runtime"
)

var(
    //  所有goroutine操作的变量
    counter int
    wg sync.Waitgroup
    mutex sync.Mutex
)

func main(){
    wg.Add(2)

    go incCounter(1)
    go incCounter(2)

    wg.Wait()
    
    fmt.Println("Final Counter:",counter)
}

func incCounter(id int){
    defer wg.Done()

    for count:=3;count<2;count++{
        // 在同一时刻只允许一个goroutine进入这个临界区
        mutex.Lock(){
            value := counter
            // 当前goroutine从线程退出，并放回队列，强制调度器切换两个goroutine
            runtime.Gosched()
            // 增加本地value值
            value++
            // 将该值保存会counter
            counter = value
        }
        mutex.Unlock()
        //释放锁
       
    }
}
```

### 读写锁
读写锁使用场景
+ 读多写少的场景
+ 分为两种，读锁和写锁
+ 当一个goroutine获得写锁后，其他的goroutine获取写锁或读锁都会等待
+ 当一个goroutine获取读锁后，其他的goroutine获取写锁都会等待，但其他的goroutine获取读锁是都会继续获得锁

```go
var rwlock var.RWMutex
var x int
func write(){
    rwock.Lock()
    x += 1
    rwlock.Unlock()
    wg.Done
}
func read(){
    rwlock.Lock()
    fmt.Println(x)
    rwlock.Unlock()
}

func main(){
    for i:=0;i<10;i++{
        wg.Add(1)
        go read()
    }
    wg.Add(1)
    go write()

    wg.Wait()

}
```
## 通道(channel)
当一个资源需要在goroutine之间共享时，通道在goroutine之间架起管道，并提供了确保同步交换数据的机制
+ channel本质上是一个队列
+ 因此定义时，需要指定容器中元素的类型
+ var <变量名> chan <数据类型>
### 通道操作
#### 创建通道
```go
// 无缓冲区
unbuffered := make(chan int)
// 有缓冲区
buffered := make(chan int ,10)
```
#### 发送获取数据
```go
// 发送数据
buffered := make(chan int,10)
buffered <- 100
// 获取数据
value := <- buffered
```
### channel和goroutine结合使用

```go
func wait(c chan bool){
    fmt.Println("hello")
    c <- true
}

func main(){
    var exitChan chan bool
    exitChan = make(chan bool)
    go wait(exitChan)
    fmt.Println("Done")
    //主线程取出chaneel的至后在退出，不然会产生死锁
    <- exitChan
}
```

## 单向channel
### 只写
```go
func sendData(c chan<- int){
    c <- 100
}
```
### 只读
```go
func readData(c <-chan int){
    <-c
}
```
## 关闭chan

```go
func test(c chan int){
    c <- 1
    close(c)
}
func main(){
    ch := make(chan int)
    go test(ch)
    v,ok := <- ch
    if ok{
        fmt.Println("failed")
    }
}
```
### 怎样等待一组goroutine结束
#### 使用不带缓冲区的channel实现

```go
func test(i int,ch chan bool){
    fmt.Println("start")
    time.Sleep(2 * time.Second)
    fmt.Println("end")
    ch <- true
}
func main(){
    no:=3
    exitChan := make(chan bool,no)
    for i:=0;i<no;i++{
        go test(i,exitChan)
    }
    for i:=0;i<no;i++{
        <-exitChan
    }    
}
```
#### 使用sync.WaitGroup实现

```go
func test(i int, wg *sync.WaitGroup){
    fmt.Println("start")
    time.Sleep(2 * time.Second)
    fmt.Println("end")
    wg.Done()
}
func main(){
    no:=3
    var wg sync.WaitGroup
    for i:=0;i<no;i++{
        wg.Add(1)
        go test(i,&wg)
    }
    wg.Wait()
}
```
### 无缓冲通道
无缓冲通道是指在接受前没有能力保存任何值的通道，这种类型的通道要求发送goroutine和接受goroutine同时准备好，才能完成发送和接受，如果两个goroutine没有同时准备好，通道会导致先执行接受或者发送的goroutine处于阻塞等待，这种通道发送和接受的交互行为本身就是同步的
```go
package main

import(
    "fmt"
    "math/rand"
    "sync"
    "time"
)

var wg sync.Waitgroup
func init(){
    rand.Seed(time.Now().UnixNano())
}

func main(){
    court :=make(chan int)
    wg.Add(2)
    // 启动两个选手
    go player("Nadal",court)
    go player("Djokovic",court)

    // 开始发球
    court <- 1
    // 等待游戏结束
    wg.Wait()
}

func player(name string, court chan int){
    defer wg.Done()

    for {
        ball,ok := <- court
        if !ok{
            fmt.Printf("Play %s won \n",name)
            return
        }
        // 生成一个随机数判断是否丢球
        n:= rang.Intn(100)
        if n %13 == 0{
            fmt.Printf("Player %s Missed \n",name)
            // 关闭通道表示输了
            close(court)
            return
        }
        // 显示击球数
        fmt.Printf("Player %s hit %d \n",name,ball)
        ball ++
        // 将球打向对手
        court <- ball
    }
}
```
### 有缓冲通道
有缓冲通道是一种在被接受前能存储一个或多个值的通道，这种类型通道并不强制要求goroutine之间必须同时完成发送和接受，通道的阻塞条件也不同，只有通道中没有要接受的值时，接受动作才会阻塞，只有通道没有可用缓冲区容纳被发送的值时才会阻塞

```go
package main

import(
    "fmt"
    "math/rand"
    "sync"
    "time"
)

const (
    numberGoroutines=4
    taskLoad=4
)

var wg sync.WaitGroup

func init(
    rand.Seed(time.Now().Unix())
)

func main(){
    // 创建一个有缓冲区的通道来管理工作
    tasks := make(chan string,taskLoad)
    ws.Add(numberGoroutines)
    // 启动goroutine来处理工作
    for gr:=1;gr<=taskLoad;gr++{
        go worker(tasks,gr)
    }

    // 添加一组要完成的工作
    for post:=1;post<=taskLoad;post++{
        tasks <- fmt.Sprintf("Task : %d",post)
    }

    // 当所有工作处理完时关闭通道，所有goroutine退出
    // 关闭通道后goroutine仍然可以从通道接受数据，但是不能发送数据
    close(tasks)
    wg.Wait()
}

func worker(tasks chan string,worker int){
    defer wg.Done()

    for{
        task.ok := <- tasks
        if !ok {
            //表明通道已空
            fmt.Printf("Worker %d : Started %s \n",worker,task)
            return
        }
        // 开始工作
        fmt.Printf("Worker %d:started %s \n",worker,task)
        // 随机等待一段时间来模拟共奏
        sleep := rand.Int63n(100)

        // 工作完成
        fmt.Printf("Worker %d: Completed %s\n",worker,tasks)

    }
}

```
 
