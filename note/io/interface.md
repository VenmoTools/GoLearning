
# Reader接口

```go
// Reader是包装基本Read方法的接口。
// Read 将 len(p) 个字节读取到 p 中。它返回读取的字节数 n（0 <= n <= len(p)） 以及任何遇到的错误。即使 Read 返回的 n < len(p)，它也会在调用过程中占用 len(p) 个字节作为暂存空间。若可读取的数据不到 len(p) 个字节，Read 会返回可用数据，而不是等待更多数据。
// 当 Read 在成功读取 n > 0 个字节后遇到一个错误或 EOF (end-of-file)，它会返回读取的字节数。它可能会同时在本次的调用中返回一个non-nil错误,或在下一次的调用中返回这个错误（且 n 为 0）。 一般情况下, Reader会返回一个非0字节数n, 若 n = len(p) 个字节从输入源的结尾处由 Read 返回，Read可能返回 err == EOF 或者 err == nil。并且之后的 Read() 都应该返回 (n:0, err:EOF)。
// 调用者在考虑错误之前应当首先处理返回的数据。这样做可以正确地处理在读取一些字节后产生的 I/O 错误，同时允许EOF的出现。
// 除非len（p）== 0，否则不允许实现Read的零个字节和nil错误。
// 调用者应该将0和nil的返回视为表示没有发生任何事情;特别是它并不表示EOF。
type Reader interface {
	Read(p []byte) (n int, err error)
}
```


# Writer接口

```go
// Writer是包装基本Write方法的接口。
// Write 将 len(p) 个字节从 p 中写入到基本数据流中。它返回从 p 中被写入的字节数 n（0 <= n <= len(p)）以及任何遇到的引起写入提前停止的错误。若 Write 返回的 n < len(p)，它就必须返回一个 非nil 的错误。
type Writer interface {
	Write(p []byte) (n int, err error)
}
```

## 实现了 io.Reader 接口或 io.Writer 接口的类型

+ os.File 同时实现了 io.Reader 和 io.Writer
+ strings.Reader 实现了 io.Reader
+ bufio.Reader/Writer 分别实现了 io.Reader 和 io.Writer
+ bytes.Buffer 同时实现了 io.Reader 和 io.Writer
+ bytes.Reader 实现了 io.Reader
+ compress/gzip.Reader/Writer 分别实现了 io.Reader 和 io.Writer
+ crypto/cipher.StreamReader/StreamWriter 分别实现了 io.Reader 和 io.Writer
+ crypto/tls.Conn 同时实现了 io.Reader 和 io.Writer
+ encoding/csv.Reader/Writer 分别实现了 io.Reader 和 io.Writer
+ mime/multipart.Part 实现了 io.Reader
+ net/conn 分别实现了 io.Reader 和 io.Writer(Conn接口定义了Read/Write)
+ 实现了Reader的LimitedReader、PipeReader、SectionReader类型
+ 实现了Writer的PipeWriter类型


# Closer接口

```go
// Closer是包装基本Close方法的接口。
// 第一次调用后Close的行为未定义。具体实现可以根据他们自己的行为
type Closer interface {
    Close() error
}

```

# Seeker 接口

```go
// Seeker是包装基本Seek方法的接口
// Seek 设置下一次 Read 或 Write 的偏移量为 offset，它的解释取决于 whence： 0 表示相对于文件的起始处，1 表示相对于当前的偏移，而 2 表示相对于其结尾处。 Seek 返回新的偏移量和一个错误，如果有的话
// Seek文件开始之前偏移会返回一个error。
// Seek任何正偏移是合法的，但后续I/O操作对底层对象的行为是依赖于实现的。
type Seeker interface {
    Seek(offset int64, whence int) (ret int64, err error)
}

// whence 的值，在 io 包中定义了相应的常量，应该使用这些常量
const (
  SeekStart   = 0 // Seek文件的开始处
  SeekCurrent = 1 // Seek相对于当前的偏移量
  SeekEnd     = 2 // Seek文件的结束
)


```
# ReadFrom和WriteTo

```go
// ReaderFrom是包装ReadFrom方法的接口
//
// ReadFrom 从 r 中读取数据，直到 EOF 或发生错误。其返回值 n 为读取的字节数。除 io.EOF 之外，在读取过程中遇到的任何错误也将被返回。
// 如果 ReaderFrom 可用，Copy 函数就会使用它
// ReadFrom 方法不会返回 err == EOF。
type ReaderFrom interface {
	ReadFrom(r Reader) (n int64, err error)
}

// WriterTo是包装 WriteTo方法的接口
//
// WriteTo 将数据写入 w 中，直到没有数据可写或发生错误。其返回值 n 为写入的字节数。 在写入过程中遇到的任何错误也将被返回。
// 如果 WriterTo 可用，Copy 函数就会使用它。

type WriterTo interface {
	WriteTo(w Writer) (n int64, err error)
}

```
# ReaderAt 和 WriterAt 接口

```go
// ReadAt 从基本输入源的偏移量 off 处开始，将 len(p) 个字节读取到 p 中。它返回读取的字节数 n（0 <= n <= len(p)）以及任何遇到的错误。
// 当 ReadAt 返回的 n < len(p) 时，它就会返回一个 非nil 的错误来解释 为什么没有返回更多的字节。在这一点上，ReadAt 比 Read 更严格。
// 即使 ReadAt 返回的 n < len(p)，它也会在调用过程中使用 p 的全部作为暂存空间。若可读取的数据不到 len(p) 字节，ReadAt 就会阻塞,直到所有数据都可用或一个错误发生。 在这一点上 ReadAt 不同于 Read。
// 若 n = len(p) 个字节从输入源的结尾处由 ReadAt 返回，Read可能返回 err == EOF 或者 err == nil
// 若 ReadAt 携带一个偏移量从输入源读取，ReadAt 应当既不影响偏移量也不被它所影响。
// 可对相同的输入源并行执行 ReadAt 调用。
// ReaderAt 接口使得可以从指定偏移量处开始读取数据
type ReaderAt interface {
    ReadAt(p []byte, off int64) (n int, err error)
}

// WriteAt 从 p 中将 len(p) 个字节写入到偏移量 off 处的基本数据流中。它返回从 p 中被写入的字节数 n（0 <= n <= len(p)）以及任何遇到的引起写入提前停止的错误。若 WriteAt 返回的 n < len(p)，它就必须返回一个 非nil 的错误。
// 若 WriteAt 携带一个偏移量写入到目标中，WriteAt 应当既不影响偏移量也不被它所影响。
// 若被写区域没有重叠，可对相同的目标并行执行 WriteAt 调用。
// p数据将会在offset的偏移处写入
type WriterAt interface {
    WriteAt(p []byte, off int64) (n int, err error)
}

```

# 其他接口

其他接口为以上接口的组合
```go
// ReadWriter is the interface that groups the basic Read and Write methods.
type ReadWriter interface {
	Reader
	Writer
}

// ReadCloser is the interface that groups the basic Read and Close methods.
type ReadCloser interface {
	Reader
	Closer
}

// WriteCloser is the interface that groups the basic Write and Close methods.
type WriteCloser interface {
	Writer
	Closer
}

// ReadWriteCloser is the interface that groups the basic Read, Write and Close methods.
type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}

// ReadSeeker is the interface that groups the basic Read and Seek methods.
type ReadSeeker interface {
	Reader
	Seeker
}

// WriteSeeker is the interface that groups the basic Write and Seek methods.
type WriteSeeker interface {
	Writer
	Seeker
}

// ReadWriteSeeker is the interface that groups the basic Read, Write and Seek methods.
type ReadWriteSeeker interface {
	Reader
	Writer
	Seeker
}

```