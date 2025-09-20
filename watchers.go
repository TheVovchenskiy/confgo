package confgo

import (
	"sync"
	"time"
)

const (
	pollInterval = 3 * time.Second
)

// ModTimer interface defines the contract for objects that can report their modification time.
type ModTimer interface {
	// ModTime returns the last modification time of the data.
	ModTime() (time.Time, error)
}

var _ Watcher = (*ModTimeWatcher)(nil)

// ModTimeWatcher is a watcher that monitors file modification times to detect configuration changes.
type ModTimeWatcher struct {
	modTimer ModTimer
	interval time.Duration
	stop     chan struct{}
	lastMod  time.Time
}

func NewModTimeWatcher(modTimer ModTimer) *ModTimeWatcher {
	return &ModTimeWatcher{
		modTimer: modTimer,
		interval: pollInterval,
		stop:     make(chan struct{}),
	}
}

func (fw *ModTimeWatcher) Watch(callback func()) {
	go func() {
		for {
			select {
			case <-fw.stop:
				return
			case <-time.After(fw.interval):
				modTime, err := fw.modTimer.ModTime()
				if err != nil {
					continue
				}
				if fw.lastMod.IsZero() {
					fw.lastMod = modTime
				} else if modTime.After(fw.lastMod) {
					fw.lastMod = modTime
					callback()
				}
			}
		}
	}()
}

func (fw *ModTimeWatcher) Stop() error {
	close(fw.stop)
	return nil
}

var _ Watcher = (*TriggerWatcher)(nil)

// TriggerWatcher is a simple watcher that calls a callback every time the Trigger method is called.
// In practice, it's useful for testing.
type TriggerWatcher struct {
	mu       sync.Mutex
	callback func()
}

func NewTriggerWatcher() *TriggerWatcher {
	return &TriggerWatcher{
		mu:       sync.Mutex{},
		callback: nil,
	}
}

func (tw *TriggerWatcher) Watch(callback func()) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.callback = callback
}

func (tw *TriggerWatcher) Stop() error {
	return nil
}

func (tw *TriggerWatcher) Trigger() {
	tw.mu.Lock()
	cb := tw.callback
	tw.mu.Unlock()
	if cb != nil {
		cb()
	}
}
