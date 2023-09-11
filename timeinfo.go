package timewheel

// timeinfo 用来记录当前时间轮在哪一个slot，也可以理解为时间轮所有指针的当前指向位置
// 例如22:23:33:800, step 为20ms，那么表示为timeinfo{h: 22, m: 23, s: 33, ms: 40}
type timeinfo struct {
	step int // step控制精度，即把1000毫秒划分成x份，每份长度为step
	d    int // Day Hand
	h    int // 时针
	m    int // 分针
	s    int // 秒针
	ms   int // 毫秒针   1000 / step
}

// 类似于time.UnixNano(), 返回当前timeinfo所表示的总纳秒数
func (t *timeinfo) unixNano() int {
	return t.step*t.ms + t.s*Second + t.m*Minute + t.h*Hour + t.d*Day
}
