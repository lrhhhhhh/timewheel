package timewheel

import (
	"fmt"
	"time"
)

type Event struct {
	Id       int //
	Interval int // 事件每隔interval毫秒运行一次

	// 记录上一次运行的时间信息, 配合时间轮当前时间 和 interval 可以计算出下一次运行的具体时间
	// 初始化除了step外，全为0 (即不支持interval为0)
	lastTime timeinfo

	Cnt      int    // 执行次数，负数代表执行无数次
	RunSync  bool   // 是否同步执行（阻塞主循环）
	Callback func() // 事件的运行函数
}

func (e *Event) String() string {
	return fmt.Sprintf("Event(id=%d,interval=%s,cnt=%d)", e.Id, time.Duration(e.Interval), e.Cnt)
}
