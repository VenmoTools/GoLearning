# ini文件解析
## 简介
该工具为ini文件解析，支持字符串，数字，数组形式

## 使用

test.ini文件内容如下
```ini
[user]
# xcvxcv
username = admin
passwrod = admin123
server = 192.168.1.1
port = 80
needInit = true
needPassword = y
arrInt = [1,2,3,4,5,6]
arrFloat = [1.2,2.1,3.5,4.0,5.1,6]
arrString = [h,cz,a,q,f,g]

[administrator]
username = administrator
# cc
```

使用"::"来表示不同域中的键，例如user域下的username得值可以这样表示`user::username`然后根据对应的类型调用对应的方法，如果你想转为字符串形式，可以调用GetAsString方法数组同理

##### 字符串类型
```go

import "conf"

func main(){
    f,err := NewParser("test.ini")
	if err != nil {
		fmt.Println(err)
	}
	v := f.Get("user::username")
	fmt.Println(v.GetAsString())
}
```

##### 布尔类型

```go

import "conf"

func main(){
    // "1", "t", "T", "true", "True", "ok", "OK", "Ok", "Y", "YES", "yes", "y"均为True
    f,err := NewParser("test.ini")
	if err != nil {
		fmt.Println(err)
	}
    fmt.Println(f.Get("user::needInit").GetAsBool())
	fmt.Println(f.Get("user::needPassword").GetAsBool())
    
}
```

##### 切片类型

```go

func main(){
    f,err := NewParser("/Users/venmosnake/Documents/go/test.ini")
	if err != nil {
		fmt.Println(err)
	}
    // int切片
	arr,err := f.Get("user::arrInt").GetAsIntSlice()
	for _,v :=range arr{
		fmt.Println(v)
    }

    // float切片
    arrF,err := f.Get("user::arrFloat").GetAsFloat64Slice()
	for _,v :=range arrF{
		fmt.Println(v)
    }
    // 字符串切片
    arrS,err := f.Get("user::arrFloat").GetAsStringSlice()
	for _,v :=range arrF{
		fmt.Println(v)
	}
}

```
