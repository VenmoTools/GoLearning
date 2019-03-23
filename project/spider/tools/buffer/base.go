package buffer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

var ErrClosedBuffer = errors.New("closed buffer")
var ErrClosedPool = errors.New("closed pool")

const (
	OPEN  = 0
	CLOSE = 1
)

type MultipleReader interface {
	Reader() io.ReadCloser
}

type multipleReader struct {
	data []byte
}

func NewMultipleReader(reader io.Reader) (multiple MultipleReader, err error) {
	var data []byte
	if reader != nil {
		data, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("multiple reader:couldn`t create a new one : %s", err)
		}
	} else {
		data = []byte{}
	}
	return &multipleReader{
		data: data,
	}, nil

}

func (m *multipleReader) Reader() io.ReadCloser {
	return ioutil.NopCloser(bytes.NewReader(m.data))
}
