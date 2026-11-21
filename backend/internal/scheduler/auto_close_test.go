package scheduler

import "testing"

func TestAutoCloseDue(t *testing.T) {
	const minute = int64(60000)
	end := int64(1_000_000_000_000)
	cases := []struct {
		name      string
		now       int64
		delayMin  int
		autoClose bool
		want      bool
	}{
		{"未到结束", end - minute, 0, true, false},
		{"刚到点(延迟0)", end, 0, true, true},
		{"延迟5分钟未到", end + 4*minute, 5, true, false},
		{"延迟5分钟已到", end + 5*minute, 5, true, true},
		{"未开启", end + 10*minute, 0, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := autoCloseDue(c.now, end, c.delayMin, c.autoClose); got != c.want {
				t.Fatalf("autoCloseDue(now=%d, delay=%d, on=%v)=%v want %v", c.now, c.delayMin, c.autoClose, got, c.want)
			}
		})
	}
}
