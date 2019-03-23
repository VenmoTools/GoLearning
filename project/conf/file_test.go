package conf

import (
	"fmt"
	"testing"
)

func TestReadFile(t *testing.T) {
	f := FileReader{}
	err := f.Open("/Users/venmosnake/Documents/go/test.ini")
	if err != nil {
		t.Error(err)
	}

	err = f.Parser()
	if err != nil {
		t.Error(err)
	}

	v,err := f.Get("tet::username")
	if err != nil {
		t.Error(err)
	}

	fmt.Println(v.GetAsString())

}
