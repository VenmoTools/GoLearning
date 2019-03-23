package schedluer

import (
	"spider/config"
	"testing"
)



func TestNewDefaultScheduler(t *testing.T) {
	conf := config.NewDefaultConfig([]string{""},nil,nil)

	sche := NewDefaultScheduler(conf)
	if sche == nil {
		t.Fatal("Couldn't create scheduler!")
	}
}

func TestScheduler_Init(t *testing.T) {

}


