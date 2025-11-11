package confgo

import "fmt"

// Source represents a configuration source that can provide raw data.
type Source interface {
	// Read reads configuration data from the source.
	Read() ([]byte, error)
}

// Formatter converts raw data into structured configuration objects.
type Formatter interface {
	// Unmarshal converts raw data into a structured configuration object.
	Unmarshal(data []byte, v any) error
}

// Watcher monitors configuration sources for changes and notifies when updates occur.
type Watcher interface {
	// Watch starts monitoring for changes and calls the callback when changes are detected.
	// This method must not block execution of the calling function.
	Watch(callback func())
	// Stop halts the monitoring process.
	Stop() error
}

// CallbackFunc is a function called when an operation completes successfully.
type CallbackFunc func()

// CallbackErrFunc is a function called when an operation encounters an error.
type CallbackErrFunc func(err error)

type LoaderOption func(l *Loader)

// WithWatcher adds an optional watcher.
func WithWatcher(w Watcher) LoaderOption {
	return func(l *Loader) {
		l.watcher = w
	}
}

// WithOnUpdateSuccess adds an optional function which will be triggered on successful.
func WithOnUpdateSuccess(callback CallbackFunc) LoaderOption {
	return func(l *Loader) {
		l.onUpdateSuccess = callback
	}
}

func WithOnUpdateError(callback CallbackErrFunc) LoaderOption {
	return func(l *Loader) {
		l.onUpdateError = callback
	}
}

// Loader defines a set of required Source, required Formatter and optional Watcher with callbacks.
type Loader struct {
	source          Source
	formatter       Formatter
	watcher         Watcher
	onUpdateSuccess CallbackFunc
	onUpdateError   CallbackErrFunc
}

func NewLoader(s Source, f Formatter, opts ...LoaderOption) *Loader {
	ldr := &Loader{
		source:    s,
		formatter: f,
	}

	for _, opt := range opts {
		opt(ldr)
	}

	return ldr
}

func (l *Loader) ensureState() error {
	if l.source == nil {
		return ErrSourceIsNil
	}

	if l.formatter == nil {
		return ErrFormatterIsNil
	}

	return nil
}

// Load reads the data from the source and unmarshalls it into passed value.
func (l *Loader) Load(val any) error {
	data, err := l.source.Read()
	if err != nil {
		return fmt.Errorf("read data from source: %w", err)
	}

	if err := l.formatter.Unmarshal(data, val); err != nil {
		return fmt.Errorf("format data: %w", err)
	}

	return nil
}

// RunWatcherIfNeeded starts the watcher if configured.
func (l *Loader) RunWatcherIfNeeded(callback func() error) {
	if l.watcher == nil {
		return
	}

	l.watcher.Watch(func() {
		if err := callback(); err != nil {
			if l.onUpdateError != nil {
				l.onUpdateError(err)
			}

			return
		}

		if l.onUpdateSuccess != nil {
			l.onUpdateSuccess()
		}
	})
}

// StopWatcherIfNeeded stops the running watcher if the one exists.
func (l *Loader) StopWatcherIfNeeded() error {
	if l.watcher == nil {
		return nil
	}

	return l.watcher.Stop()
}
