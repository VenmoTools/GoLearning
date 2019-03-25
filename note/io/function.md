# 基本函数
## Copy
```go
// Copy 将 src 复制到 dst，直到在 src 上到达 EOF 或发生错误。它返回复制的字节数，如果有错误的话，还会返回在复制时遇到的第一个错误。
// 成功的 Copy 返回 err == nil，而非 err == EOF。由于 Copy 被定义为从 src 读取直到 EOF 为止，因此它不会将来自 Read 的 EOF 当做错误来报告。
// 若 dst 实现了 ReaderFrom 接口，其复制操作可通过调用 dst.ReadFrom(src) 实现。此外，若 src 实现了 WriterTo 接口，其复制操作可通过调用 src.WriteTo(dst) 实现。
func Copy(dst Writer, src Reader) (written int64, err error) {
	return copyBuffer(dst, src, nil)
}
```
## CopyN 函数

```go
// CopyN 将 n 个字节(或到一个error)从 src 复制到 dst。 它返回复制的字节数以及在复制时遇到的最早的错误。当且仅当err == nil时,written == n 。
// 若 dst 实现了 ReaderFrom 接口，复制操作也就会使用它来实现。
func CopyN(dst Writer, src Reader, n int64) (written int64, err error) {
	written, err = Copy(dst, LimitReader(src, n))
	if written == n {
		return n, nil
	}
	if written < n && err == nil {
		// src stopped early; must have been EOF.
		err = EOF
	}
	return
}
```
## 类型
### LimitedReader 类型
```go
// LimitedReader从R读取但限制返回的数据量仅为N个字节。每次调用Read都会更新 N 来反应新的剩余数量。当N <= 0或基础R返回EOF时，返回EOF。
// 最多只能返回 N 字节数据。
type LimitedReader struct {
    R Reader // underlying reader，最终的读取操作通过 R.Read 完成
    N int64  // max bytes remaining
}
// LimitReader 函数的实现其实就是调用 LimitedReader：
func LimitReader(r Reader, n int64) Reader { return &LimitedReader{r, n} }
```

## 函数实现
```go
// CopyBuffer与Copy相同，只是它通过提供的缓冲区（如果需要）缓冲，而不是分配临时缓冲区。如果buf为零，则分配一个;否则，如果它的长度为零，则CopyBuffer会panic
func CopyBuffer(dst Writer, src Reader, buf []byte) (written int64, err error) {
	if buf != nil && len(buf) == 0 {
		panic("empty buffer in io.CopyBuffer")
	}
	return copyBuffer(dst, src, buf)
}

// copyBuffer是Copy和CopyBuffer的实际实现。如果buf为nil，则分配一个。
func copyBuffer(dst Writer, src Reader, buf []byte) (written int64, err error) {
	// 如果reader具有WriteTo方法，使用它来执行复制。避免分配和复制
	if wt, ok := src.(WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// 同样，如果编写器具有ReadFrom方法，则使用它来执行复制
	if rt, ok := dst.(ReaderFrom); ok {
		return rt.ReadFrom(src)
    }
    // 如果没有之指定缓冲区则创建一个
	if buf == nil {
        size := 32 * 1024
        // 尝试从LimitedReader中读取字节长度用于缓冲区大小
		if l, ok := src.(*LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}
	for {
        // 从src读取字节放进缓冲区
		nr, er := src.Read(buf)
		if nr > 0 {
            // 每次写入读取到的字节
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

//ReadAtLeast 将 r 读取到 buf 中，直到读了最少 min 个字节为止。它返回复制的字节数，如果读取的字节较少，还会返回一个错误。若没有读取到字节，错误就只是 EOF。如果一个 EOF 发生在读取了少于 min 个字节之后，ReadAtLeast 就会返回 ErrUnexpectedEOF。若 min 大于 buf 的长度，ReadAtLeast 就会返回 ErrShortBuffer。对于返回值，当且仅当 err == nil 时，才有 n >= min
func ReadAtLeast(r Reader, buf []byte, min int) (n int, err error) {
	if len(buf) < min {
		return 0, ErrShortBuffer
	}
	for n < min && err == nil {
		var nn int
		nn, err = r.Read(buf[n:])
		n += nn
	}
	if n >= min {
		err = nil
	} else if n > 0 && err == EOF {
		err = ErrUnexpectedEOF
	}
	return
}

//ReadFull 精确地从 r 中将 len(buf) 个字节读取到 buf 中。它返回复制的字节数，如果读取的字节较少，还会返回一个错误。若没有读取到字节，错误就只是 EOF。如果一个 EOF 发生在读取了一些但不是所有的字节后，ReadFull 就会返回 ErrUnexpectedEOF。对于返回值，当且仅当 err == nil 时，才有 n == len(buf)。
func ReadFull(r Reader, buf []byte) (n int, err error) {
	return ReadAtLeast(r, buf, len(buf))
}

//WriteString 将s的内容写入w中，当 w 实现了 WriteString 方法时，会直接调用该方法，否则执行 w.Write([]byte(s))。
func WriteString(w Writer, s string) (n int, err error) {
	if sw, ok := w.(stringWriter); ok {
		return sw.WriteString(s)
	}
	return w.Write([]byte(s))
}

//MultiReader返回一个Reader，它是所提供输入阅读器的逻辑串联。他们按顺序读取。一旦所有输入都返回EOF，Read将返回EOF。如果任何读者返回非零，非EOF错误，Read将返回该错误。
func MultiReader(readers ...Reader) Reader {
	r := make([]Reader, len(readers))
	copy(r, readers)
	return &multiReader{r}
}

type multiReader struct {
	readers []Reader
}

func (mr *multiReader) Read(p []byte) (n int, err error) {
	for len(mr.readers) > 0 {
		// Optimization to flatten nested multiReaders (Issue 13558).
		if len(mr.readers) == 1 {
			if r, ok := mr.readers[0].(*multiReader); ok {
				mr.readers = r.readers
				continue
			}
		}
		n, err = mr.readers[0].Read(p)
		if err == EOF {
			// Use eofReader instead of nil to avoid nil panic
			// after performing flatten (Issue 18232).
			mr.readers[0] = eofReader{} // permit earlier GC
			mr.readers = mr.readers[1:]
		}
		if n > 0 || err != EOF {
			if err == EOF && len(mr.readers) > 0 {
				// Don't return EOF yet. More readers remain.
				err = nil
			}
			return
		}
	}
	return 0, EOF
}

// MultiWriter创建一个Writer，复制其对所有提供的Writer的写入，类似于Unix tee（1）命令。
// 将调用列表中所有的Writer，一次一个。如果列表中的编写器返回错误，则整个写入操作将停止并返回错误;便不会继续
func MultiWriter(writers ...Writer) Writer {
	allWriters := make([]Writer, 0, len(writers))
	for _, w := range writers {
		if mw, ok := w.(*multiWriter); ok {
			allWriters = append(allWriters, mw.writers...)
		} else {
			allWriters = append(allWriters, w)
		}
	}
	return &multiWriter{allWriters}
}

type multiWriter struct {
	writers []Writer
}

func (t *multiWriter) Write(p []byte) (n int, err error) {
    // 往所有的Writer中写入p
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = ErrShortWrite
			return
		}
	}
	return len(p), nil
}


// TeeReader 返回一个 Reader，它将从 r 中读到的数据写入 w 中。所有经由它处理的从 r 的读取都匹配于对应的对 w 的写入。它没有内部缓存，即写入必须在读取完成前完成。任何在写入时遇到的错误都将作为读取错误返回。
// 我们通过 Reader 读取内容后，会自动写入到 Writer 中去
func TeeReader(r Reader, w Writer) Reader {
	return &teeReader{r, w}
}


type teeReader struct {
	r Reader
	w Writer
}

func (t *teeReader) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		if n, err := t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return
}

```

## 使用

### Copy

```go
import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main(){
    length, err := io.Copy(os.Stdout, strings.NewReader("hello"))
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("\n 读取长度为 ", length)

}
```
### CopyN

```go
import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	length, err := io.CopyN(os.Stdout, strings.NewReader("含有中文，hello"), 12)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("\n 读取长度为 ", length)

}

```

### MultiReader

```go
import (
	"fmt"
	"io"
)
func main() {
	readers := []io.Reader{
		strings.NewReader("hello"),
		bytes.NewBuffer([]byte("world")),
	}

	reader := io.MultiReader(readers...)
	buf := make([]byte, 10)
	for n, err := reader.Read(buf); err != io.EOF; n, err = reader.Read(buf) {
		fmt.Println(string(buf[:n]))
	}

}
```

### MultiWriter
```go
import (
	"fmt"
	"io"
	"os"
)
func main(){
    file, err := os.Create("text.txt")
    if err != nil {
        panic(err)
    }
    defer file.Close()
    writers := []io.Writer{
        file,
        os.Stdout,
    }
    writer := io.MultiWriter(writers...)
    writer.Write([]byte("Hello World"))
}

```

### TeeWriter

```go
import (
	"io"
	"os"
	"strings"
)
func main(){
    reader := io.TeeReader(strings.NewReader("Go语言中文网"), os.Stdout)
    reader.Read(make([]byte, 20))
}

```