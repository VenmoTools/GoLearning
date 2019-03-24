# go语言笔记
本项目为go学习学习是做的笔记，将其分享出来分为两个部分，笔记部分主要以源码分析和使用方式为主，project主要是实际应用

# 目录
## go源码探索
###  内存管理
+ [TCmalloc](memory/TCMalloc介绍.md)(待完成)
+ [内存分配](memory/内存分配.md)
### 垃圾回收(待完成)

### 调度(待完成)

### 通道(待完成)

## go标准库探索
+ [Context](note/lib/context.md)


## 网络编程
+ [Web服务器](web/server.md)
+ [请求与响应](web/request&Resp.md)
+ [HTTPClient](web/client.md)
## 并发编程(待完成)
+ [并发概念](note/concurrency/概念.md)
+ [并发模式](note/concurrency/pattern.md)
## I/O(待完成)
+ [Reader](note/io/reader.md)
+ [Writer](note/io/writer.md)
## 工具链(待完成)
### 编译
### 性能分析
### 调试(待完成)
+ [Delve](note/tools/delve.md)
## 框架源码探索
## beego
+ [beegoHttpLib](note/beego/httplib.md)
+ [context](note/beego/context.md)
+ [beego多路复用器](note/beego/router.md)
+ [beegoORM](note/beego/orm.md)

## 项目(待完成)
+ [并发字典(哈希版)]()
+ [并发字典(红黑树版)]()
+ [通道版爬虫](project/down/README.md)
+ [ini文件解析库](project/conf/README.md)
+ [多路复用器]()
+ [爬虫框架](project/spider/README.md)
+ [分布式缓存]()

# 参考资源
+ [go语言学习笔记](https://book.douban.com/subject/26832468/)
+ [go Web编程](https://wizardforcel.gitbooks.io/build-web-application-with-golang/content/)
+ [go语言高级编程](https://books.studygolang.com/advanced-go-programming-book/)
+ http://legendtkl.com/2017/04/02/golang-alloc/
+ http://goog-perftools.sourceforge.net/doc/tcmalloc.html
+ https://studygolang.com/articles/12444