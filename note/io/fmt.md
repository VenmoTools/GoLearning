# 格式化IO

## 接口
相关接口定义在fmt/print.go文件中
### Stringer接口
具有String方法会实现Stringer接口，该方法定义该值的“native”格式。 String方法用于将作为操作数传递的值打印到任何接受字符串的格式或打印到未格式化的打印的格式
```go
// 打印实现Stringer接口时默认调用String()方法，类似于Java的toString方法
type Stringer interface {
    String() string
}
```
例如
```go
type Water struct{
    Name string // 名称
    Suger int //甜度
}

func (w *Water)String() string{
    return w.Name+" 甜度:"+w.Suger
}

func main(){
    w := Water{Name:"纯净水",Suger:50}
    fmt.Println(w)
}
```
例如
```go
type Water struct{
    Name string // 名称
    Suger int //甜度
}
func (w *Water) GoString() string {
    return "&Person{Name:"+w.Name+", Suger: "+strconv.Itoa(w.Suger)
}

func main(){
    w := Water{Name:"纯净水",Suger:50}
    fmt.Printf("%#v",w)
}
```

### GoStringer接口
GoStringer由具有GoString方法的任何值实现，该方法定义该值的Go语法。 GoString方法用于将作为操作数传递的值打印为％＃v格式
```go
type GoStringer interface {
	GoString() string
}
```

### Formatter 接口
实现Formatter接口可以使用自定义格式化的值Format的实现可以调用Sprint（f）或Fprint（f）等来生成其输出。
```go
type Formatter interface {
    Format(f State, c rune)
}
```
例如
```go
type Water struct{
    Name string // 名称
    Suger int //甜度
}

func (w *Water)String() string{
    return w.Name+" 甜度:"+w.Suger
}

func (w *Water)Format(f fmt.State,c rune){
    if c == "WA"{
        f.Write([]byte("this is water"))
        f.Write([]byte(w.String())
    }else{
        f.Write([]byte(fmt.Sprintln(w.String())))
    }
}

func main(){
    w := Water{Name:"纯净水",Suger:50}
    fmt.Println(w)
}
```

1. fmt.State 是一个接口。由于Format方法是被fmt包调用的，它内部会实例化好一个fmt.State接口的实例，我们不需要关心该接口；
2. 可以实现自定义占位符，同时fmt包中和类型相对应的预定义占位符会无效。因此例子中Format的实现加上了else子句；
3. 实现了Formatter接口，相应的Stringer接口不起作用。但实现了Formatter接口的类型应该实现Stringer接口，这样方便在Format方法中调用String()方法。就像本例的做法；
4. Format方法的第二个参数是占位符中%后的字母（有精度和宽度会被忽略，只保留字母）；

## 占位符
占位符内容为go语言中文网中内容
https://books.studygolang.com/The-Golang-Standard-Library-by-Example/chapter01/01.3.html

```go
type Website struct {
    Name string
}

// 定义结构体变量
var site = Website{Name:"studygolang"}
```
### 普通占位符
|占位符|说明|举例|输出
|--|--|--|--|
|%v|相应值的默认格式|Printf("%v", site)，Printf("%+v", site)|{studygolang}，{Name:studygolang}   
|%#v|在打印结构体时，“加号”标记（%+v）会添加字段名|Printf("#v", site)|main.Website{Name:"studygolang"}
|%T|使用go语法表示相应值|Printf("%T", site)|main.Website
|%%|字面上的百分号，并非值的占位符|Printf("%%")|% 

### 布尔占位符
|占位符|说明|举例|输出|
|--|--|--|--|
|%t|单词 true 或 false|Printf("%t", true)|true|

### 整数占位符

占位符|说明|举例|输出
|--|--|--|--|
|%b|二进制表示|Printf("%b", 5)|101|
|%c|相应Unicode码点所表示的字符|Printf("%c", 0x4E2D)|中|
|%d|十进制表示|Printf("%d", 0x12)|18|
|%o|八进制表示|Printf("%d", 10)|12|
|%q|单引号围绕的字符字面值，由Go语法安全地转义|Printf("%q", 0x4E2D)|'中'|
|%x|十六进制表示，字母形式为小写 a-f|Printf("%x", 13)|d|
|%X|十六进制表示，字母形式为大写 A-F|Printf("%x", 13)|D|
|%U|Unicode格式：U+1234，等同于 "U+%04X"|Printf("%U", 0x4E2D)|U+4E2D|

### 浮点数和复数的组成部分（实部和虚部）

|占位符|说明|举例|输出|
|--|--|--|--|
|%b|无小数部分的，指数为二的幂的科学计数法，与 strconv.FormatFloat的 'b' 转换格式一致。|-123456p-78||
|%e|科学计数法，例如 -1234.456e+78|Printf("%e", 10.2)|1.020000e+01|
|%E|科学计数法，例如 -1234.456E+78|Printf("%e", 10.2)|1.020000E+01
|%f|有小数点而无指数，例如 123.456|Printf("%f", 10.2)|10.200000|
|%g|根据情况选择 %e 或 %f 以产生更紧凑的（无末尾的0）输出|Printf("%g", 10.20)|10.2|
|%G|根据情况选择 %E 或 %f 以产生更紧凑的（无末尾的0）输出|Printf("%G", 10.20+2i)|(10.2+2i)|


### 字符串与字节切片
|占位符|说明|举例|输出|
|--|--|--|--|
|%s|输出字符串表示（string类型或[]byte)|Printf("%s", []byte("Go语言中文网"))|Go语言中文网
|%q|双引号围绕的字符串，由Go语法安全地转义|Printf("%q", "Go语言中文网")|"Go语言中文网"|
|%x|十六进制，小写字母，每字节两个字符|Printf("%x", "golang")| 676f6c616e67|
|%X|十六进制，大写字母，每字节两个字符|Printf("%X", "golang")|676F6C616E67

### 指针

|占位符|说明 |举例|输出|
|--|--|--|--|
|%p|十六进制表示，前缀 0x|Printf("%p", &site)|0x4f57f0

### 其它标记

|占位符|说明|举例|输出|
|--|--|--|--|
|+|总打印数值的正负号；对于%q（%+q）保证只输出ASCII编码的字符。|Printf("%+q", "中文")|"\u4e2d\u6587"|
|-|在右侧而非左侧填充空格（左对齐该区域|||
|#|备用格式：为八进制添加前导 0（%#o），为十六进制添加前导 0x（%#x）或0X（%#X），为 %p（%#p）去掉前导 0x；如果可能的话，%q（%#q）会打印原始即反引号围绕的）字符串；如果是可打印字符，%U（%#U）会写出该字符的Unicode 编码形式（如字符 x 会被打印成 U+0078 'x'）。|    Printf("%#U", '中')|U+4E2D '中'|
' '|(空格）为数值中省略的正负号留出空白（% d）；以十六进制（% x, % X）打印字符串或切片时，在字节之间用空格隔开|
|0|填充前导的0而非空格；对于数字，这会将填充移到正负号之后|\|

## 源码

Printf是对Fprintf的一层封装,函数如下
```go
//Fprintf根据格式说明符格式化并写入w。
// 它返回写入的字节数和遇到的任何写入错误。
func Fprintf(w io.Writer, format string, a ...interface{}) (n int, err error) {
	p := newPrinter()
	p.doPrintf(format, a)
	n, err = w.Write(p.buf)
	p.free()
	return
}
```

几个基本的格式化函数都会使用newPrinter()

newPrinter分配一个新的pp结构或使用一个缓存的结构
```go
func newPrinter() *pp {
    // ppFree是由sync.Pool创建的缓冲区
    // 使用时从缓冲区获取
	p := ppFree.Get().(*pp)
	p.panicking = false
    p.erroring = false
    // init用于初始化fmt已经重置fmtFlags
    // fmtFlags放置在单独的结构中以便于清除
	p.fmt.init(&p.buf)
	return p
}
```

#### pp结构

```go
// pp用于存储printer's的状态，并与sync.Pool一起使用以避免分配
type pp struct {
	buf buffer

	// arg使用interface保存当前项，
	arg interface{}

	// 对于反射值，使用值代替arg
	value reflect.Value

    // fmt用于格式化整数或字符串等基本项
    // fmt是Printf等使用的原始格式化程序。它print到必须单独设置的缓冲区中
	fmt fmt

	// 重新排序记录格式字符串是否使用参数重新排序。
	reordered bool
	// goodArgNum记录最近的重新排序指令是否有效。
    goodArgNum bool
    // panic是由catchPanic设置，以避免无限panic，recover，panic，......递归
	panicking bool
	// 在打印错误字符串时设置错误以防止调用handleMethods。
	erroring bool
}

```
#### pp方法

```go
// free保存ppFree中使用的pp结构;避免每次调用分配。
func (p *pp) free() {
	p.buf = p.buf[:0]
	p.arg = nil
	p.value = reflect.Value{}
	ppFree.Put(p)
}

// fmtFlags相关内容
func (p *pp) Flag(b int) bool {
	switch b {
	case '-':
		return p.fmt.minus
	case '+':
		return p.fmt.plus || p.fmt.plusV
	case '#':
		return p.fmt.sharp || p.fmt.sharpV
	case ' ':
		return p.fmt.space
	case '0':
		return p.fmt.zero
	}
	return false
}

// 实现Write，这样我们可以在pp（通过State）上调用Fprintf
func (p *pp) Write(b []byte) (ret int, err error) {
	p.buf.Write(b)
	return len(b), nil
}

// 实现WriteString，以便可以调用io.WriteString
// 在pp（通过state），为了效率。
func (p *pp) WriteString(s string) (ret int, err error) {
	p.buf.WriteString(s)
	return len(s), nil
}

// Xprintf函数的实现
func (p *pp) doPrintf(format string, a []interface{}) {
	end := len(format)
	argNum := 0         // 我们按照non-trivial格式处理一个参数
	afterIndex := false // 格式中的上一项是[3]之类的索引
	p.reordered = false
formatLoop:
	for i := 0; i < end; {
		p.goodArgNum = true
		lasti := i
		for i < end && format[i] != '%' {
			i++
		}
		if i > lasti {
			p.buf.WriteString(format[lasti:i])
		}
		if i >= end {
			// 完成处理格式字符串
			break
		}

		// 处理完一次
		i++

		// 重置flag
		p.fmt.clearflags()
	simpleFormat:
		for ; i < end; i++ {
			c := format[i]
			switch c {
			case '#':
				p.fmt.sharp = true
			case '0':
				p.fmt.zero = !p.fmt.minus // 仅允许向左填充零
			case '+':
				p.fmt.plus = true
			case '-':
				p.fmt.minus = true
				p.fmt.zero = false // 不要用零填充到右边。
			case ' ':
				p.fmt.space = true
			default:
				// 没有精度或宽度或参数索引的ascii小写简单动词的常见情况的快速路径。
				if 'a' <= c && c <= 'z' && argNum < len(a) {
					if c == 'v' {
						// Go syntax
						p.fmt.sharpV = p.fmt.sharp
						p.fmt.sharp = false
						// Struct-field syntax
						p.fmt.plusV = p.fmt.plus
						p.fmt.plus = false
					}
					p.printArg(a[argNum], rune(c))
					argNum++
					i++
					continue formatLoop
				}
				// 格式比简单的标志和动词更复杂或格式错误。
				break simpleFormat
			}
		}

		// 有一个明确的参数索引吗？
		argNum, i, afterIndex = p.argNumber(argNum, format, i, len(a))

		// Do we have width?
		if i < end && format[i] == '*' {
			i++
			p.fmt.wid, p.fmt.widPresent, argNum = intFromArg(a, argNum)

			if !p.fmt.widPresent {
				p.buf.WriteString(badWidthString)
			}

			// 我们有一个负宽度，所以取其值并确保设置减号
			if p.fmt.wid < 0 {
				p.fmt.wid = -p.fmt.wid
				p.fmt.minus = true
				p.fmt.zero = false // Do not pad with zeros to the right.
			}
			afterIndex = false
		} else {
			p.fmt.wid, p.fmt.widPresent, i = parsenum(format, i, end)
			if afterIndex && p.fmt.widPresent { // "%[3]2d"
				p.goodArgNum = false
			}
		}

		// Do we have precision?
		if i+1 < end && format[i] == '.' {
			i++
			if afterIndex { // "%[3].2d"
				p.goodArgNum = false
			}
			argNum, i, afterIndex = p.argNumber(argNum, format, i, len(a))
			if i < end && format[i] == '*' {
				i++
				p.fmt.prec, p.fmt.precPresent, argNum = intFromArg(a, argNum)
				// Negative precision arguments don't make sense
				if p.fmt.prec < 0 {
					p.fmt.prec = 0
					p.fmt.precPresent = false
				}
				if !p.fmt.precPresent {
					p.buf.WriteString(badPrecString)
				}
				afterIndex = false
			} else {
				p.fmt.prec, p.fmt.precPresent, i = parsenum(format, i, end)
				if !p.fmt.precPresent {
					p.fmt.prec = 0
					p.fmt.precPresent = true
				}
			}
		}

		if !afterIndex {
			argNum, i, afterIndex = p.argNumber(argNum, format, i, len(a))
		}

		if i >= end {
			p.buf.WriteString(noVerbString)
			break
		}

		verb, size := rune(format[i]), 1
		if verb >= utf8.RuneSelf {
			verb, size = utf8.DecodeRuneInString(format[i:])
		}
		i += size

		switch {
		case verb == '%': // Percent does not absorb operands and ignores f.wid and f.prec.
			p.buf.WriteByte('%')
		case !p.goodArgNum:
			p.badArgNum(verb)
		case argNum >= len(a): // No argument left over to print for the current verb.
			p.missingArg(verb)
		case verb == 'v':
			// Go syntax
			p.fmt.sharpV = p.fmt.sharp
			p.fmt.sharp = false
			// Struct-field syntax
			p.fmt.plusV = p.fmt.plus
			p.fmt.plus = false
			fallthrough
		default:
			p.printArg(a[argNum], verb)
			argNum++
		}
	}

	// 检查额外的参数，除非调用无序地访问了参数，在这种情况下，检测它们是否全部被使用过于昂贵，如果它们没有被证明是可以的。
	if !p.reordered && argNum < len(a) {
		p.fmt.clearflags()
		p.buf.WriteString(extraString)
		for i, arg := range a[argNum:] {
			if i > 0 {
				p.buf.WriteString(commaSpaceString)
			}
			if arg == nil {
				p.buf.WriteString(nilAngleString)
			} else {
				p.buf.WriteString(reflect.TypeOf(arg).String())
				p.buf.WriteByte('=')
				p.printArg(arg, 'v')
			}
		}
		p.buf.WriteByte(')')
	}
}

func (p *pp) doPrint(a []interface{}) {
	prevString := false
	for argNum, arg := range a {
		isString := arg != nil && reflect.TypeOf(arg).Kind() == reflect.String
		// Add a space between two non-string arguments.
		if argNum > 0 && !isString && !prevString {
			p.buf.WriteByte(' ')
		}
		p.printArg(arg, 'v')
		prevString = isString
	}
}

// doPrintln就像doPrint，但总是在参数和最后一个参数后面的换行符之间添加一个空格
func (p *pp) doPrintln(a []interface{}) {
	for argNum, arg := range a {
		if argNum > 0 {
			p.buf.WriteByte(' ')
		}
		p.printArg(arg, 'v')
	}
	p.buf.WriteByte('\n')
}

// ...剩下的都是针对，整数，字符串，指针print的处理方法

```


