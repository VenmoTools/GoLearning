package exception

import (
	"errors"
	"testing"
	"time"
)

func TestNewError(t *testing.T) {
	errs := NewError(10)
	if errs == nil {
		t.Fatal("the Error Handler is nil")
	}
	if errs.errChan == nil {
		t.Fatal("the errChan is nil")
	}
}

func TestDonwloadError_Init(t *testing.T) {
	d := donwloadError{}
	err := d.Init(8)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDonwloadError_SendError(t *testing.T) {
	errs := NewError(10)
	errs.Start()
	errs.Notify()

	errs.SendError(errors.New("exception1"))
	errs.SendError(errors.New("exception1"))
	errs.SendError(errors.New("exception1"))

	time.Sleep(time.Second * 5)
	errs.Cancel()
}

func TestDonwloadError_Start(t *testing.T) {
	errs := NewError(10)
	errs.Start()
	errs.Notify()
	errs.Cancel()

}
