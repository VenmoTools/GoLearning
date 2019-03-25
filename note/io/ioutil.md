# NopCloser 函数
```go
// NopCloser返回一个ReadCloser，其中包含一个无操作的Close方法，用于包装提供的Reader r。
// 将Reader -> ReadCloser
func NopCloser(r io.Reader) io.ReadCloser {
	return nopCloser{r}
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }
```
# ReadAll 函数
### 一次性读取io.Reader中的数据
```go
// ReadAll从r读取直到错误或EOF并返回它读取的数据

// 成功的调用返回err == nil，而不是err == EOF。因为ReadAll被定义为从src读取直到EOF，所以它不会将来自Read的EOF视为要报告的错误。
func ReadAll(r io.Reader) ([]byte, error) {
	return readAll(r, bytes.MinRead)
}


// readAll从r读取，直到出现错误或EOF，并返回从分配有指定容量的内部缓冲区读取的数据。
func readAll(r io.Reader, capacity int64) (b []byte, err error) {
	var buf bytes.Buffer
	// 如果缓冲区溢出，将出现bytes.ErrTooLarge.
	// 将其作为错误返回。还有其他panic
	defer func() {
		e := recover() //尝试恢复
		if e == nil {
			return
        }
        // 判断是否为ErrTooLarge，如果不是则panic
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			panic(e)
		}
    }()
    // Grow会增加缓冲区的容量，以保证另外capacity个字节的空间
    // 在Grow（n）之后，可以将至少n个字节写入缓冲区而无需另外分配。如果n为负数，那么Grow会panic。如果缓冲区无法增长，它将会因ErrTooLarge而出现panic
	if int64(int(capacity)) == capacity {
		buf.Grow(int(capacity))
    }
    // ReadFrom从r读取数据直到EOF并将其附加到缓冲区，根据需要增长缓冲区。返回值n是读取的字节数。除了在读取期间遇到的io.EOF之外的任何错误也会被返回。如果缓冲区变得太大，ReadFrom将panic ErrTooLarge
	_, err = buf.ReadFrom(r)
	return buf.Bytes(), err
}


```

# ReadDir 函数
### 读取目录并返回排好序的文件和子目录名
```go
// ReadDir读取由dirname命名的目录，并返回按filename排序的目录条目列表。
func ReadDir(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname) //调用open方法
	if err != nil {
		return nil, err
    }
    // Readdir读取与file关联的目录的内容，并返回最多n个FileInfo值的片段，如Lstat将按目录顺序返回。对同一文件的后续调用将产生更多的FileInfos。
    // 如果n <= 0，则Readdir在单个片中返回目录中的所有FileInfo。在这种情况下，如果Readdir成功（一直读到目录的末尾），它将返回切片和nil错误。如果在目录结束之前遇到错误，Readdir将返回FileInfo读取，直到该点和非零错误。
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
    }
    //排序
	sort.Slice(list, func(i, j int) bool { return list[i].Name() < list[j].Name() })
	return list, nil
}

```

### 给定目录，打印出该目录下的所有文件
```go

func listAll(path string, curHier int){
    fileInfos, err := ioutil.ReadDir(path)
    if err != nil{fmt.Println(err); return}

    for _, info := range fileInfos{
        if info.IsDir(){
            // 根据指定的深度打印目录
            for tmpHier := curHier; tmpHier > 0; tmpHier--{
                fmt.Printf("|\t")
            }
            fmt.Println(info.Name(),"\\")
            listAll(path + "\\" + info.Name(),curHier + 1)
        }else{
            for tmpHier := curHier; tmpHier > 0; tmpHier--{
                fmt.Printf("|\t")
            }
            fmt.Println(info.Name())
        }
    }
}

```

# ReadFile 和 WriteFile 函数

### ReadFile 读取文件整个内容
```go
// ReadFile读取由filename命名的文件并返回内容
// A successful call returns err == nil, not err == EOF. Because ReadFile
// 成功的调用返回err == nil，而不是err == EOF。因为ReadFile读取整个文件，所以它不会将Read中的EOF视为要报告的错误
func ReadFile(filename string) ([]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// It's a good but not certain bet that FileInfo will tell us exactly how much to
	// read, so let's try it but be prepared for the answer to be wrong.
	var n int64 = bytes.MinRead

	if fi, err := f.Stat(); err == nil {
		// 作为readAll的初始容量，在Size为零的情况下使用Size+额外的一点空间，并且在Read填充缓冲区之后避免另一次分配。 调用readAll读入其分配的内部缓冲区。如果尺寸错误，要么在最后浪费一些空间，要么根据需要重新分配，但在绝大多数情况下，刚刚好
		if size := fi.Size() + bytes.MinRead; size > n {
			n = size
		}
	}
	return readAll(f, n)
}
```
### WriteFile 写入文件内容
```go
// WriteFile将数据写入由filename命名的文件
// 如果该文件不存在，则WriteFile使用权限perm创建它;否则WriteFile会在写入之前截断它。
// 对于perm参数，一般可以指定为：0666
func WriteFile(filename string, data []byte, perm os.FileMode) error {
    f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
    
	if err != nil {
		return err
	}
    n, err := f.Write(data)
    
	if err == nil && n < len(data) {
        // rrShortWrite意味着写入接受的字节数少于请求的字节数但未能返回显式错误。
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}
```

# TempDir 和 TempFile 函数
### 创建临时文件和临时目录
```go
// TempFile在目录dir中创建一个新的临时文件，打开文件进行读写，并返回生成的* os.File。
// 文件名是通过获取模式并在末尾添加随机字符串生成的。如果pattern包含“*”，则随机字符串将替换最后一个“*”。如果dir是空字符串，则TempFile使用临时文件的默认目录（请参阅os.TempDir）。同时调用TempFile的多个程序将不会选择相同的文件。调用者可以使用f.Name（）来查找文件的路径名。调用者负责在不再需要时删除该文件
func TempFile(dir, pattern string) (f *os.File, err error) {
	if dir == "" {
        // TempDir返回用于临时文件的默认目录。
		dir = os.TempDir()
	}

	var prefix, suffix string
	if pos := strings.LastIndex(pattern, "*"); pos != -1 {
		prefix, suffix = pattern[:pos], pattern[pos+1:]
	} else {
		prefix = pattern
	}

	nconflict := 0
	for i := 0; i < 10000; i++ {
		name := filepath.Join(dir, prefix+nextRandom()+suffix)
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		break
	}
	return
}

// TempDir在目录dir中创建一个新的临时目录，其名称以prefix开头，并返回新目录的路径。如果dir是空字符串，TempDir将使用临时文件的默认目录（请参阅os.TempDir）。同时调用TempDir的多个程序将不会选择相同的目录。调用者有责任在不再需要时删除目录
func TempDir(dir, prefix string) (name string, err error) {
	if dir == "" {
		dir = os.TempDir()
	}

	nconflict := 0
	for i := 0; i < 10000; i++ {
		try := filepath.Join(dir, prefix+nextRandom())
		err = os.Mkdir(try, 0700)
		if os.IsExist(err) {
			if nconflict++; nconflict > 10 {
				randmu.Lock()
				rand = reseed()
				randmu.Unlock()
			}
			continue
		}
		if os.IsNotExist(err) {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return "", err
			}
		}
		if err == nil {
			name = try
		}
		break
	}
	return
}

```