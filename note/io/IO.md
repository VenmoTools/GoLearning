<!-- vscode-markdown-toc -->
* 1. [文件操作](#)
	* 1.1. [通过bufio读取文件](#bufio)
	* 1.2. [通过ioutil读取文件](#ioutil)

<!-- vscode-markdown-toc-config
	numbering=true
	autoSave=true
	/vscode-markdown-toc-config -->
<!-- /vscode-markdown-toc -->
# 文件操作

##  1. <a name=''></a>文件读取

```go
package main

import(
    "bufio"
    "fmt"
    "io"
    "os"
)

func main(){
    file,err := os.Open("./IO.md")
    if err != nil{
        panic(err)
    }
    defer file.Close()

    var content []byte
    // 每次读取128字节
    var buf [128]byte
    // 循环读取
    for {
        n,err := file.Read(buf[:])
        // 如果读到文件末尾
        if err == io.EOF{
            break
        }
        if != nil{
            return
        }
        content = append(content,buf[:n]...)
    }
}
```
###  1.1. <a name='bufio'></a>通过bufio读取文件

```go
package main

import(
    "bufio"
    "fmt"
    "io"
    "os"
)

func main(){
    file,err := os.Open("./IO.md")
    if err != nil{
        panic(err)
    }
    defer file.Close()
    reader := bufio.NewReader(file)
    for{
        // 以\n作为分隔符
        line,err := reader.ReadString("\n")
        if err == io.EOF{
            break
        }
        if err != nil{
            return
        }
        fmt.Println(line)
    }
}
```
###  1.2. <a name='ioutil'></a>通过ioutil读取文件
```go
package main

import(
    "fmt"
    "io/ioutil"
    "os"
)

func main(){
    content,err := ioutil.ReadFile("/神经网络.md")
    if err != nil{
        retunr 
    }
    fmt.Println(string(content∏))
}

```
### 读取gz压缩文件

```go
package main

import(
    "bufio"
    "fmt"
    "io"
    "os"
)

func main(){
    file,err := os.Open("./IO.gz")
    if err != nil{
        panic(err)
    }
    defer file.Close()
    reader,err := gzip.NewReader(file)
    if err != nil{
        return
    }
    var buf [128]byte
    var content []byte
    for {
        n,err := reader.Read(buf[:])
        if err == io.EOF{
            break
        }
        if err != nil{
            return
        }
        content = append(content,buf[:]...)
    }

}
```
## 文件写入

文件打开模式
|    模式     |   含义   |
|:-----------:|:--------:|
| os.O_WRONLY |   只写   |
| os.O_CREATE | 创建文件 |
| os.O_RDONLY |   只读   |
|  os.O_RDWR  |   读写   |
| os.O_TRUNC  |   清空   |
| os.O_APPEND |   追加   |

权限控制
|   权限    | 含义 |
|:---------:|:----:|
|  r(可读)  | 004  |
|  w(可写)  | 002  |
| x(可执行) | 001  |

```go
file,err := os.OpenFile("./IO.md",os.O_CREATE|os.O_WRONLY,0666)

if err != nil{
    return
}
defer file.Close()
str := "hello"
file.Write([]byte(str))
file.WriteString(str)
```
### 通过bufio进行写操作

```go
func main(){
    file,err := os.OpenFile("./IO.md",os.O_CREATE|os.O_WRONLY,0666)
    if err!= nil{
        return
    }
    defer file.Close()
    writer,err := bufio.NewWriter(file)
    writer.WriteString("hello \n")
    writer.Flush()
}
```

### 通过ioutil写操作
```go
func main(){
    str := "hello"
    err := ioutil.WriteFile("./IO.md",[]byte(str),0755)
    if err != nil{
        return 
    }
}
```

## 拷贝文件

```go
 
func Copy(srcName,destName string)(int64,error){
    src,err := os.Open(srcName)
    if err != nil{
        return
    }
    defer src.Close()
    dst,err := os.OpenFile(dstName,os.O_CREATE|os.O_WRONLY,0775)
    if err != nil{
        return
    }
    defer dst.Close()
    return io.Copy(dst,src)
}
```
# 实现Tree命令

```go
func ListDir(dirPath string, deep int)(error){
    dir,err := ioutil.ReadDir(dirPath)
    if err != nil{
        return err
    }
    if deep == 1{
        fmt.Printf("|---%s\n",filePath.Base(dirPath))
    }
    sep := string(os.PathSeparator)
    for _,i := range dri{
        // 如果是文件夹
        if i.IsDir(){
            fmt.Printf("|")
            // 根据当前深度进行打印
            for i:=0;i<deep;i++{
                fmt.Printf("    |")
            }
            fmt.Printf("----%s\n",i.Name())
            // 递归调用
            ListDir(dirpath+sep+i.Name(),deep+1)
            continue
        }
        fmt.Printf("|")
        for i:=0;i<deep;i++{
            fmt.Printf("    |")
        }
        fmt.Printf("----%s\n",i.Name())
    }
    return nil
}
```



