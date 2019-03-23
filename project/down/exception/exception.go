package exception

import (
	"context"
	"errors"
	"helper/logs"
	"sync/atomic"
)

type ErrorType string

type DownloadError interface {
	Init(buffer uint32) error
	SendError(err error)
	Start()
	Cancel() bool
	Close()
}

type donwloadError struct {
	name    string
	errChan chan error
	total   uint32
	ctx     context.Context
	ctxFunc context.CancelFunc
}

func (d *donwloadError) Close() {
	close(d.errChan)
}
func (d *donwloadError) TotalHandler() uint32 {
	return atomic.LoadUint32(&d.total)
}

func (d *donwloadError) Cancel() bool {
	if d.ctx == nil || d.ctxFunc == nil {
		return false
	}
	d.ctxFunc()
	return true
}

func (d *donwloadError) Init(buffer uint32) error {
	if buffer == 0 {
		buffer = 1
	}
	d.errChan = make(chan error, buffer)
	if d.errChan == nil {
		return errors.New("error chan create failed")
	}

	d.ctx, d.ctxFunc = context.WithCancel(context.Background())
	return nil
}

func (d *donwloadError) SendError(err error) {
	if d.errChan == nil {
		return
	}
	if err != nil {
		d.errChan <- err
	}
}

func (d *donwloadError) Start() {

	go func(errChan chan error) {
		defer func() {
			logs.Info("Error Handler exited")
		}()
		logs.Info("Start %s Error Handler", d.name)

		for {
			select {
			case err := <-errChan:
				logs.Error("%s have error %s", d.name, err)
				atomic.AddUint32(&d.total, 1)
			case <-d.ctx.Done():
				logs.Info("Error Handler exiting...")
				return
			}
		}
	}(d.errChan)
}

func NewError(buffer uint32, name string) *donwloadError {
	d := &donwloadError{}
	err := d.Init(buffer)
	d.name = name
	logs.Info("init %s completed", d.name)

	if err != nil {
		panic(err)
	}
	defer func() {
		if err := recover(); err != nil {
			logs.Error("A fatal error when init error")
		}
	}()
	return d
}
