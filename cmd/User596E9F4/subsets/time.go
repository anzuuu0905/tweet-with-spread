package subsets

import (
	"math/rand"
	"time"
)

// Wait は、指定秒を最大に、0~maxsec秒待機します。
func Wait(maxSec int) {
	waitMillisec := rand.Intn(maxSec * 1000)
	time.Sleep(time.Duration(waitMillisec) * time.Millisecond)
}
