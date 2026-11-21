package handler

import (
	"sync"
	"time"

	"smart-space-management/backend/internal/model/device"
)

type doorTimer interface {
	Stop() bool
}

type doorTimerFactory func(time.Duration, func()) doorTimer

type doorAutoCloseEntry struct {
	generation uint64
	timer      doorTimer
}

type doorAutoCloseManager struct {
	mu      sync.Mutex
	entries map[uint64]doorAutoCloseEntry
	nextGen uint64
	after   doorTimerFactory
}

func newDoorAutoCloseManager(after doorTimerFactory) *doorAutoCloseManager {
	return &doorAutoCloseManager{entries: make(map[uint64]doorAutoCloseEntry), after: after}
}

func (m *doorAutoCloseManager) schedule(deviceID uint64, delay time.Duration, closeDoor func()) {
	m.mu.Lock()
	if old, ok := m.entries[deviceID]; ok {
		old.timer.Stop()
	}
	m.nextGen++
	generation := m.nextGen
	timer := m.after(delay, func() {
		m.mu.Lock()
		current, ok := m.entries[deviceID]
		if !ok || current.generation != generation {
			m.mu.Unlock()
			return
		}
		delete(m.entries, deviceID)
		m.mu.Unlock()
		closeDoor()
	})
	m.entries[deviceID] = doorAutoCloseEntry{generation: generation, timer: timer}
	m.mu.Unlock()
}

func (m *doorAutoCloseManager) cancel(deviceID uint64) {
	m.mu.Lock()
	if entry, ok := m.entries[deviceID]; ok {
		entry.timer.Stop()
		delete(m.entries, deviceID)
	}
	m.mu.Unlock()
}

func doorCloseCommand(dev device.Device) string {
	if dev.Type != device.TypeDoorLock {
		return ""
	}
	if cmd := readDeviceCommandConfig(dev.ConfigJSON).CmdOff; cmd != "" {
		return cmd
	}
	return "door_close"
}

func doorOpenCommand(dev device.Device, cmd string) bool {
	if dev.Type != device.TypeDoorLock {
		return false
	}
	configured := readDeviceCommandConfig(dev.ConfigJSON).CmdOn
	if configured != "" {
		return cmd == configured
	}
	return cmd == "door_open"
}

func doorManualCloseCommand(dev device.Device, cmd string) bool {
	if dev.Type != device.TypeDoorLock {
		return false
	}
	configured := readDeviceCommandConfig(dev.ConfigJSON).CmdOff
	if configured != "" {
		return cmd == configured
	}
	return cmd == "door_close"
}
