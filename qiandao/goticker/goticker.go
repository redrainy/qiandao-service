package goticker

/**
 * 计划任务
 * 用于替代 go 自带的 time.Ticker
 * @Auther QiuXiangCheng
 * @DateTime 2019/12/05 12:00:09
 */

import (
	"sync"
	"time"
)

type Tasks struct {
	exec       bool // 是否在打开任务时执行一次
	exit       bool
	ChanClosed chan bool
	sync.Mutex
	maxTsid        int
	scanning_cycle time.Duration // 每间隔 scanning_cycle 秒扫一次任务
	taskList       []*Task
}

type Task struct {
	id         int
	expireTime int64     // 任务间隔时间
	lastHnTime int64     // 上一次执行时间
	taskType   uint8     // 1-普通周期性任务 2-定时周期性任务
	timerTime  time.Time // 针对定时周期性任务时使用
	cbackChan  chan<- bool
	taskParams interface{}
	handleFunc func(arg interface{})
}

func New(scanningTime time.Duration, ex bool) *Tasks {
	if scanningTime < 50 {
		scanningTime = 50
	}
	ts := &Tasks{
		scanning_cycle: scanningTime,
		exec:           ex,
		ChanClosed:     make(chan bool, 1),
	}
	go ts.ListeningTasks()
	return ts
}

func (ts *Tasks) Exit() {
	ts.exit = true
}

func (ts *Tasks) Exec(e bool) *Tasks {
	ts.exec = e
	return ts
}

/**
 * 新增一个周期性运转的计划任务
 * 当被触发时向管道 ch 写入 bool
 */
func (ts *Tasks) AddTaskCallBackChannel(ch chan<- bool, expire int64) int {
	ts.Lock()
	defer ts.Unlock()

	task := &Task{
		id:         ts.maxTsid,
		taskType:   1, // 普通周期性任务，如：每隔 10 秒运行一次的任务
		expireTime: expire,
		cbackChan:  ch,
	}
	if !ts.exec {
		task.lastHnTime = time.Now().Unix()
	}
	ts.maxTsid++
	ts.taskList = append(ts.taskList, task)
	return ts.maxTsid
}

/**
 * 新增一个周期性运转的计划任务
 * 当被触发时执行回调函数 k
 */
func (ts *Tasks) AddTaskCallBackFunc(k func(arg interface{}), expire int64, params interface{}) int {
	ts.Lock()
	defer ts.Unlock()

	ts.maxTsid++
	task := &Task{
		id:         ts.maxTsid,
		taskType:   1, // 1-指定为 ticker 周期性任务, 如：每隔 10 秒运行一次的任务
		expireTime: expire,
		handleFunc: k,
		taskParams: params,
	}
	if !ts.exec {
		task.lastHnTime = time.Now().Unix()
	}
	ts.taskList = append(ts.taskList, task)
	return ts.maxTsid
}

// 新增一个计划任务 以时/分/秒为周期 每天运行一次
// 如果当前新增的任务的执行时间已经小于当前时间 则将该任务下一次执行时间推迟到明天
// 调用方式，当天的 14:34:50 执行一次：task.AddCycleTaskCallBackFunc(test, "14:34:50", nil)

// sr := strtotime(nt.Format("2006-01-02") + " 20:04:00")
// nt := time.Now().UTC()
// sr := strtotime(nt.Format("2006-01-02") + " 20:04:00")
// Println(sr.After(nt))
// Println(nt.Unix())
// sr = sr.AddDate(0, 0, 1)
// Println(sr)
// return

func (ts *Tasks) AddCycleTaskCallBackFunc(k func(arg interface{}), Time string, params interface{}) int {
	ts.Lock()
	defer ts.Unlock()

	nt := time.Now().UTC()
	sr := strtotime(nt.Format("2006-01-02") + " " + Time)
	if !sr.After(nt) {
		sr = sr.AddDate(0, 0, 1) // 推迟一天执行
	}
	ts.maxTsid++

	ts.taskList = append(ts.taskList, &Task{
		id:         ts.maxTsid,
		taskType:   2, // 2-指定为 timer 定时周期性任务, 如：每天5:00:00运行一次的任务
		timerTime:  sr,
		handleFunc: k,
		taskParams: params,
	})
	return ts.maxTsid
}

// 周期性任务，当任务条件达到时，向管道回写 bool true
func (ts *Tasks) AddCycleTaskCallBackChannel(ch chan<- bool, Time string) int {
	ts.Lock()
	defer ts.Unlock()

	nt := time.Now().UTC()
	sr := strtotime(nt.Format("2006-01-02") + " " + Time)
	if !sr.After(nt) {
		sr = sr.AddDate(0, 0, 1) // 推迟一天执行
	}
	ts.maxTsid++
	ts.taskList = append(ts.taskList, &Task{
		id:        ts.maxTsid,
		taskType:  2, // 2-指定为 timer 定时周期性任务, 如：每天5:00:00运行一次的任务
		timerTime: sr,
		cbackChan: ch,
	})
	return ts.maxTsid
}

// 串行异步非阻塞处理任务
func (ts *Tasks) ListeningTasks() {
	for {
		if ts.exit {
			ts.ChanClosed <- true
			println("exit for running task...")
			return
		}
		// 周期性任务
		now := time.Now().UTC()
		for _, task := range ts.taskList {
			handle := false
			if task.taskType == 1 && now.Unix()-task.lastHnTime+1 > task.expireTime {
				handle = true
				task.lastHnTime = now.Unix()
			} else if task.taskType == 2 && now.After(task.timerTime) {
				handle = true
				task.timerTime = task.timerTime.AddDate(0, 0, 1)
			}
			if handle {
				if task.handleFunc != nil {
					go task.handleFunc(task.taskParams)
				} else {
					go func(t *Task) {
						select {
						case t.cbackChan <- true:
						case <-time.After(time.Second):
							println("time out...")
						}
					}(task)
				}
			}
		}
		time.Sleep(time.Millisecond * ts.scanning_cycle)
	}
}

// 传入任务 id 以取消该任务
func (ts *Tasks) Cancle(tsid int) {
	ts.Lock()
	defer ts.Unlock()
	for i, task := range ts.taskList {
		if task.id == tsid {
			ts.taskList = append(ts.taskList[:i], ts.taskList[i+1:]...)
		}
	}
}
