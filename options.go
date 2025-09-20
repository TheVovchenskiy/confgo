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
func WithEnv(cm *ConfigManager) error {
	cm.AddLoader(Loader{
		Source:    NewEnvSource(),
		Formatter: NewEnvFormatter(),
	})
	return nil
}

// WithJSONFile adds a Loader layer with FileSource and JSONFormatter to parse config data from.
func WithJSONFile(file string, jsonFormatterOptions ...JSONFormatterOption) Option {
	return func(cm *ConfigManager) error {
		cm.AddLoader(Loader{
			Source:    NewFileSource(file),
			Formatter: NewJSONFormatter(jsonFormatterOptions...),
		})
		return nil
	}
}

// WithDynamicJSONFile adds a Loader layer with FileSource, JSONFormatter and
// ModTimeWatcher with callbacks to parse and dynamically update config data from.
func WithDynamicJSONFile(
	file string,
	onUpdateSuccess CallbackFunc,
	onUpdateError CallbackErrFunc,
	jsonFormatterOptions ...JSONFormatterOption,
) Option {
	return func(cm *ConfigManager) error {
		s := NewFileSource(file)
		cm.AddLoader(Loader{
			Source:          s,
			Formatter:       NewJSONFormatter(jsonFormatterOptions...),
			Watcher:         NewModTimeWatcher(s),
			OnUpdateSuccess: onUpdateSuccess,
			OnUpdateError:   onUpdateError,
		})
		return nil
	}
}

// WithYAMLFile adds a Loader layer with FileSource and YAMLFormatter to parse config data from.
func WithYAMLFile(file string, yamlFormatterOptions ...YAMLFormatterOption) Option {
	return func(cm *ConfigManager) error {
		cm.AddLoader(Loader{
			Source:    NewFileSource(file),
			Formatter: NewYAMLFormatter(yamlFormatterOptions...),
		})
		return nil
	}
}

// WithDynamicYAMLFile adds a Loader layer with FileSource, YAMLFormatter and
// ModTimeWatcher with callbacks to parse and dynamically update config data from.
func WithDynamicYAMLFile(
	file string,
	onUpdateSuccess CallbackFunc,
	onUpdateError CallbackErrFunc,
	yamlFormatterOptions ...YAMLFormatterOption,
) Option {
	return func(cm *ConfigManager) error {
		s := NewFileSource(file)
		cm.AddLoader(Loader{
			Source:          s,
			Formatter:       NewYAMLFormatter(yamlFormatterOptions...),
			Watcher:         NewModTimeWatcher(s),
			OnUpdateSuccess: onUpdateSuccess,
			OnUpdateError:   onUpdateError,
		})
		return nil
	}
}
