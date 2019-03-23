package project

import (
	"fmt"
	"testing"
)

func TestParserCompany(t *testing.T) {

	for i := 2; i < 10; i++ {
		fmt.Println(GetUrl(i))
	}

}
