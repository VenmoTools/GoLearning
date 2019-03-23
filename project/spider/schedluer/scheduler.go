package schedluer

import (
	"context"
	"errors"
	"fmt"
	"helper/logs"
	"log"
	"net/http"
	"spider/datastructs"
	"spider/module"
	analyzer2 "spider/module/analyzer"
	"spider/module/downloader"
	"spider/module/pipeline"
	"spider/tools/buffer"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Scheduler interface {
	/**
	@requestArgs:请求相关的参数
	@dataArgs:数据相关的参数
	@moduleArgs:组建相关参数
	*/
	Init(requestArgs RequestArgs, dataArgs DataArgs, moduleArgs ModuleArgs) error //用于初始化调度器，
	Start(firstReq *http.Request) error                                           //启动调度器执行爬去流程
	Stop() error                                                                  //停止调度器
	Status() Status                                                               //调度器状态
	ErrorChan() <-chan error                                                      //获得错误通道，调度器以及各个模块运行时出现的所有错误都会被发送到改通道
	Idle() bool                                                                   //判断所有处理模块都为空闲状态
	Summary() SchedSummary                                                        //获取摘要
}

type SchedSummary interface {
	Struct() Summary
	String() string
}

type scheduler struct {
	maxDepth          uint32             // 爬取最大的深度，首次为0
	acceptedDomianMap CurrentMap         // 可接受的域名
	register          module.Register    // 组件组册器
	reqBufferPool     buffer.Pool        // 请求缓冲池
	respBufferPool    buffer.Pool        // 响应缓冲池
	itemBufferPool    buffer.Pool        // 条目缓冲池
	errBufferPool     buffer.Pool        // 错误缓冲区
	urlMap            CurrentMap         // 已处理的url
	ctx               context.Context    // 上线文，用于感知调度器停止
	cancelFunc        context.CancelFunc // 取消函数
	status            Status             // 状态
	lock              sync.RWMutex       // 状态读写锁
	summary           SchedSummary       // 摘要信息
}

// 调度器初始化
func (s *scheduler) Init(requestArgs RequestArgs, dataArgs DataArgs, moduleArgs ModuleArgs) (err error) {
	logs.Info("[Scheduler] --> Check status for initialization")
	var oldStatus Status
	oldStatus, err = s.checkAndSetStatus(SCHED_STATUS_INITIALIZING)
	if err != nil {
		return
	}

	defer func() {
		s.lock.Lock()
		if err != nil {
			s.status = oldStatus
		} else {
			s.status = SCHED_STATUS_INITIALIZED
		}
		s.lock.Unlock()
	}()

	logs.Info("[Scheduler] --> Check request arguments")
	if err = requestArgs.Check(); err != nil {
		return
	}
	logs.Info("[Scheduler] --> Check data arguments")

	if err = dataArgs.Check(); err != nil {
		return
	}
	logs.Info("[Scheduler] --> Check module arguments")

	if err = moduleArgs.Check(); err != nil {
		return
	}

	logs.Info("[Scheduler] --> initialization Register")
	if s.register == nil {
		s.register = module.NewRegister()
	} else {
		s.register.Clear()
	}

	s.maxDepth = requestArgs.MaxDepth
	logs.Info("[Scheduler] --> max depth is %d", s.maxDepth)

	s.acceptedDomianMap = NewCurrentMap()
	for _, domain := range requestArgs.AcceptedDomains {
		s.acceptedDomianMap.Put(domain, struct{}{})
	}
	logs.Info("[Scheduler] --> accepted primary Domain %v ", s.acceptedDomianMap)

	s.urlMap = NewCurrentMap()
	logs.Info("[Scheduler] --> URL map: concurrency: %d", s.urlMap.length)

	s.initBufferPool(dataArgs)
	s.resetContext()
	s.summary = newSchedulerSummary(requestArgs, dataArgs, moduleArgs, s)
	logs.Info("[Scheduler] --> Register module")
	if err := s.registerModule(moduleArgs); err != nil {
		return err
	}
	logs.Info("[Scheduler] --> The number of Register modules %d", len(s.register.GetAll()))
	logs.Info("[Scheduler] --> Scheduler has been initialized.")
	return nil
}

func (s *scheduler) resetContext() {
	s.ctx, s.cancelFunc = context.WithCancel(context.Background())
	logs.Info("[Scheduler] --> context reset completed")
}

// canceled 用于判断调度器的上下文是否已被取消。
func (s *scheduler) canceled() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

func (s *scheduler) registerModule(args ModuleArgs) error {

	for _, ar := range args.Downloaders {
		logs.Info("[Register Request] --> Register module %v", ar)
		if ar == nil {
			continue
		}
		ok, err := s.register.Register(ar)
		if err != nil {
			logs.Error("[Register Request] --> Register module error %v", err)
			return err
		}
		if !ok {
			return genError(fmt.Sprintf("Couldn't downloads analyzer instance with MID %q!", ar.Id()))
		}
	}

	logs.Info("[Register] --> All downloads have been registered. (number: %d) ", len(args.Downloaders))

	for _, ar := range args.Analyzer {
		if ar == nil {
			continue
		}
		ok, err := s.register.Register(ar)
		if err != nil {
			logs.Error("[Register Analyzer] --> Register module error %v", err)
			return err
		}
		if !ok {
			return genError(fmt.Sprintf("Couldn't register analyzer instance with MID %q!", ar.Id()))
		}
	}

	logs.Info("[Register] --> All analyzer have been registered. (number: %d) ", len(args.Analyzer))

	for _, ar := range args.Pipline {
		if ar == nil {
			continue
		}
		ok, err := s.register.Register(ar)
		if err != nil {
			logs.Error("[Register PipeLine] --> Register module error %v", err)
			return err
		}
		if !ok {
			return genError(fmt.Sprintf("Couldn't register pipline instance with MID %q!", ar.Id()))
		}
	}

	logs.Info("[Register] --> All Pipline have been registered. (number: %d) ", len(args.Pipline))

	return nil
}

// 用于按照给定的参数初始化缓冲池。
// 如果某个缓冲池可用且未关闭，就先关闭该缓冲池。
func (s *scheduler) initBufferPool(args DataArgs) {
	var err error

	if s.reqBufferPool != nil && !s.reqBufferPool.Closed() {
		s.reqBufferPool.Close()
	}
	// 请求缓冲器
	s.reqBufferPool, err = buffer.NewPool(args.RequestBufferCap, args.RequestMaxBufferNumber, "Request")
	if err != nil {
		logs.Error("[Buffer] --> initialization request buffer error", err)
	}
	logs.Info("[Buffer] --> Request buffer pool: bufferCap: %d, maxBufferNumber: %d,  total:%d", s.reqBufferPool.BufferCap(), s.reqBufferPool.MaxBufferNumber(), s.reqBufferPool.Total())

	if s.respBufferPool != nil && !s.respBufferPool.Closed() {
		s.respBufferPool.Close()
	}
	s.respBufferPool, err = buffer.NewPool(args.ResponseBufferCao, args.ResponseMaxBufferNumber, "Response")
	if err != nil {
		logs.Error("[Buffer] --> initialization response buffer error", err)
	}
	logs.Info("[Buffer] --> Response buffer pool: bufferCap: %d, maxBufferNumber: %d, total:%d", s.respBufferPool.BufferCap(), s.respBufferPool.MaxBufferNumber(), s.respBufferPool.Total())

	if s.itemBufferPool != nil && !s.itemBufferPool.Closed() {
		s.itemBufferPool.Close()
	}
	s.itemBufferPool, err = buffer.NewPool(args.ItemBufferCap, args.ItemMaxBufferNumber, "item")
	if err != nil {
		logs.Error("[Buffer] --> initialization item processor buffer error", err)
	}
	logs.Info("[Buffer] --> item Processor buffer pool: bufferCap: %d, maxBufferNumber: %d, total:%d", s.itemBufferPool.BufferCap(), s.itemBufferPool.MaxBufferNumber(), s.itemBufferPool.Total())

	if s.errBufferPool != nil && !s.errBufferPool.Closed() {
		s.errBufferPool.Close()
	}
	s.errBufferPool, err = buffer.NewPool(args.ErrorBufferCap, args.ErrorMaxBufferNumber, "error")
	if err != nil {
		logs.Error("[Buffer] --> initialization item processor buffer error", err)
	}
	logs.Info("[Buffer] --> Error Handler buffer pool: bufferCap: %d, maxBufferNumber: %d, total:%d", s.errBufferPool.BufferCap(), s.errBufferPool.MaxBufferNumber(), s.errBufferPool.Total())

}

func (s *scheduler) Status() Status {
	var status Status
	s.lock.Lock()
	defer s.lock.Unlock()
	status = s.status
	return status
}

func (s *scheduler) Start(firstReq *http.Request) (err error) {

	// 调度器异常处理
	defer func() {
		if p := recover(); p != nil {
			errMsg := fmt.Sprintf("Fatal scheduler error %s", p)
			log.Println(errMsg)
			err = genError(errMsg)
		}
	}()

	logs.Info("[Scheduler] --> Start scheduler...")

	// 调度器状态转换
	var oldStart Status
	oldStart, err = s.checkAndSetStatus(SCHED_STATUS_INITIALIZING)
	defer func(oldStatus Status) {
		s.lock.Lock()
		if err != nil {
			s.status = oldStatus
		} else {
			s.status = SCHED_STATUS_STARTED
		}
	}(oldStart)

	if err != nil {
		return
	}

	// 参数检查
	logs.Info("[Scheduler] --> Check first Http request")
	if firstReq == nil {
		err = genError("nil first http request")
		return
	}
	logs.Info("[Scheduler] --> The first http is valid")
	logs.Info("[Scheduler] --> Get primary domain")
	logs.Info("[Scheduler] --> Host: %s ", firstReq.Host)

	// 域名处理
	var primaryDomain string
	primaryDomain, err = getPrimaryDomain(firstReq.Host)
	if err != nil {
		return
	}
	logs.Info("[Scheduler] --> PrimaryDomain %s", primaryDomain)
	s.acceptedDomianMap.Put(primaryDomain, struct{}{})

	// 缓冲器检查
	logs.Info("[Schedule] --> check buffer status")
	if err = s.checkBufferPoolForStart(); err != nil {
		return
	}
	s.download()

	s.analyze()
	s.pick()
	logs.Info("[Scheduler] --> Scheduler has been started")

	first := datastructs.NewRequest(firstReq, 0)
	logs.Info("[DownloadBuffer] --> Send Request %#v", first.String())
	if s.SendRequest(first) {
		logs.Info("[Scheduler] --> Request send successful")
	} else {
		logs.Error("[Scheduler] --> Request send failed")
	}

	return nil
}

func (s *scheduler) Stop() (err error) {

	logs.Info("[Scheduler] --> Stop scheduler...")
	logs.Info("[Scheduler] --> Check status for stop scheduler")
	var oldStatus Status
	oldStatus, err = s.checkAndSetStatus(SCHED_STATUS_STOPPING)

	defer func() {
		if err != nil {
			old := uint32(s.status)
			for {
				if atomic.CompareAndSwapUint32(&old, old, uint32(oldStatus)) {
					s.status = oldStatus
					break
				}
			}
		} else {
			old := uint32(s.status)
			for {
				if atomic.CompareAndSwapUint32(&old, old, uint32(SCHED_STATUS_STOPPED)) {
					s.status = SCHED_STATUS_STOPPED
					break
				}
			}
		}
	}()

	if err != nil {
		return
	}

	s.cancelFunc()
	s.reqBufferPool.Close()
	s.respBufferPool.Close()
	s.itemBufferPool.Close()
	s.itemBufferPool.Close()

	logs.Info("[Scheduler] --> Scheduler has been stopped.")
	return nil
}

func (s *scheduler) download() {

	go func(pool *buffer.Pool) {
		for {
			//log.Println("[DownloadBuffer] --> Start Downloader")

			if s.canceled() {
				break
			}
			if s.respBufferPool.Total() < 1 {
				<-time.After(time.Second)
			}
			data, err := (*pool).Get()

			if err != nil {
				logs.Error("[Buffer] --> The Request buffer pool was closed Break Request reception")
				break
			}

			req, ok := data.(*datastructs.Request)

			if !ok {
				errMsg := fmt.Sprintf("incorret request type %T", data)
				sendError(errors.New(errMsg), "", s.errBufferPool)
			}
			s.downloadOne(req)
		}
	}(&s.reqBufferPool)
}

func (s *scheduler) downloadOne(request *datastructs.Request) {

	if request == nil {
		return
	}
	if s.canceled() {
		return
	}

	logs.Info("[DownloadBuffer] --> Get Downloader from DownloadBuffer")
	m, err := s.register.Get(module.TYPE_DOWNLOADER)
	if err != nil || m == nil {
		//log.Printf("[DownloadBuffer] --> could`t get a downloader:%s", err)
		errMsg := fmt.Sprintf("could`t get a downloader:%s", err)
		sendError(errors.New(errMsg), "", s.errBufferPool)
		logs.Error("[DownloadBuffer] --> Send Request %#v", request)
		s.SendRequest(request)
		return
	}

	download, ok := m.(module.Downloader)
	if !ok {
		logs.Error("[DownloadBuffer] --> incorret downloader type %T MID %s", download, m.Id())
		errMsg := fmt.Sprintf("incorret downloader type %T MID %s", download, m.Id())
		sendError(errors.New(errMsg), m.Id(), s.errBufferPool)
		logs.Error("[DownloadBuffer] --> Send Request %#v", request)
		s.SendRequest(request)
		return
	}

	resp, err := download.Download(request)

	if resp != nil {
		logs.Info("[Downloader] --> Send Response URL:", resp.Response().Request.URL.String())
		s.SendResponse(resp, s.respBufferPool)
	}

	if err != nil {
		sendError(err, m.Id(), s.errBufferPool)
	}

}

func (s *scheduler) analyze() {
	go func() {
		for {
			if s.canceled() {
				break
			}

			if s.respBufferPool.Total() < 1 {
				<-time.After(time.Second)
			}

			data, err := s.respBufferPool.Get()

			if err != nil {
				logs.Error("[Analyzer] --> The response buffer was closed, Break response reception")
				break
			}
			resp, ok := data.(*datastructs.Response)
			if !ok {
				if data == nil {
					continue
				}
				errMsg := fmt.Sprintf("incorret response type %T", data)
				sendError(errors.New(errMsg), "", s.errBufferPool)
			}
			s.analyzeOne(resp)
		}
	}()
}

func (s *scheduler) analyzeOne(response *datastructs.Response) {

	if response == nil {
		return
	}

	if s.canceled() {
		return
	}

	//log.Println("[AnalyzerBuffer] --> Get a analyzer from AnalyzerBuffer")
	m, err := s.register.Get(module.TYPE_ANALYZER)
	if err != nil {
		logs.Error("[AnalyzerBuffer] --> could`t get a analyzer:%s", err)
		errMsg := fmt.Sprintf("could`t get a analyzer:%s", err)
		sendError(errors.New(errMsg), "", s.errBufferPool)
		s.SendResponse(response, s.respBufferPool)
		return
	}

	analyzer, ok := m.(module.Analyzer)
	if !ok {
		logs.Error("[AnalyzerBuffer] --> incorrect analyzer type %T MID %s", analyzer, m.Id())
		errMsg := fmt.Sprintf("incorret analyzer type %T MID %s", analyzer, m.Id())
		sendError(errors.New(errMsg), m.Id(), s.errBufferPool)
		s.SendResponse(response, s.respBufferPool)
		return
	}

	dataList, errs := analyzer.Analyze(response)
	if dataList != nil {
		for _, data := range dataList {
			if data == nil {
				continue
			}
			switch d := data.(type) {
			case *datastructs.Request:
				s.SendRequest(d)
			case *datastructs.Item:
				s.SendItem(d, s.itemBufferPool)
			default:
				errMsg := fmt.Sprintf("Unsupported data type %T (data %#v)", d, d)
				sendError(errors.New(errMsg), m.Id(), s.errBufferPool)
			}
		}

	}

	if errs != nil {
		for _, err := range errs {
			sendError(err, m.Id(), s.errBufferPool)
		}
	}
}

func (s *scheduler) pick() {

	go func() {
		for {
			if s.canceled() {
				break
			}
			data, err := s.itemBufferPool.Get()

			if data == nil {
				continue
			}
			if err != nil {
				logs.Info("[Response Buffer] --> The item buffer was closed, Break response reception")
				break
			}

			item, ok := data.(*datastructs.Item)
			if !ok {
				errMsg := fmt.Sprintf("incorret item type %T", data)
				sendError(errors.New(errMsg), "", s.errBufferPool)
			}
			s.pickOne(item)
		}
	}()
}

func (s *scheduler) pickOne(item *datastructs.Item) {
	if s.canceled() {
		return
	}
	if item == nil {
		return
	}
	logs.Info("[ItemBuffer] --> get a item from ItemBuffer")

	m, err := s.register.Get(module.TYPE_PIPLINE)
	if err != nil {
		errMsg := fmt.Sprintf("could`t get a item:%s", err)
		sendError(errors.New(errMsg), "", s.errBufferPool)
		//s.SendItem(item, s.itemBufferPool)
	}
	pipeLine, ok := m.(module.Pipeline)
	if !ok {
		errMsg := fmt.Sprintf("incorret item type %T MID %s", item, m.Id())
		sendError(errors.New(errMsg), m.Id(), s.errBufferPool)
		//s.SendItem(item, s.respBufferPool)
		return
	}
	errs := pipeLine.Send(*item)
	if errs != nil {
		for _, err := range errs {
			sendError(err, pipeLine.Id(), s.errBufferPool)
		}
	}

}

func (s *scheduler) SendRequest(request *datastructs.Request) bool {

	//log.Println("[Request] --> Checking request argument")
	if request == nil {
		return false
	}
	if s.canceled() {
		return false
	}

	httpReq := request.Request()
	if httpReq == nil {
		logs.Debug("[Request] --> Ignore the request! Its HTTP request is invalid!")
		return false
	}

	//log.Println("[Request] --> Check request URL")
	url := httpReq.URL
	if url == nil {
		logs.Debug("[Request] --> Ignore the request! Its URL is invalid!")
		return false
	}

	//log.Println("[Request] --> Check request Scheme")
	scheme := strings.ToLower(url.Scheme)
	if scheme != "http" && scheme != "https" {
		logs.Debug("[Request] --> Ignore the request! Its URL scheme is %q, but should be %q or %q. (URL: %s)", scheme, "http", "https", url)
		return false
	}
	if v, ok := s.urlMap.Get(url.String()); v != nil && !ok {
		logs.Debug("[Request] --> Ignore the request! Its URL is repeated. (URL: %s)", url)
		return false
	}

	//log.Println("[Request] --> Check request Domain")
	pl, err := getPrimaryDomain(url.Host)
	if v, ok := s.acceptedDomianMap.Get(pl); v == nil || !ok || err != nil {
		logs.Debug("[Request] --> Ignore the request! Its host %q is not in accepted primary domain map. (URL: %s)", httpReq.Host, url)
	}

	//log.Println("[Request] --> Check request Depth")
	if request.Depth() > s.maxDepth {
		logs.Info("Request --> Ignore the request! Its depth %d is greater than %d. (URL: %s)", request.Depth(), s.maxDepth, url)
		return false
	}

	go func(request2 *datastructs.Request, pool *buffer.Pool) {
		if err := s.reqBufferPool.Put(request2); err != nil {
			logs.Info("[RequestBuffer] --> The request buffer pool was closed. Ignore request sending. error:%s", err)
		}
		return
	}(request, &s.reqBufferPool)
	//logs.Info("[Request] --> Check request argument completed")
	//logs.Info("[Request] --> Download URL %s", url.String())
	s.urlMap.Put(url.String(), struct{}{})
	return true
}

func (s *scheduler) SendResponse(response *datastructs.Response, pool buffer.Pool) bool {
	if response == nil || pool == nil || pool.Closed() {
		return false
	}
	go func(response2 *datastructs.Response) {
		if err := pool.Put(response2); err != nil {
			logs.Info("[Response] --> The response buffer pool was closed ignore response sending")
			return
		}
	}(response)
	return true
}

func (s *scheduler) SendItem(item *datastructs.Item, pool buffer.Pool) bool {
	if item == nil || pool == nil || pool.Closed() {
		return false
	}
	go func(item2 *datastructs.Item) {
		if err := pool.Put(item); err != nil {
			logs.Info("[Response] --> The item buffer pool was closed. Ignore item sending.")
		}
	}(item)
	return true
}

// 检查缓冲器，如果关闭按照之前的配置重新配置
func (s *scheduler) checkBufferPoolForStart() error {

	// 检查请求缓冲器
	logs.Info("[Scheduler] --> checking request buffer")
	if s.reqBufferPool == nil {
		return genError("request buffer pool is nil")
	}
	if s.reqBufferPool != nil && s.reqBufferPool.Closed() {
		s.reqBufferPool, _ = buffer.NewPool(s.reqBufferPool.BufferCap(), s.reqBufferPool.MaxBufferNumber(), "Request")
	}

	// 检查响应缓冲器
	logs.Info("[Scheduler] --> checking response buffer")
	if s.respBufferPool == nil {
		return genError("response buffer pool is nil")
	}
	if s.respBufferPool != nil && s.respBufferPool.Closed() {
		s.respBufferPool, _ = buffer.NewPool(s.respBufferPool.BufferCap(), s.respBufferPool.MaxBufferNumber(), "Response")
	}

	// 检查条目缓冲器
	logs.Info("[Scheduler] --> checking item buffer")
	if s.itemBufferPool == nil {
		return genError("item buffer pool is nil")
	}
	if s.itemBufferPool != nil && s.itemBufferPool.Closed() {
		s.itemBufferPool, _ = buffer.NewPool(s.itemBufferPool.BufferCap(), s.itemBufferPool.MaxBufferNumber(), "item")
	}

	// 检查错误缓冲器
	logs.Info("[Scheduler] --> checking errors buffer")
	if s.errBufferPool == nil {
		return genError("errors buffer pool is nil")
	}
	if s.errBufferPool != nil && s.errBufferPool.Closed() {
		s.errBufferPool, _ = buffer.NewPool(s.errBufferPool.BufferCap(), s.errBufferPool.MaxBufferNumber(), "Error")
	}
	logs.Info("[Scheduler] --> checking buffer completed")

	logs.Info("[BufferCheck] --> request  buffer status [%v]", s.reqBufferPool.Closed())
	logs.Info("[BufferCheck] --> response buffer status [%v]", s.respBufferPool.Closed())
	logs.Info("[BufferCheck] --> item     buffer status [%v]", s.itemBufferPool.Closed())
	logs.Info("[BufferCheck] --> error    buffer status [%v]", s.errBufferPool.Closed())

	return nil

}

func (s *scheduler) ErrorChan() <-chan error {
	errBuffer := s.errBufferPool
	errCh := make(chan error, errBuffer.BufferCap())

	go func(pool buffer.Pool, errChan chan error) {
		for {
			if s.canceled() {
				logs.Debug("[Scheduler] --> scheduler is stop")
				close(errCh)
				break
			}

			data, err := pool.Get()
			if err != nil {
				logs.Error("[ErrorChan] --> The error buffer pool was closed. Break error reception.")
				close(errCh)
				break
			}

			err, ok := data.(error)
			if !ok {
				sendError(errors.New(fmt.Sprintf("incorrect error type: %T", data)), "", s.errBufferPool)
				continue
			}

			if s.canceled() {
				logs.Debug("[Scheduler] --> scheduler is stop")
				close(errCh)
				break
			}

			errCh <- err
		}
	}(errBuffer, errCh)

	return errCh
}

func (s *scheduler) Idle() bool {
	moduleMap := s.register.GetAll()
	for _, m := range moduleMap {
		if m.HandlingNumber() > 0 {
			return false
		}
	}
	return s.reqBufferPool.Total() > 0 || s.respBufferPool.Total() > 0 || s.itemBufferPool.Total() > 0
}

func (s *scheduler) Summary() SchedSummary {
	return s.summary
}

func NewScheduler(args RequestArgs, moduleArgs ModuleArgs, dataArgs DataArgs) Scheduler {

	se := &scheduler{}
	err := se.Init(args, dataArgs, moduleArgs)
	if err != nil {
		panic(err)
	}
	return se
}

type ConfigOfModule struct {
	NumberOfDownload uint8
	NumberOfAnalyzer uint8
	NumberOfPipeline uint8
	Domain           []string
	SavePath         string
	MaxDepth         uint32
}

type Config struct {
	ConfigOfModule
	Client         *http.Client
	Processors     []module.ProcessItem
	ParserResponse []module.ParserResponse
}

func NewDefaultScheduler(config *Config) Scheduler {
	dataArgs := DataArgs{
		RequestBufferCap:        55,
		RequestMaxBufferNumber:  1000,
		ResponseBufferCao:       50,
		ResponseMaxBufferNumber: 10,
		ItemBufferCap:           50,
		ItemMaxBufferNumber:     100,
		ErrorBufferCap:          50,
		ErrorMaxBufferNumber:    1,
	}
	reqArgs := RequestArgs{
		AcceptedDomains: config.Domain,
		MaxDepth:        config.MaxDepth,
	}

	download, err := downloader.GetDownloaders(config.NumberOfDownload, config.Client)
	if err != nil {
		logs.Error(err)
	}
	analyzer, err := analyzer2.GetAnalyzers(config.NumberOfAnalyzer, config.ParserResponse)
	if err != nil {
		logs.Error(err)
	}
	pipeLine, err := pipeline.GetPipelines(config.NumberOfPipeline, config.Processors)
	if err != nil {
		logs.Error(err)
	}
	modArgs := ModuleArgs{
		Downloaders: download,
		Analyzer:    analyzer,
		Pipline:     pipeLine,
	}

	return NewScheduler(reqArgs, modArgs, dataArgs)
}
