package exceptions

import (
	"bytes"
	"fmt"
	"strings"
)

type ErrorType string

//CrawlerError
type CrawlerError interface {
	Type() ErrorType
	Error() string
}

const (
	ERROR_TYPE_DOWNLOADER ErrorType = "downloader error"
	ERROR_TYPE_ANALYZER   ErrorType = "analyzer error"
	ERROR_TYPE_PIPLINE    ErrorType = "pipline error"
	ERRPR_TYPE_SCHEDULER  ErrorType = "scheduler error"
)

type CrawlerErrors struct {
	errType    ErrorType
	msg        string
	fullErrMsg string
}


func (c *CrawlerErrors) Type() ErrorType {
	return c.errType
}

func (c *CrawlerErrors) Error() string {
	if c.fullErrMsg == "" {
		c.genFullMsg()
	}
	return c.fullErrMsg
}

func (c *CrawlerErrors) genFullMsg() {
	var buffer bytes.Buffer
	buffer.WriteString("crawler error:")
	if c.errType != "" {
		buffer.WriteString(string(c.errType))
		buffer.WriteString(":")
	}
	buffer.WriteString(c.msg)
	c.fullErrMsg = fmt.Sprintf("%s", buffer.String())
}

func NewCrawlerErrors(errType ErrorType, msg string) CrawlerError {
	return &CrawlerErrors{errType: errType, msg: strings.TrimSpace(msg)}
}


type IllegalParameterError struct {
	msg string
}

func NewIllegalParameterError(errMsg string) IllegalParameterError {
	return IllegalParameterError{
		msg: fmt.Sprintf("illegal parameter: %s",
			strings.TrimSpace(errMsg)),
	}
}

func (ipe IllegalParameterError) Error() string {
	return ipe.msg
}