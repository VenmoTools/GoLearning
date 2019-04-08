# go语言笔记
本项目为go学习学习是做的笔记，分为两个部分，note部分主要以源码分析和使用方式为主，project主要是实际应用,源码探索除了go源码以外均为自己完成，原理部分很多文章写的很不错，所以拿过来了（标明了出处），参考资源部分为源码分析以及原理参考以及借鉴的资源

使用的go版本以及环境 go version go1.11 darwin/amd64 | mac os

# 目录
## go标准库探索
+ [Context](note/lib/context.md)
## 网络编程
+ [Web服务器](note/web/server.md)
+ [请求与响应](note/web/request&Resp.md)
+ [HTTPClient](wnote/web/client.md)
## 并发编程(待完成)
+ [并发概念](note/concurrency/概念.md)
+ [并发模式](note/concurrency/pattern.md)
## I/O
+ [基本接口](note/io/interface.md)
+ [IO函数](note/io/interface.md)
+ [ioutil](note/io/ioutil.md)
+ [bufio](note/io/bufio.md)(待完成)
+ [格式化IO](note/io/fmt.md)

## 工具链(待完成)
### 编译
### 性能分析
### 调试(待完成)
+ [Delve](note/tools/delve.md)

## 框架源码探索
## beego
+ [beego启动流程](note/beego/start.md)
+ [beegoHttpLib](note/beego/httplib.md)
+ [context](note/beego/context.md)
+ [beego多路复用器](note/beego/router.md)
+ [注解路由](note/beego/parser.md)
+ [beegoORM](note/beego/orm.md)(待完成)

## Gin(待完成)

## Iris(待完成)


## go源码探索
###  内存管理
+ [TCmalloc](note/memory/TCMalloc介绍.md)
+ [内存分配](note/memory/内存分配.md)(待完成)
+ [逃逸分析](note/memory/逃逸分析.md)
+ [内存分配原理]()(待完成)
### 垃圾回收(待完成)

### 调度(待完成)

### 通道(待完成)
+ [Channel](note/chan/chan.md)

PS:源码暂时不更了。。。真看不动了。。头发都要掉光了

## 项目
+ [并发字典(哈希版)]()(待完成)
+ [并发字典(红黑树版)]()(待完成)
+ [通道版爬虫](project/down/README.md)
+ [ini文件解析库](project/conf/README.md)
+ [多路复用器]()(待完成)
+ [爬虫框架](project/spider/README.md)
+ [分布式缓存]()(待完成)

# 参考资源
+ [go语言学习笔记](https://book.douban.com/subject/26832468/)
+ [go Web编程](https://wizardforcel.gitbooks.io/build-web-application-with-golang/content/)
+ [go语言高级编程](https://books.studygolang.com/advanced-go-programming-book/)
+ http://legendtkl.com/2017/04/02/golang-alloc/
+ http://goog-perftools.sourceforge.net/doc/tcmalloc.html
+ https://studygolang.com/articles/12444
+ https://books.studygolang.com/The-Golang-Standard-Library-by-Example/
