package schedluer

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type Status uint32

const (
	SCHED_STATUS_UNINITIALIZED Status = 0
	SCHED_STATUS_INITIALIZING  Status = 1
	SCHED_STATUS_INITIALIZED   Status = 2
	SCHED_STATUS_STARTING      Status = 3
	SCHED_STATUS_STARTED       Status = 4
	SCHED_STATUS_STOPPING      Status = 5
	SCHED_STATUS_STOPPED       Status = 6
)

func (s *scheduler) checkAndSetStatus(wantStatus Status) (oldStatus Status, err error) {
	//s.lock.Lock()
	//defer s.lock.Unlock()
	err = checkStatus(s.status, wantStatus, nil)
	if err == nil {
		old := uint32(s.status)
		for {
			if atomic.CompareAndSwapUint32(&old, old, uint32(wantStatus)) {
				s.status = Status(old)
				break
			}
		}
	}
	return
}

// checkStatus 用于状态的检查。
// 参数currentStatus代表当前的状态。
// 参数wantedStatus代表想要的状态。
// 检查规则：
//     1. 处于正在初始化、正在启动或正在停止状态时，不能从外部改变状态。
//     2. 想要的状态只能是正在初始化、正在启动或正在停止状态中的一个。
//     3. 处于未初始化状态时，不能变为正在启动或正在停止状态。
//     4. 处于已启动状态时，不能变为正在初始化或正在启动状态。
//     5. 只要未处于已启动状态就不能变为正在停止状态。
func checkStatus(current Status, wantStatus Status, lock sync.Locker) (err error) {
	if lock != nil {
		lock.Lock()
		defer lock.Unlock()
	}

	switch current {
	case SCHED_STATUS_INITIALIZING:
		err = genError("the scheduler is being initialized!")
	case SCHED_STATUS_STARTING:
		err = genError("the scheduler is being started!")
	case SCHED_STATUS_STOPPING:
		err = genError("the scheduler is being stopped!")
	}
	if err != nil {
		return
	}

	if current == SCHED_STATUS_UNINITIALIZED && (wantStatus == SCHED_STATUS_STARTING || wantStatus == SCHED_STATUS_STOPPING) {
		err = genError("the scheduler has not yet been initialized!")
	}

	switch wantStatus {
	case SCHED_STATUS_INITIALIZING:
		if current == SCHED_STATUS_STARTED {
			err = genError("he scheduler has been started!")
		}
	case SCHED_STATUS_STARTING:
		if current == SCHED_STATUS_UNINITIALIZED {
			err = genError("the scheduler has not been initialized!")
		} else if current == SCHED_STATUS_STARTED {
			err = genError("the scheduler has been started!")
		}
	case SCHED_STATUS_STOPPING:
		if current != SCHED_STATUS_STARTED {
			err = genError("the scheduler has not been started!")
		}
	default:
		err = genError(fmt.Sprintf("unsupported wanted status for check! (wantedStatus: %d)", wantStatus))
	}
	return
}

func GetGetStatusDescription(status Status) string {
	switch status {
	case SCHED_STATUS_UNINITIALIZED:
		return "uninitialized"
	case SCHED_STATUS_INITIALIZING:
		return "initializing"
	case SCHED_STATUS_INITIALIZED:
		return "initialized"
	case SCHED_STATUS_STARTING:
		return "starting"
	case SCHED_STATUS_STARTED:
		return "started"
	case SCHED_STATUS_STOPPING:
		return "stopping"
	case SCHED_STATUS_STOPPED:
		return "stopped"
	default:
		return "unknown"
	}
}
