package confgo

// WithValidator adds a custom validator which will be called on each config load.
func WithValidator(v ValidateFunc) Option {
	return func(cm *ConfigManager) error {
		cm.validators = append(cm.validators, v)
		return nil
	}
}

// WithNamedValidator adds a custom named validator which will be called on each config load.
func WithNamedValidator(name string, v ValidateFunc) Option {
	return func(cm *ConfigManager) error {
		cm.namedValidators[name] = v
		return nil
	}
}

// WithEnv adds a Loader layer with EnvSource and EnvFormatter to parse config data from.
func WithEnv() Option {
	return func(cm *ConfigManager) error {
		cm.AddLoader(Loader{
			Source:    NewEnvSource(),
			Formatter: NewEnvFormatter(),
		})
		return nil
	}
}

// WithJSONFile adds a Loader layer with FileSource and JSONFormatter to parse config data from.
func WithJSONFile(file string) Option {
	return func(cm *ConfigManager) error {
		cm.AddLoader(Loader{
			Source:    NewFileSource(file),
			Formatter: NewJSONFormatter(),
		})
		return nil
	}
}

// WithDynamicJSONFile adds a Loader layer with FileSource, JSONFormatter and
// ModTimeWatcher with callbacks to parse and dynamically update config data from.
// Callbacks might be nil.
func WithDynamicJSONFile(file string, onUpdateSuccess CallbackFunc, onUpdateError CallbackErrFunc) Option {
	return func(cm *ConfigManager) error {
		s := NewFileSource(file)
		cm.AddLoader(Loader{
			Source:          s,
			Formatter:       NewJSONFormatter(),
			Watcher:         NewModTimeWatcher(s),
			OnUpdateSuccess: onUpdateSuccess,
			OnUpdateError:   onUpdateError,
		})
		return nil
	}
}
