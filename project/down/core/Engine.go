package core

import (
	"down/saver"
	"time"
)

/*******************
使用的对象：
	worker ： 工作者用于执行工作
	work ： 分配给工作者的工作
	scheduler ： 调度器接口，实现调度方法
	core: 总引擎，启动工作，处理工作
*/

type Engine struct {
	Scheduler   Scheduler        //调度器
	WorkChanNum int              // 最大channel数量
	ItemSave    chan interface{} // 解析出来的item
	resp        chan Response    //请求结果
}

func NewEngine() Engine {
	eng := Engine{
		Scheduler:   &QueueScheduler{},
		WorkChanNum: 10,
		ItemSave:    saver.Save(),
		resp:        make(chan Response,10),
	}
	return eng
}

func (e *Engine) Run(request Request) {

	e.Scheduler.Start()

	for i:=0;i<e.WorkChanNum；i++{
		e.createWorker(e.Scheduler.WorkChan(), e.Scheduler)
	}

	e.Scheduler.Submit(request)

	e.Handler()
}

func (e *Engine) Handler() {
	for {
		res := <-e.resp

		for _, item := range res.GetItemQueue() {
			if item != nil {
				e.save(item)
			}
		}
		time.Sleep(time.Second)

		for _, request := range res.GetRequestQueue() {
			e.Scheduler.Submit(request)
		}
		//if res.SizeOfRequestQueue() == 0 && res.SizeOfItemQueue() == 0 {
		//	return
		//}
	}
}

func (e *Engine) save(item interface{}) {
	go func() {
		e.ItemSave <- item
	}()
}

func (e *Engine) createWorker(work chan Request, notify Notify) {
	wo := NewParserWork()
	go func(work chan Request) {
		for {
			notify.WorkReady(work)
			request := <-work
			Result, err := Worker(request, wo)
			if err != nil {
				continue
			}
			e.resp <- Result
		}
	}(work)
}
