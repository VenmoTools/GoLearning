package core

type Notify interface {
	WorkReady(chan Request)
}

type Scheduler interface {
	Notify
	Start()
	Submit(request Request)
	WorkChan() chan Request
}
