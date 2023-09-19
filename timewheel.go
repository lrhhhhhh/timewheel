package timewheel

import (
	"container/list"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	Millisecond = int(time.Millisecond)
	Second      = int(time.Second)
	Minute      = int(time.Minute)
	Hour        = int(time.Hour)
	Day         = int(time.Hour * 24)

	MaxDay = 366 // 时间轮最大值
)

var ErrInvalidStep = errors.New("invalid step")
var ErrInvalidInterval = errors.New("invalid interval")

type TimeWheel struct {
	millisecondCnt int
	secondCnt      int
	minuteCnt      int
	hourCnt        int
	dayCnt         int

	ticker      *time.Ticker
	maxInterval int                 // 当前时间轮所能表示的最大的时间，超过这个时间将被取模
	eventCnt    int                 // 时间轮中当前共有多少event
	eventSet    map[string]struct{} // key=Event.Key
	time        timeinfo            // 当前时间轮时间
	slots       []*list.List        // from 0 ~ n - 1 : millisecond | second | minute | hour | day
	locker      *sync.Mutex         // 锁住整个slots
}

// Run 主循环
func (tw *TimeWheel) Run() {
	fmt.Println("TimeWheel is Running, step=", time.Duration(tw.time.step))
	for {
		select {
		case <-tw.ticker.C:
			next := tw.afterInterval(tw.time.step)
			index := tw.index(next)
			tw.time = next

			if err := tw.handleSlot(index); err != nil {
				log.Println("handle err: ", err)
			}
		}
	}
}

// handleSlot 处理index指定的slot, 每个slot里存储着一个事件链表
// 处理到某个slot，意味着这个slot里的event，要么现在执行，要么往下推
// Put操作和handleSlot操作互斥，不允许同时修改tw.slots
func (tw *TimeWheel) handleSlot(index int) error {
	tw.locker.Lock()
	defer tw.locker.Unlock()
	for it := tw.slots[index].Front(); it != nil; it = it.Next() {
		event := it.Value.(*Event)
		now := tw.time.unixNano()
		last := event.lastTime.unixNano()
		if event.Interval <= now-last {
			if event.RunSync {
				event.Callback()
			} else {
				go event.Callback()
			}
			if event.Cnt > 0 {
				event.Cnt -= 1
			}
			if event.Cnt == 0 { // remove event from timewheel
				tw.eventCnt -= 1
				delete(tw.eventSet, event.Key)
			} else {
				event.lastTime = tw.time
				tw.insertAfter(event.Interval, event)
			}
		} else {
			tw.insertAfter(last+event.Interval-now, event)
		}
	}
	tw.slots[index].Init() // 重置当前slot
	return nil
}

func (tw *TimeWheel) validate(e *Event) bool {
	if e.Interval < tw.time.step || e.Interval%tw.time.step != 0 ||
		e.Interval >= tw.maxInterval || e.Cnt == 0 ||
		e.Key == "" || e.Callback == nil {
		return false
	}
	return true
}

// Put 添加Event到时间轮中
// 每次插入的 Event，都插入到当前时间轮时间的 interval 微秒后
func (tw *TimeWheel) Put(e *Event) error {
	if !tw.validate(e) {
		return ErrInvalidInterval
	}
	tw.locker.Lock()
	defer tw.locker.Unlock()
	tw.insertAfter(e.Interval, e)
	tw.eventCnt += 1
	tw.eventSet[e.Key] = struct{}{}
	return nil
}

func (tw *TimeWheel) Size() int {
	return tw.eventCnt
}

func (tw *TimeWheel) Find(key string) bool {
	if _, ok := tw.eventSet[key]; ok {
		return true
	}
	return false
}

// insert 将事件e插入到当前时间轮时间的interval微秒后
// 使用此方法需要保证获得了locker锁
func (tw *TimeWheel) insertAfter(interval int, e *Event) int {
	future := tw.afterInterval(interval)
	index := tw.index(future)
	tw.slots[index].PushBack(e)
	return index
}

// 计算当前时间轮经过interval后的时间, 单位nanosecond
func (tw *TimeWheel) afterInterval(interval int) timeinfo {
	future := tw.time.unixNano() + interval
	future %= tw.maxInterval // mod max interval
	return timeinfo{
		step: tw.time.step,
		d:    future / Day,
		h:    (future % Day) / Hour,
		m:    (future % Hour) / Minute,
		s:    (future % Minute) / Second,
		ms:   (future % Second) / tw.time.step,
	}
}

// index根据当前时间轮时间计算t（未来时间）对应的slot的位置
func (tw *TimeWheel) index(t timeinfo) int {
	var index int
	if tw.time.d != t.d {
		index = t.d + tw.hourCnt + tw.minuteCnt + tw.secondCnt + tw.millisecondCnt
	} else if tw.time.h != t.h {
		index = t.h + tw.minuteCnt + tw.secondCnt + tw.millisecondCnt
	} else if tw.time.m != t.m {
		index = t.m + tw.secondCnt + tw.millisecondCnt
	} else if tw.time.s != t.s {
		index = t.s + tw.millisecondCnt
	} else if tw.time.ms != t.ms {
		index = t.ms
	}
	return index
}

// New
// step 是精度，即每一步时长，推荐 5ms 以上，保证1000%step为0
func New(step int) (*TimeWheel, error) {
	if Second%step != 0 {
		return nil, ErrInvalidStep
	}
	tw := &TimeWheel{
		ticker:         time.NewTicker(time.Duration(step)),
		time:           timeinfo{step: step}, // 初始化为0
		millisecondCnt: Second / step,        // 1秒划分成多少个slot
		secondCnt:      60,                   // 60个1秒的slot
		minuteCnt:      60,                   // 60个1分的slot
		hourCnt:        24,                   // 24个1小时的slot
		dayCnt:         MaxDay,               // 时间轮最多支持多少天，超过则取模
		locker:         &sync.Mutex{},
		eventSet:       make(map[string]struct{}),
	}
	total := tw.millisecondCnt + tw.secondCnt + tw.minuteCnt + tw.hourCnt + tw.dayCnt
	tw.slots = make([]*list.List, total)
	for i := range tw.slots {
		tw.slots[i] = list.New()
	}

	maxInterval := tw.time.step * tw.millisecondCnt * tw.secondCnt * tw.minuteCnt * tw.hourCnt * tw.dayCnt
	tw.maxInterval = maxInterval
	return tw, nil
}
