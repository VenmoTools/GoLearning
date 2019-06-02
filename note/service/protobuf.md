# protobuf

## 安装

```
go get -u github.com/golang/protobuf/protoc-gen-go
```


## 创建protobuf文件

创建user.proto文件

### 定义使用的版本，以及包名

可以为.proto文件新增一个可选的package声明符，用来防止不同的消息类型有命名冲突
```proto
// 定义使用的版本位proto3
syntax="proto3"
// 定义包名为micro.user
package micro.user;
```

### 定义消息

例如需要定义用户信息的消息格式其中包含用户名，密码，性别等可以这样定义

```proto
message User{
    // 定义字符串类型的username, 使用required修饰符
    required string username = 1; // 1为识别号
    // 定义字符串类型的password，使用required修饰符
    required string password = 2;
    // 定义字符串类型的sex
    string sex = 3;
}
```

**识别号**

>在消息定义中，每个字段都有唯一的一个标识符。这些标识符是用来在消息的二进制格式中识别各个字段的，一旦开始使用就不能够再改 变。注：[1,15]之内的标识号在编码的时候会占用一个字节。[16,2047]之内的标识号则占用2个字节。所以应该为那些频繁出现的消息元素保留 [1,15]之内的标识号。切记：要为将来有可能添加的、频繁出现的标识号预留一些标识号<br>
最小的标识号可以从1开始，最大到229 - 1, or 536,870,911。不可以使用其中的[19000－19999]的标识号， Protobuf协议实现中对这些进行了预留。如果非要在.proto文件中使用这些预留标识号，编译时就会报警。

**修饰符**
>singular：一个格式良好的消息应该有0个或者1个这种字段（但是不能超过1个）。<br>
repeated：在一个格式良好的消息中，这种字段可以重复任意多次（包括0次）。重复的值的顺序会被保留。<br>
在proto3中，repeated的标量域默认情况虾使用packed。


**默认值**

当一个消息被解析的时候，如果被编码的信息不包含一个特定的singular元素，被解析的对象锁对应的域被设置位一个默认值，对于不同类型指定如下：
+ 对于strings，默认是一个空string
+ 对于bytes，默认是一个空的bytes
+ 对于bools，默认是false
+ 对于数值类型，默认是0
+ 对于枚举，默认是第一个定义的枚举值，必须为0;
+ 对于消息类型（message），域没有被设置，确切的消息是根据语言确定的，详见generated code guide
+ 对于可重复域的默认值是空（通常情况下是对应语言中空列表）。

注：对于标量消息域，一旦消息被解析，就无法判断域释放被设置为默认值（例如，例如boolean值是否被设置为false）还是根本没有被设置。你应该在定义你的消息类型时非常注意。例如，比如你不应该定义boolean的默认值false作为任何行为的触发方式。也应该注意如果一个标量消息域被设置为标志位，这个值不应该被序列化传输。

**注意**
> Optional和required修饰符在protobuf3中不可用

### 枚举

当需要定义一个消息类型的时候，可能想为一个字段指定某“预定义值序列”中的一个值。

枚举常量必须在32位整型值的范围内。因为enum值是使用可变编码方式的，对负数不够高效，因此不推荐在enum中使用负数

例如用户的验证状态有已验证，未验证两种状态(Vaild,Invaild)可以使用枚举做到

```protobuf
message User{
    // 定义字符串类型的username
    string username = 1; // 1为识别号
    // 定义字符串类型的password
    string password = 2;
    // 定义字符串类型的sex
    string sex = 3 [default="男"];

    // 创建枚举 Status
    enum Status{
        VAILD = 1001;
        INVAILD = 1002;
    }
    // 定义状态，默认为Invaild
    Status status = 4 [default = INVAILD]
}
```

### 映射

```proto
message User{
    // 每一个string对应一个GIFT
    map<string,Gift> users = 1;
}

message Gift{
    string name = 1;
    enmu Type{
        APPLE = 1;
        BANANA = 2;
    }
    Type type = 2;
}
```

### 消息嵌套 

在一个消息中使用另一个消息，例如：通过验证后有响应结果消息

```proto
message Response{
    int32 code = 1 [default = 200];
    Message msg = 2;
}

message Message{
    string msg = 1;
    string token = 2;
}
```

在其他消息类型中定义、使用消息类型

```proto
message Response{
    int32 code = 1 [default = 200];
    message Message{
        required string msg = 1;
        required string token = 2;
    }   
    Message msg = 2;
}
```
如果你想在它的父消息类型的外部重用这个消息类型

```proto
message OtherResponse{
     Response.Message msg = 1;
}
```

### 导入定义

可以通过导入（importing）其他.proto文件中的定义来使用它们。要导入其他.proto文件的定义，你需要在你的文件中添加一个导入声明

例如:
./proto/msg.proto
```proto
syntax = "proto3";
package micro.msg;

message Message{
     string msg = 1;
     string token = 2;
}
```

./proto/response.proto
```proto
syntax = "proto3";
package micro.resp;

import "proto/response.proto'

message Response{
     int32 code = 1 [default = 200];
     Message msg = 2;
}
```

protocol编译器就会在一系列目录中查找需要被导入的文件，这些目录通过protocol编译器的命令行参数-I/–import_path指定。如果不提供参数，编译器就在其调用目录下查找。


### 更新消息

如果一个已有的消息格式已无法满足新的需求，例如要在消息中添加一个额外的字段——但是同时旧版本写的代码仍然可用

更新消息规则
1. 不要更改任何已有的字段的数值标识。
2. 所添加的任何字段都必须是optional或repeated的(就意味着任何使用“旧”的消息格式的代码序列化的消息可以被新的代码所解析，因为它们 不会丢掉任何required的元素。应该为这些元素设置合理的默认值，这样新的代码就能够正确地与老代码生成的消息交互了。类似地，新的代码创建的消息 也能被老的代码解析：老的二进制程序在解析的时候只是简单地将新字段忽略。然而，未知的字段是没有被抛弃的。此后，如果消息被序列化，未知的字段会随之一 起被序列化——所以，如果消息传到了新代码那里，则新的字段仍然可用。注意：对Python来说，对未知字段的保留策略是无效的。)
3. 非required的字段可以移除（只要它们的标识号在新的消息类型中不再使用，更好的做法可能是重命名那个字段）
4. 一个非required的字段可以转换为一个扩展，反之亦然——只要它的类型和标识号保持不变。
5.  int32, uint32, int64, uint64,和bool是全部兼容的，这意味着可以将这些类型中的一个转换为另外一个，而不会破坏向前、 向后的兼容性。如果解析出来的数字与对应的类型不相符，那么结果就像在C++中对它进行了强制类型转换一样（例如，如果把一个64位数字当作int32来 读取，那么它就会被截断为32位的数字）
6. sint32和sint64是互相兼容的，但是它们与其他整数类型不兼容
7. string和bytes是兼容的，只要bytes是有效的UTF-8编码
8. 嵌套消息与bytes是兼容的，只要bytes包含该消息的一个编码过的版本
9. fixed32与sfixed32是兼容的，fixed64与sfixed64是兼容的

### 扩展

通过扩展，可以将一个范围内的字段标识号声明为可被第三方扩展所用。然后，其他人就可以在他们自己的.proto文件中为该消息类型声明新的字段，而不必去编辑原始文件了


```proto
syntax = "proto3";
package micro.msg;

message Message{
    string msg = 1;
    string token = 2;
    // Message消息中 范围[10,15]之内的字段标识号被保留为扩展用
    extensions 10 to 15;
}
```

其他人就可以在他们自己的.proto文件中添加新字段到Message里了，但是添加的字段标识号要在指定的范围内


```proto
extend Message{
    // 为Message添加cookie字段
    string cookie = 10;
}
```

### 可扩展的标符号

在同一个消息类型中一定要确保两个用户不会扩展新增相同的标识号，否则可能会导致数据的不一致。可以通过为新项目定义一个可扩展标识号规则来防止该情况的发生

如果标识号需要很大的数量时，可以将该可扩展标符号的范围扩大至max，其中max是229 - 1, 或536,870,911

通常情况下在选择标符号时，标识号产生的规则中应该避开[19000－19999]之间的数字，因为这些已经被Protocol Buffers实现中预留了。

```proto
message Message {
  extensions 1000 to max;
}
```

## 定义服务

如果想要将消息类型用在RPC(远程方法调用)系统中，可以在.proto文件中定义一个RPC服务接口，protocol buffer编译器将会根据所选择的不同语言生成服务接口代码及存根

如，用户登录需要接受传递的用户名密码，并返回响应结果

```proto
syntax="proto3";
package micro.user.login;

message UserInfo{
     string username = 1;
     string password = 2;
}

message Response{
    int32 code = 1;
    string msg = 2 [default="Succeed"];
}

// 定义服务接口
service UserService{
    rpc Login(UserInfo) returns (Response);
}


```

protocol编译器将产生一个抽象接口UserService以及一个相应的存根实现。存根将所有的调用指向RpcChannel，它是一 个抽象接口，必须在RPC系统中对该接口进行实现，产生的存根提供了一个类型安全的接口用来完成基于protocolbuffer的RPC调用，而不是将你限定在一个特定的RPC的实现中

user.pb.go

```go
// Code generated by protoc-gen-go. DO NOT EDIT.
// source: test.proto

package micro_user_login

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type UserInfo struct {
	Username             string   `protobuf:"bytes,1,opt,name=username,proto3" json:"username,omitempty"`
	Password             string   `protobuf:"bytes,2,opt,name=password,proto3" json:"password,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UserInfo) Reset()         { *m = UserInfo{} }
func (m *UserInfo) String() string { return proto.CompactTextString(m) }
func (*UserInfo) ProtoMessage()    {}
func (*UserInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_c161fcfdc0c3ff1e, []int{0}
}

func (m *UserInfo) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UserInfo.Unmarshal(m, b)
}
func (m *UserInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UserInfo.Marshal(b, m, deterministic)
}
func (m *UserInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UserInfo.Merge(m, src)
}
func (m *UserInfo) XXX_Size() int {
	return xxx_messageInfo_UserInfo.Size(m)
}
func (m *UserInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_UserInfo.DiscardUnknown(m)
}

var xxx_messageInfo_UserInfo proto.InternalMessageInfo

func (m *UserInfo) GetUsername() string {
	if m != nil {
		return m.Username
	}
	return ""
}

func (m *UserInfo) GetPassword() string {
	if m != nil {
		return m.Password
	}
	return ""
}

type Response struct {
	Code                 int32    `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Msg                  string   `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Response) Reset()         { *m = Response{} }
func (m *Response) String() string { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()    {}
func (*Response) Descriptor() ([]byte, []int) {
	return fileDescriptor_c161fcfdc0c3ff1e, []int{1}
}

func (m *Response) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Response.Unmarshal(m, b)
}
func (m *Response) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Response.Marshal(b, m, deterministic)
}
func (m *Response) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Response.Merge(m, src)
}
func (m *Response) XXX_Size() int {
	return xxx_messageInfo_Response.Size(m)
}
func (m *Response) XXX_DiscardUnknown() {
	xxx_messageInfo_Response.DiscardUnknown(m)
}

var xxx_messageInfo_Response proto.InternalMessageInfo

func (m *Response) GetCode() int32 {
	if m != nil {
		return m.Code
	}
	return 0
}

func (m *Response) GetMsg() string {
	if m != nil {
		return m.Msg
	}
	return ""
}

func init() {
	proto.RegisterType((*UserInfo)(nil), "micro.user.login.UserInfo")
	proto.RegisterType((*Response)(nil), "micro.user.login.Response")
}

func init() { proto.RegisterFile("test.proto", fileDescriptor_c161fcfdc0c3ff1e) }

var fileDescriptor_c161fcfdc0c3ff1e = []byte{
	// 175 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2a, 0x49, 0x2d, 0x2e,
	0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12, 0xc8, 0xcd, 0x4c, 0x2e, 0xca, 0xd7, 0x2b, 0x2d,
	0x4e, 0x2d, 0xd2, 0xcb, 0xc9, 0x4f, 0xcf, 0xcc, 0x53, 0x72, 0xe2, 0xe2, 0x08, 0x2d, 0x4e, 0x2d,
	0xf2, 0xcc, 0x4b, 0xcb, 0x17, 0x92, 0xe2, 0xe2, 0x00, 0xc9, 0xe4, 0x25, 0xe6, 0xa6, 0x4a, 0x30,
	0x2a, 0x30, 0x6a, 0x70, 0x06, 0xc1, 0xf9, 0x20, 0xb9, 0x82, 0xc4, 0xe2, 0xe2, 0xf2, 0xfc, 0xa2,
	0x14, 0x09, 0x26, 0x88, 0x1c, 0x8c, 0xaf, 0x64, 0xc0, 0xc5, 0x11, 0x94, 0x5a, 0x5c, 0x90, 0x9f,
	0x57, 0x9c, 0x2a, 0x24, 0xc4, 0xc5, 0x92, 0x9c, 0x9f, 0x02, 0xd1, 0xcf, 0x1a, 0x04, 0x66, 0x0b,
	0x09, 0x70, 0x31, 0xe7, 0x16, 0xa7, 0x43, 0xb5, 0x81, 0x98, 0x46, 0x7e, 0x5c, 0xdc, 0x20, 0x5b,
	0x83, 0x53, 0x8b, 0xca, 0x32, 0x93, 0x53, 0x85, 0xec, 0xb9, 0x58, 0x7d, 0x40, 0xae, 0x11, 0x92,
	0xd2, 0x43, 0x77, 0xa0, 0x1e, 0xcc, 0x75, 0x52, 0x58, 0xe4, 0x60, 0xb6, 0x26, 0xb1, 0x81, 0xbd,
	0x67, 0x0c, 0x08, 0x00, 0x00, 0xff, 0xff, 0xd0, 0xcf, 0x7a, 0x9e, 0xec, 0x00, 0x00, 0x00,
}
```

