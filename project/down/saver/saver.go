package saver

import (
	"fmt"
	"os"
)

func Save() chan interface{} {
	out := make(chan interface{})
	go func() {
		count := 0
		f, _ := os.OpenFile("test.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
		defer f.Close()

		for {
			item := <-out
			f.WriteString(fmt.Sprintf("第 %d 条数据，内容: %s \n", count, item))
			count++
		}
	}()
	return out
}
