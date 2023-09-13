package timewheel

import (
	"fmt"
	_ "net/http/pprof"
	"testing"
	"time"
)

var step int

type testCase struct {
	interval int
	cnt      int
}

var tc = []testCase{
	{20 * Millisecond, -1},
	{30 * Millisecond, -1},
	{50 * Millisecond, -1},
	{100 * Millisecond, -1},
	{200 * Millisecond, -1},
	{1 * Second, 3},
	{20 * Millisecond, -1},
	{2 * Second, -1},
	{2 * Second, 2},
	{3 * Second, -1},
	{5 * Second, -1},
	{30 * Second, -1},
	{1 * Minute, -1},
	{5 * Minute, -1},
}

var g = map[int]int{}

func helper(id, interval int) {
	var d int
	now := time.Now().UnixNano()
	if v := g[interval]; v != 0 {
		d = int(now) - v - interval
		if d > step { // 判断误差, 如果误差超过step（即一步的长度）则panic
			fmt.Printf("now=%v, last=%v, delta=%v > step=%v, id=%d\n", now, v, time.Duration(d), step, id)
			//panic(interval)
		}
	}
	//fmt.Printf("Exec Event(id=%d, interval=%dms), delta=%v\n", id, interval/Millisecond, time.Duration(d))
	g[interval] = int(now)
}

func TestTimeWheel(t *testing.T) {
	step = 5 * Millisecond
	tw, err := New(step)
	if err != nil {
		panic(err)
	}

	for k, v := range tc {
		id := k
		interval := v.interval
		if interval < step {
			continue
		}
		err := tw.Put(&Event{
			Id:       id,
			Cnt:      v.cnt,
			Interval: interval,
			lastTime: timeinfo{step: step},
			RunSync:  true,
			Callback: func() { helper(id, interval) },
		})
		if err != nil {
			panic(err)
		}
	}

	fmt.Printf("%+v\n", tw)
	fmt.Println("Size: ", tw.Size())
	tw.Run()
}
