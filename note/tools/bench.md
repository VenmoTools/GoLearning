# Golang性能测试
## 基准测试
基准测试可以测试一段程序的运行性能及耗费 CPU 的程度。Go 语言中提供了基准测试框架，使用方法类似于单元测试，使用者无须准备高精度的计时器和各种分析工具，基准测试本身即可以打印出非常标准的测试报告

基准测试函数的开头为BenchMark,b.N会根据函数不同设定不同的执行次数

测试代码不要使用全局变量等带有记忆性质的数据结构。避免多次运行同一段代码时的环境不一致
```go
func BenchmarkAdd(b *testing.B){
    arr := make(int[],0)
    for i:=0;i<b.N;i++{
        arr = append(arr,55)
    }
}
```

##### 执行命令
```
go test -v -bench=. benchmark_test.go
```

##### 输出结果如下
```
goos: darwin
goarch: amd64
BenchmarkAdd-4          100000000               54.5 ns/op
PASS
ok      command-line-arguments  5.976s
```

`100000000`表示执行的次数，`54.5 ns/op`表示每次操作所花费的时间

### 命令相关参数
|参数|功能|使用|
|:--:|:--:|:--:|
|-benchtime|自定义测试时间|`go test -v -bench=. -benchtime=5s benchmark_test.go`
|-benchmem|显示内存分配情况(16 B/op”表示每一次调用需要分配 16 个字节,2 allocs/op”表示每一次调用有两次分配)|`go test -v -bench=. -benchmem benchmark_test.go`
|-cpuprofile|生成性能测试结果文件|`go test -v -bench=BenchmarkAdd bench_test.go -cpuprofile=cpu.prof`|



### 控制计时器

|函数名|说明|
|:--:|:--:|
|ResetTimer()|重置计时器|
|StopTimer()|停止计时器|
|StartTimer()|开始计时器|

## pprof
go pprof 可以帮助开发者快速分析及定位各种性能问题，如 CPU 消耗、内存分配及阻塞分析。

使用pprof之前需要安装Graphviz

下载地址：http://www.graphviz.org/download/


### 生成pprof文件

```
go test -v -bench=. bench_test.go -cpuprofile=cpu.prof
```

查看内容
```
go tool pprof -http=:8080 cpu.prof 
```
