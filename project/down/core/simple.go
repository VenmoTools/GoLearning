package core

type SimpleScheduler struct {
	workerChan chan Request
}

func (s *SimpleScheduler) WorkerChan() chan Request {
	return s.workerChan
}

func (s *SimpleScheduler) WorkerReady(chan Request) {
}

func (s *SimpleScheduler) Run() {
	s.workerChan = make(chan Request)
}

func (s *SimpleScheduler) Submit(r Request) {
	//调度器将待完成的任务送到workerChan队列
	// s.workerChan <- r 这行代码会造成死锁，不能保证每提交都会有相应的goroutine接收

	// 开启另一个goroutine来提交任务
	go func() {
		s.workerChan <- r
	}()
}
