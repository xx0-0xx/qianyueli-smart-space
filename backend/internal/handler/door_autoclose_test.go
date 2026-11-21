package handler

import (
	"sync"
	"testing"
	"time"

	"smart-space-management/backend/internal/model/device"
)

type fakeDoorTimer struct {
	stopped bool
	fn      func()
}

func (t *fakeDoorTimer) Stop() bool {
	wasActive := !t.stopped
	t.stopped = true
	return wasActive
}

func (t *fakeDoorTimer) fire() {
	if !t.stopped {
		t.fn()
	}
}

func TestDoorAutoCloseResetsFromLastOpen(t *testing.T) {
	var mu sync.Mutex
	var timers []*fakeDoorTimer
	manager := newDoorAutoCloseManager(func(_ time.Duration, fn func()) doorTimer {
		timer := &fakeDoorTimer{fn: fn}
		mu.Lock()
		timers = append(timers, timer)
		mu.Unlock()
		return timer
	})

	var closes []uint64
	manager.schedule(41, 3*time.Second, func() { closes = append(closes, 41) })
	manager.schedule(41, 3*time.Second, func() { closes = append(closes, 41) })

	if len(timers) != 2 || !timers[0].stopped {
		t.Fatal("second open must cancel the first pending close")
	}
	timers[0].fire()
	if len(closes) != 0 {
		t.Fatal("superseded close fired")
	}
	timers[1].fire()
	if len(closes) != 1 {
		t.Fatalf("close count = %d, want 1", len(closes))
	}
}

func TestDoorManualCloseCancelsPendingAutoClose(t *testing.T) {
	timer := &fakeDoorTimer{fn: func() { t.Fatal("cancelled close fired") }}
	manager := newDoorAutoCloseManager(func(_ time.Duration, _ func()) doorTimer { return timer })
	manager.schedule(99, 3*time.Second, func() { t.Fatal("cancelled close callback ran") })
	manager.cancel(99)
	timer.fire()
	if !timer.stopped {
		t.Fatal("manual close must stop pending auto-close timer")
	}
}

func TestDoorAutoCloseUsesConfiguredOffCommand(t *testing.T) {
	dev := device.Device{Type: device.TypeDoorLock, ConfigJSON: `{"cmd_on":"unlock","cmd_off":"lock"}`}
	if got := doorCloseCommand(dev); got != "lock" {
		t.Fatalf("close command = %q, want lock", got)
	}
	dev.ConfigJSON = "{}"
	if got := doorCloseCommand(dev); got != "door_close" {
		t.Fatalf("fallback close command = %q, want door_close", got)
	}
	dev.Type = device.TypeLight
	if got := doorCloseCommand(dev); got != "" {
		t.Fatalf("non-lock close command = %q, want empty", got)
	}
}
