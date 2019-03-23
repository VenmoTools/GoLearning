package codeDected

import (
	"bufio"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"helper/logs"
	"io"
	"io/ioutil"
	"net/http"
)

func determineEncoding(r *bufio.Reader) encoding.Encoding {
	bytes, err := r.Peek(1024)
	if err != nil {
		logs.Error("Fetcher error ", err)
		return unicode.UTF8
	}
	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e
}

func ParserFromReader(r io.Reader) *transform.Reader {
	reader := bufio.NewReader(r)
	encode := determineEncoding(reader)
	return transform.NewReader(reader, encode.NewEncoder())
}

func ParserFromResponse(r *http.Response) ([]byte,error) {
	data := ParserFromReader(r.Body)
	return ioutil.ReadAll(data)
}
