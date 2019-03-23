package core

import "helper/logs"

type QueueScheduler struct {
	requestChan chan Request
	workChan    chan chan Request
	Active      bool
}

func (q *QueueScheduler) WorkReady(r chan Request) {
	q.workChan <- r
}

func (q *QueueScheduler) Start() {
	q.workChan = make(chan chan Request,10)
	q.requestChan = make(chan Request,10)

	go func() {
		var (
			requestQueue []Request
			workQueue    []chan Request
		)

		for {
			var (
				activeRequest Request
				activeWork    chan Request
			)

			if len(requestQueue) > 0 && len(workQueue) > 0 {
				activeRequest = requestQueue[0]
				activeWork = workQueue[0]
			}

			select {
			case req := <-q.requestChan:
				requestQueue = append(requestQueue, req)
			case work := <-q.workChan:
				workQueue = append(workQueue, work)
			case activeWork <- activeRequest:
				requestQueue = requestQueue[1:]
				workQueue = workQueue[1:]
			}

		}

	}()
}

func (q *QueueScheduler) Submit(request Request) {
	if request.Req == nil{
		logs.Info("request is nil")
		return
	}
	q.requestChan <- request
}

func (q *QueueScheduler) WorkChan() chan Request {
	return make(chan Request)
}
