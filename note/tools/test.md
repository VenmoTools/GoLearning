# Golang测试

## 单元测试

### 命名格式
文件命名以xxx_test结尾的文件，例如
```
models_test.go
```

每个测试函数都是以Test开头（例如Test_xxx，TestXxxx）,每个测试函数都接受一个*testing.T类型参数，用于输出信息或中断测试

```go
func TestSearch(t *testing.T){

}
```

常用函数
|函数名|说明|
|--|--|
|Fail|标记失败，但继续执行当前测试函数|
|FailNow|失败，立即终止当前测试函数执行|
|Log|输出错误信息|
|Error|Fail + Log|
|Fatal|FailNow + Log|
|Skip|跳过当前函数，通常用于未完成的测试用例|


#### 待测试代码
```go
func Vaild(username string) bool{
    if username == ""{
        return false
    }
    if len(username) < 4 || len(username)>20{
        return false
    }
    return true
}
```
#### 测试代码
```go
// 文件:vaild_test.go
func TestVaild(t *testing.T){
    data := [
        "us",
        "\n",
        " ",
        ""
    ]
    for _,v := range data{
        if Vaild(v){
            t.Fail(fmt.Printf("数据 %s 不应该验证通过",v))
        }
    }
}
```

#### 运行测试
```
go test 
```
执行该命令将会自动搜集所有的测试文件（*_test.go），提取全部测试函数，然后执行

或者可以指定测试文件
```
go test vaild_test.go
```

#### 数据驱动测试
被测函数
```go
func login(username,password string)bool{
    if username == "" || password == ""{
        return false
    }
    if len(username) >20 || len(username) <4{
        return false
    }
    if len(password) >16 || len(password) <8{
        return false
    }
    return true
}
```
```go
// 文件:vaild_test.go
func TestVaild(t *testing.T){
    data := []struct{
        username string
        password string
    }{
        {"ad","12345678"},
        {" ","12345678"},
        {"admin","13"},
        {"admin",""},
    }
    for _,v := range data{
        if login(v.username,v.password){
            t.Fail(fmt.Printf("数据 %v 不应该验证通过",v))
        }
    }
}
```

#### 参数

|参数名|说明|使用|
|--|--|--|
|-v|显示所有测试函数运行细节|`go test -v`|
|-run [funcname]|指定要执行的测试函数（支持正则表达式）|`go test -run TestVaild`
|-c|生成用于运行测试的可执行文件，但不执行它。这个可执行文件会被命名为“[pkg].test”，其中的“pkg”即为被测试代码包的导入路径的最后一个元素的名称。|`go test -c vaild_test.go`|
|-i|安装测试包及其依赖包，但不运行它们|`go test -i vaild_test.go`|
|-o|指定编译测试文件生成的结果名称，此参数还是会运行除非你同时使用 -c -i 标记|`go test -o test vaild_test.go`

#### 日志
|函数|说明|
|--|--|
|Log|打印日志|
|Logf|格式化打印日志|
|Error|打印错误日志|
|Errorf|格式化打印错误日志|
|Fatal|打印致命日志|
|Fatalf|格式化打印致命日志|

#### 指定测试流程

使用WorkFlow指定测试的流程

```go
func TestCaseWorkFlow(t *testing.T){
    // 测试顺序为 B->A->C
    // TestBFunc为流程名称 TestB测试函数
    t.Run("TestBFunc",TestB)
    t.Run("TestAFunc",TestA)
    t.Run("TestCFunc",TestC)
}

func TestA(t *testing.T){
    t.Log("执行测试A")
}
func TestB(t *testing.T){
    t.Log("执行测试B")
}
func TestC(t *testing.T){
    t.Log("执行测试C")    
}

```

执行命令
```
go test -v -run TestCaseWorkFlow
```

结果如下
```
=== RUN   TestCaseWorkFlow
=== RUN   TestCaseWorkFlow/TestBFunc
=== RUN   TestCaseWorkFlow/TestAFunc
=== RUN   TestCaseWorkFlow/TestCFunc
--- PASS: TestCaseWorkFlow (0.00s)
    --- PASS: TestCaseWorkFlow/TestBFunc (0.00s)
        http_test.go:39: 执行测试B
    --- PASS: TestCaseWorkFlow/TestAFunc (0.00s)
        http_test.go:36: 执行测试A
    --- PASS: TestCaseWorkFlow/TestCFunc (0.00s)
        http_test.go:42: 执行测试C
PASS
ok      education/app/student/service   0.029s

```

### 使用断言

依赖包
```
go get -u github.com/stretchr/testify
```

#### 基本断言
每个断言函数第一个参数都会接受一个testing.T对象
每个断言函数都会返回一个bool值来表示是否通过断言
```go 
import 	"github.com/stretchr/testify/assert"

func TestA(t *testing.T){
    // 断言相等
    assert.Equal(t,123,123,"两个应该相等")
    // 断言不相等
    assert.NotEqual(t,123,123,"两数应该不想等")
    var client *http.Client
    // 判断指针是否为nil
    assert.Nil(t,client)
}
```

如果需要断言很多次可以使用如下方式

```go
import 	"github.com/stretchr/testify/assert"

func TestA(t *testing.T){
    asser := assert.New(t)
    // 断言相等
    asser.Equal(123,123,"两个应该相等")
    // 断言不相等
    asser.NotEqual(123,123,"两数应该不想等")
    var client *http.Client
    // 判断指针是否为nil
    asser.Nil(client)
}

```

## Web接口测试

使用httptest来实现对handlers接口函数的单元测试(使用gin框架)


### 定义handler

```go
filename:  login.go
func LoginHandler(ctx *gin.Context){
    username := ctx.PostForm("username")
    password := ctx.PostForm("password")

    if msg:= login(username,password);msg != ""{
        ctx.JSON(http.StatusOk,gin.H{"msg":msg})
        return
    }

    ctx.JSON(http.StatusOk,gin.H{"msg":"登录成功"})
}

func login(username,password) string{
    if strings.TrimSpace(username) == ""{
        return "用户名为空"
    }
    if strings.TrimSpace(password) == ""{
        return "密码为空"
    }

    if password != "12345678"{
        return "密码错误"
    }
}
```

#### 测试

login_test.go
```go

func TestLoginHandler(t *testing.T){
    engine := gin.Deafult()
    engine.POST("/login",LoginHandler)

    data := url.Values{}
    data.Add("username","admin")
    data.Add("password","12345678")
    req := httptest.NewRequest("POST","http://127.0.0.1:8080/login",strings.NewReader(data.Encode()))
    w := httptest.NewRecorder()

    engine.ServeHTTP(w,req)

    res = w.Result()
    defer func(){
        if err = data.Body.Close(); err != nil {
			t.FailNow()
		}
    }
    
    if res.Status != http.StatusOk{
        t.Fatal(fmt.Sprintf("except:200 but given ",res.Status))
    }

    if res == nil{
        t.Fatal(fmt.Sprintf("connection refused ",res.Status))
        t.FailNow()
    }

    data,err := ioutil.ReadAll(res.Body)
    if err != nil{
        t.Fatal(err)
        t.FailNow()
    }

    result := make(map[string]string)
    err := json.Unmarshal(data,result)
    if err != nil{
        t.Fatal(err)
        t.FailNow()
    }

    if msg := result["msg"];msg != "登录成功"{
        t.Fatalf("login failed cause:%s",msg)
    }

}

```