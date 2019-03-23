package module

import (
	"fmt"
	"testing"
)

var illegalMIDs = []MID{
	MID("D"),
	MID("DZ"),
	MID("D1|"),
	MID("D1|127.0.0.1:-1"),
	MID("D1|127.0.0.1:"),
	MID("D1|127.0.0.1"),
	MID("D1|127.0.0."),
	MID("D1|127"),
	MID("D1|127.0.0.0.1:8080"),
	MID("DZ|127.0.0.1:8080"),
	MID("A"),
	MID("AZ"),
	MID("A1|"),
	MID("A1|127.0.0.1:-1"),
	MID("A1|127.0.0.1:"),
	MID("A1|127.0.0.1"),
	MID("A1|127.0.0."),
	MID("A1|127"),
	MID("A1|127.0.0.0.1:8080"),
	MID("AZ|127.0.0.1:8080"),
	MID("P"),
	MID("PZ"),
	MID("P1|"),
	MID("P1|127.0.0.1:-1"),
	MID("P1|127.0.0.1:"),
	MID("P1|127.0.0.1"),
	MID("P1|127.0.0."),
	MID("P1|127"),
	MID("P1|127.0.0.0.1:8080"),
	MID("PZ|127.0.0.1:8080"),
	MID("M1|127.0.0.1:8080"),
}


func TestLegalMid(t *testing.T) {
	for _,mid := range illegalMIDs{
		if !LegalMid(mid) {
			t.Log(fmt.Sprintf("test %s,succeed",mid))
		}
	}
}

func TestSplitMid(t *testing.T) {
	for _,mid := range illegalMIDs{
		if res,err := SplitMid(mid);err ==nil {
			t.Log(fmt.Sprintf("test %s succeed",res))
		}else{
			t.Log(err)
		}
	}
}