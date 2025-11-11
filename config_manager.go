package confgo

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"

	"dario.cat/mergo"
)

// ConstructorFunc creates a new instance of a configuration struct.
type ConstructorFunc func() any

// ValidateFunc is a function that validates a configuration.
type ValidateFunc func() error

// Validator defines an interface for validating configuration objects.
// If the config struct implements this interface, then on every config reload Validate method is called.
// Otherwise, no config validation is performed.
type Validator interface {
	// Validate checks if the configuration is valid.
	Validate() error
}

// Merger defines an interface for custom merging configuration objects.
// If the config struct implements this interface, then merging partial configs will occur via the Merge method.
// Otherwise, the configurations will be merged recursively via reflect package.
type Merger interface {
	// Merge merges another configuration object into this one.
	Merge(other any) error
}

// ConfigManager is a main object that manages configurations.
// It handles loading, merging, validating, and watching configuration sources.
// The manager supports multiple loaders that can read from different sources
// (files, environment variables, etc.) and merge them into a single configuration object.
// It also provides validation capabilities through custom validation functions.
// Configuration updates can be watched and automatically reloaded when changes occur.
type ConfigManager struct {
	constructor ConstructorFunc
	loaders     []*Loader

	validators      []ValidateFunc
	namedValidators map[string]ValidateFunc

	isRunning     atomic.Bool
	bootstrapLock sync.Mutex
	// isBootstrapping atomic.Bool

	current any
	mu      sync.RWMutex
}

// Option is a functional option for configuring ConfigManager.
type Option func(cm *ConfigManager) error

// NewConfigManager creates a new configuration manager with the specified constructor and options.
//
// Note that constructor must return pointer to an empty struct.
func NewConfigManager(constructor ConstructorFunc, opts ...Option) (*ConfigManager, error) {
	cm := &ConfigManager{
		constructor:     constructor,
		loaders:         make([]*Loader, 0),
		validators:      make([]ValidateFunc, 0),
		namedValidators: make(map[string]ValidateFunc),
		isRunning:       atomic.Bool{},
		bootstrapLock:   sync.Mutex{},
		current:         nil,
		mu:              sync.RWMutex{},
	}

	for _, opt := range opts {
		if opt != nil {
			if err := opt(cm); err != nil {
				return nil, err
			}
		}
	}

	return cm, nil
}

// NewConfigManagerFor creates a new configuration manager for a specific type T.
// It is the same as NewConfigManager but creates constructor automatically.
func NewConfigManagerFor[T any](opts ...Option) (*ConfigManager, error) {
	return NewConfigManager(func() any { return new(T) }, opts...)
}

func (cm *ConfigManager) ensureConstructor() error {
	if cm.constructor == nil {
		return ErrConstructorIsNil
	}

	cfg := cm.constructor()

	cfgVal := reflect.ValueOf(cfg)
	if cfgVal.Kind() != reflect.Ptr || cfgVal.Elem().Kind() != reflect.Struct {
		return ErrConstructorMustBePointer
	}

	if !cfgVal.Elem().IsZero() {
		return ErrConstructorMustReturnZeroStruct
	}

	return nil
}

func (cm *ConfigManager) ensurePreRunState() error {
	if err := cm.ensureConstructor(); err != nil {
		return fmt.Errorf("ensure constructor: %w", err)
	}

	for name, v := range cm.namedValidators {
		if v == nil {
			delete(cm.namedValidators, name)
		}
	}

	cm.validators = slices.DeleteFunc(cm.validators, func(v ValidateFunc) bool { return v == nil })
	cm.validators = slices.Clip(cm.validators)

	cm.loaders = slices.DeleteFunc(cm.loaders, func(l *Loader) bool { return l == nil })
	cm.loaders = slices.Clip(cm.loaders)

	if len(cm.loaders) == 0 {
		return ErrNoLoadersDefined
	}

	for i, l := range cm.loaders {
		if err := l.ensureState(); err != nil {
			return fmt.Errorf("loader #%d: %w", i, err)
		}
	}

	return nil
}

func (cm *ConfigManager) runWatchers() {
	for _, l := range cm.loaders {
		l.RunWatcherIfNeeded(cm.reload)
	}
}

func (cm *ConfigManager) merge(dst, src any) error {
	if m, ok := dst.(Merger); ok {
		if err := m.Merge(src); err != nil {
			return err
		}

		return nil
	}
	// Do we need to let the user configure mergo options?
	if err := mergo.Merge(dst, src, mergo.WithOverride); err != nil {
		return err
	}

	return nil
}

func (cm *ConfigManager) validate(config any) error {
	if v, ok := config.(Validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	for name, v := range cm.namedValidators {
		if err := v(); err != nil {
			return fmt.Errorf("named validator %q: %w", name, err)
		}
	}

	for i, v := range cm.validators {
		if err := v(); err != nil {
			return fmt.Errorf("validator %d: %w", i, err)
		}
	}

	return nil
}

func (cm *ConfigManager) reload() error {
	// We can probably optimize here by merging only those configs which were updated.
	merged := cm.constructor()
	for _, ldr := range cm.loaders {
		if ldr == nil {
			continue
		}

		temp := cm.constructor()
		if err := ldr.Load(temp); err != nil {
			return fmt.Errorf("load data from loader: %w", err)
		}

		if err := cm.merge(merged, temp); err != nil {
			return fmt.Errorf("merge: %w", err)
		}
	}

	if err := cm.validate(merged); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.current = merged

	return nil
}

// Start initializes and starts the configuration manager.
//
// Returns an error if ConfigManager was not properly configured.
func (cm *ConfigManager) Start() error {
	cm.bootstrapLock.Lock()
	defer cm.bootstrapLock.Unlock()

	if cm.isRunning.Load() {
		return nil
	}

	if err := cm.ensurePreRunState(); err != nil {
		return fmt.Errorf("ensure config manager state: %w", err)
	}

	if err := cm.reload(); err != nil {
		return fmt.Errorf("initial load config: %w", err)
	}

	cm.runWatchers()
	cm.isRunning.Store(true)

	return nil
}

// MustStart same as Start but panics if any error occurs.
func (cm *ConfigManager) MustStart() {
	if err := cm.Start(); err != nil {
		panic(err)
	}
}

// Stop halts the configuration manager and stops all watchers.
func (cm *ConfigManager) Stop() error {
	if !cm.isRunning.Load() {
		return nil
	}
	defer cm.isRunning.Store(false)

	errs := make([]error, len(cm.loaders))
	for i, l := range cm.loaders {
		errs[i] = l.StopWatcherIfNeeded()
	}

	resErr := errors.Join(errs...)
	if resErr != nil {
		return fmt.Errorf("stop running watchers: %w", resErr)
	}

	return nil
}

// MustStop same as Stop but panics if any error occurs.
func (cm *ConfigManager) MustStop() {
	if err := cm.Stop(); err != nil {
		panic(err)
	}
}

// AddLoader adds a new loader to the configuration manager.
func (cm *ConfigManager) AddLoader(l *Loader) {
	if cm.isRunning.Load() {
		return
	}

	cm.loaders = append(cm.loaders, l)
}

// Config returns the current configuration.
func (cm *ConfigManager) Config() any {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.current
}
