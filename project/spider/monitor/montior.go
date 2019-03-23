package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"helper/logs"
	"log"
	"runtime"
	"spider/schedluer"
	"time"
)

// summary 代表监控结果摘要的结构。
type summary struct {
	NumGoroutine int               `json:"goroutine_number"` // NumGoroutine 代表Goroutine的数量。
	SchedSummary schedluer.Summary `json:"sched_summary"`    // SchedSummary 代表调度器的摘要信息。
	EscapedTime  string            `json:"escaped_time"`     // EscapedTime 代表从开始监控至今流逝的时间。
}

// msgReachMaxIdleCount 代表已达到最大空闲计数的消息模板。
var msgReachMaxIdleCount = "[Monitor] --> The scheduler has been idle for a period of time" +
	" (about %s)." + " Consider to stop it now."

// msgStopScheduler 代表停止调度器的消息模板。
var msgStopScheduler = "Stop scheduler...%s."

// Record 代表日志记录函数的类型。
// 参数level代表日志级别。级别设定：0-普通；1-警告；2-错误。
//type Record func(level uint8, content string)

func NewDefaultMonitor(scheduler *schedluer.Scheduler) <-chan uint64 {
	checkInterval := time.Second
	summarizeInterval := 100 * time.Millisecond
	maxIdleCount := uint(5)
	return Monitor(scheduler, checkInterval, summarizeInterval, maxIdleCount, true)
}

// Monitor 用于监控调度器。
// 参数scheduler代表作为监控目标的调度器。
// 参数checkInterval代表检查间隔时间，单位：纳秒。
// 参数summarizeInterval代表摘要获取间隔时间，单位：纳秒。
// 参数maxIdleCount代表最大空闲计数。
// 参数autoStop被用来指示该方法是否在调度器空闲足够长的时间之后自行停止调度器。
// 参数record代表日志记录函数。
// 当监控结束之后，该方法会向作为唯一结果值的通道发送一个代表了空闲状态检查次数的数值。
func Monitor(scheduler *schedluer.Scheduler, checkInterval time.Duration, summarizeInterval time.Duration, maxIdleCount uint, autoStop bool) <-chan uint64 {
	// 防止调度器不可用。
	if scheduler == nil {
		panic(errors.New("The scheduler is invalid!"))
	}
	// 防止过小的检查间隔时间对爬取流程造成不良影响。
	if checkInterval < time.Millisecond*100 {
		checkInterval = time.Millisecond * 100
	}
	// 防止过小的摘要获取间隔时间对爬取流程造成不良影响。
	if summarizeInterval < time.Second {
		summarizeInterval = time.Second
	}
	// 防止过小的最大空闲计数造成调度器的过早停止。
	if maxIdleCount < 10 {
		maxIdleCount = 10
	}
	logs.Info("[Monitor] --> Monitor parameters: checkInterval: %s, summarizeInterval: %s,"+
		" maxIdleCount: %d, autoStop: %v",
		checkInterval, summarizeInterval, maxIdleCount, autoStop)
	// 生成监控停止通知器。
	stopNotifier, stopFunc := context.WithCancel(context.Background())
	// 接收和报告错误。
	reportError(scheduler, stopNotifier)
	// 记录摘要信息。
	recordSummary(scheduler, summarizeInterval, stopNotifier)
	// 检查计数通道
	checkCountChan := make(chan uint64, 2)
	// 检查空闲状态
	checkStatus(scheduler, checkInterval, maxIdleCount, autoStop, checkCountChan, stopFunc)
	return checkCountChan
}

// checkStatus 用于检查状态，并在满足持续空闲时间的条件时采取必要措施。
func checkStatus(scheduler *schedluer.Scheduler, checkInterval time.Duration, maxIdleCount uint, autoStop bool, checkCountChan chan<- uint64, stopFunc context.CancelFunc) {
	go func() {
		logs.Info("[Monitor] --> Check Schedluer Status")
		var checkCount uint64
		defer func() {
			stopFunc()
			checkCountChan <- checkCount
		}()
		// 等待调度器开启。
		waitForSchedulerStart(scheduler)
		// 准备。
		var idleCount uint
		var firstIdleTime time.Time
		for {
			// 检查调度器的空闲状态。
			if (*scheduler).Idle() {
				idleCount++
				if idleCount == 1 {
					firstIdleTime = time.Now()
				}
				if idleCount >= maxIdleCount {
					msg := fmt.Sprintf(msgReachMaxIdleCount, time.Since(firstIdleTime).String())
					log.Println(msg)
					// 再次检查调度器的空闲状态，确保它已经可以被停止。
					if (*scheduler).Idle() {
						if autoStop {
							var result string
							if err := (*scheduler).Stop(); err == nil {
								result = "success"
							} else {
								result = fmt.Sprintf("failing(%s)", err)
							}
							msg = fmt.Sprintf(msgStopScheduler, result)
							log.Println(msg)
						}
						break
					} else {
						if idleCount > 0 {
							idleCount = 0
						}
					}
				}
			} else {
				if idleCount > 0 {
					idleCount = 0
				}
			}
			checkCount++
			time.Sleep(checkInterval)
		}
	}()
}

// recordSummary 用于记录摘要信息。
func recordSummary(scheduler *schedluer.Scheduler, summarizeInterval time.Duration, stopNotifier context.Context) {
	go func() {
		logs.Info("[Record] --> Record Schedluer Summary")
		// 等待调度器开启。
		logs.Info("[Record] --> Wait for Scheduler start")
		waitForSchedulerStart(scheduler)
		logs.Info("[Record] --> Record Scheduler is started")

		// 准备。
		//var prevSchedSummaryStruct schedluer.Summary
		//var prevNumGoroutine int
		var recordCount uint64 = 1
		startTime := time.Now()
		for {
			select {
			case <-stopNotifier.Done(): // 检查监控停止通知器。
				return
			default:
			}
			currNumGoroutine := runtime.NumGoroutine()                // 获取Goroutine数量和调度器摘要信息。
			currSchedSummaryStruct := (*scheduler).Summary().Struct() // 比对前后两份摘要信息的一致性。只有不一致时才会记录。

			//if currNumGoroutine != prevNumGoroutine || !currSchedSummaryStruct.Same(prevSchedSummaryStruct) {
				// 记录摘要信息。
				summay := summary{
					NumGoroutine: runtime.NumGoroutine(),
					SchedSummary: currSchedSummaryStruct,
					EscapedTime:  time.Since(startTime).String(),
				}
				b, err := json.MarshalIndent(summay, "", "    ")
				if err != nil {
					log.Printf("[Monitor[ --> An error occurs when generating scheduler summary: %s\n", err)
					continue
				}
				msg := fmt.Sprintf("Monitor summary[%d]:\n%s", recordCount, b)
				logs.Info(msg)
			logs.Info("Current goroutine %d",currNumGoroutine)

			//prevNumGoroutine = currNumGoroutine
				//prevSchedSummaryStruct = currSchedSummaryStruct
				recordCount++
			//}
			time.Sleep(summarizeInterval)
		}
	}()
}

// reportError 用于接收和报告错误。
func reportError(scheduler *schedluer.Scheduler, stopNotifier context.Context) {
	go func() {
		logs.Info("[Monitor] --> Start Report Error")
		// 等待调度器开启。
		waitForSchedulerStart(scheduler)
		errorChan := (*scheduler).ErrorChan()
		for {
			// 查看监控停止通知器。
			select {
			case <-stopNotifier.Done():
				return
			default:
			}
			err, ok := <-errorChan
			if ok {
				errMsg := fmt.Sprintf("Received an error from error channel: %s", err)
				log.Fatal(errMsg)
			}
			time.Sleep(time.Microsecond)
		}
	}()
}

// waitForSchedulerStart 用于等待调度器开启。
func waitForSchedulerStart(scheduler *schedluer.Scheduler) {
	for (*scheduler).Status() != schedluer.SCHED_STATUS_STARTED {
		time.Sleep(time.Microsecond)
	}
}
