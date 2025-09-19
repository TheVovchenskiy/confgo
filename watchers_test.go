package confgo

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var _ ModTimer = (*mockModTimer)(nil)

type mockModTimer struct {
	mu        sync.Mutex
	times     []time.Time
	errs      []error
	callCount int
}

func (m *mockModTimer) ModTime() (time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var t time.Time
	if m.callCount >= len(m.times) {
		t = m.times[len(m.times)-1]
	} else {
		t = m.times[m.callCount]
	}
	var err error
	if m.callCount < len(m.errs) && m.errs[m.callCount] != nil {
		err = m.errs[m.callCount]
	}
	m.callCount++
	return t, err
}

func Test_ModTimeWatcher_NoCallbackOnInitialModTime(t *testing.T) {
	now := time.Now()
	mt := &mockModTimer{
		times: []time.Time{now},
	}
	w := NewModTimeWatcher(mt)
	w.interval = 10 * time.Millisecond

	var calls int
	done := make(chan struct{})
	w.Watch(func() {
		calls++
		close(done)
	})

	select {
	case <-done:
		t.Error("unexpected callback occurred ")
	case <-time.After(300 * time.Millisecond):
		// ok
		close(done)
	}
	if err := w.Stop(); err != nil {
		t.Errorf("Unexpected error while stopping watcher: %v", err)
	}
}

func Test_ModTimeWatcher_CallbackOnModTimeIncrease(t *testing.T) {
	mt := &mockModTimer{
		times: []time.Time{
			time.Unix(0, 1),   // first check (lastMod is unknown)
			time.Unix(0, 1),   // no changes
			time.Unix(100, 0), // has changed
			time.Unix(100, 0), // no changes yet again
			time.Unix(102, 0), // has changed
		},
	}
	w := NewModTimeWatcher(mt)
	w.interval = 10 * time.Millisecond

	var calls int
	done := make(chan struct{})
	w.Watch(func() {
		calls++
		if calls == 2 {
			close(done)
		}
	})

	select {
	case <-done:
		// ok
	case <-time.After(300 * time.Millisecond):
		t.Error("callback was not called as expected")
	}
	if err := w.Stop(); err != nil {
		t.Errorf("Unexpected error while stopping watcher: %v", err)
	}
}

func Test_ModTimeWatcher_NoCallbackWhenNoModTimeChange(t *testing.T) {
	now := time.Now()
	mt := &mockModTimer{
		times: []time.Time{
			now, now, now, now, now,
		},
	}
	w := NewModTimeWatcher(mt)
	w.interval = 10 * time.Millisecond

	var calls int
	w.Watch(func() {
		calls++
	})

	time.Sleep(60 * time.Millisecond)
	if err := w.Stop(); err != nil {
		t.Errorf("Unexpected error while stopping watcher: %v", err)
	}
	if calls > 1 {
		t.Errorf("callback was called %d times, expected at most 1", calls)
	}
}

func Test_ModTimeWatcher_IgnoresModTimeErrors(t *testing.T) {
	now := time.Now()
	mt := &mockModTimer{
		times: []time.Time{
			now,
			now,
			now.Add(2 * time.Second),
		},
		errs: []error{
			nil,
			errors.New("fail"),
			nil,
		},
	}
	w := NewModTimeWatcher(mt)
	w.interval = 10 * time.Millisecond

	call := make(chan struct{})
	w.Watch(func() {
		close(call)
	})

	select {
	case <-call:
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Error("callback was not called after error resolved")
	}
	if err := w.Stop(); err != nil {
		t.Errorf("Unexpected error while stopping watcher: %v", err)
	}
}

func Test_ModTimeWatcher_Stop(t *testing.T) {
	mt := &mockModTimer{
		times: []time.Time{time.Now()},
	}
	w := NewModTimeWatcher(mt)
	w.interval = 10 * time.Millisecond

	var calls int
	w.Watch(func() {
		calls++
	})

	err := w.Stop()
	if err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	// Wait to ensure callback is not called
	n := calls
	time.Sleep(30 * time.Millisecond)
	if calls != n {
		t.Errorf("callback called after Stop: before=%d, after=%d", n, calls)
	}
}
