package conf

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	_ "unsafe"
)

var BOL = errors.New("black line")

type FileReader struct {
	fileName    string
	reader      *bufio.Reader
	lineNo      uint64
	currentLine string
	data        map[string]Value
}

func (f *FileReader) Open(fileName string) (err error) {
	data, err := ioutil.ReadFile(fileName)
	f.data = make(map[string]Value)
	if err != nil {
		return
	}
	f.reader = bufio.NewReader(bytes.NewBuffer(data))
	return nil
}

func (f *FileReader) readLine() error {

	data, _, err := f.reader.ReadLine()
	if err != nil {
		return err
	}
	f.currentLine = string(data)
	f.lineNo++
	if f.currentLine == "" || len(f.currentLine) <= 0 || strings.HasPrefix(f.currentLine, "#") {
		return BOL
	}
	return nil
}

func (f *FileReader) Parser() error {
	if f.reader == nil {
		return errors.New("file not read")
	}
	var lastSpace string
	var value *Value

	for {
		err := f.readLine()
		if err != nil {
			if err == io.EOF {
				f.currentLine = ""
				return nil
			}
			if err == BOL {
				continue
			}
			fmt.Println("error at line:", f.lineNo)
		}

		if f.currentLine[:1] == "[" && f.currentLine[len(f.currentLine)-1] == ']' {
			value = NewValue()
			lastSpace = getSpace(f.currentLine)
		} else {
			k, v, err := checkAndSplit(f.currentLine)
			if err != nil {
				fmt.Println(err)
			}
			value.Put(k, v)
			f.data[lastSpace] = *value
		}
	}
}

func (f *FileReader) Get(str string) *conv {
	data := strings.Split(str, "::")
	if len(data) == 0 || len(data) > 2 {
		panic(errors.New("syntax error"))
	}
	value, ok := f.data[data[0]]
	if !ok {
		panic(fmt.Sprintf("no such key %s", str))
	}
	if d, ok := value.Get(data[1]); ok {
		return d
	}
	panic(fmt.Sprintf("no such key %s", str))
}

func checkAndSplit(line string) (k, v string, err error) {
	if line == "" {
		return "", "", errors.New("black line")
	}
	data := strings.Split(strings.TrimSpace(line), "=")
	if len(data) != 2 {
		return "", "", errors.New("syntax error '=' in one line")
	}
	k = strings.TrimSpace(data[0])
	v = strings.TrimSpace(data[1])
	return
}

func getSpace(data string) string {
	return strings.TrimSpace(data[1 : len(data)-1])
}
