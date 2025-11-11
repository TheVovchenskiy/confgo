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

// WithLoader adds a custom Loader layer.
func WithLoader(l *Loader) Option {
	return func(cm *ConfigManager) error {
		cm.AddLoader(l)
		return nil
	}
}

// WithEnv adds a Loader layer which handles env variable configurations.
func WithEnv(cm *ConfigManager) error {
	cm.AddLoader(NewLoader(NewEnvSource(), NewEnvFormatter()))
	return nil
}

// WithJSONFile adds a Loader layer which handles json file configurations.
func WithJSONFile(file string, jsonFormatterOptions ...JSONFormatterOption) Option {
	return func(cm *ConfigManager) error {
		cm.AddLoader(NewLoader(NewFileSource(file), NewJSONFormatter(jsonFormatterOptions...)))
		return nil
	}
}

// WithDynamicJSONFile adds a Loader layer which handles json file configurations and tracks all its changes.
func WithDynamicJSONFile(
	file string,
	onUpdateSuccess CallbackFunc,
	onUpdateError CallbackErrFunc,
	jsonFormatterOptions ...JSONFormatterOption,
) Option {
	return func(cm *ConfigManager) error {
		s := NewFileSource(file)
		cm.AddLoader(NewLoader(s, NewJSONFormatter(jsonFormatterOptions...),
			WithWatcher(NewModTimeWatcher(s)),
			WithOnUpdateSuccess(onUpdateSuccess),
			WithOnUpdateError(onUpdateError),
		))

		return nil
	}
}

// WithYAMLFile adds a Loader layer which handles yaml file configurations.
func WithYAMLFile(file string, yamlFormatterOptions ...YAMLFormatterOption) Option {
	return func(cm *ConfigManager) error {
		cm.AddLoader(NewLoader(NewFileSource(file), NewYAMLFormatter(yamlFormatterOptions...)))
		return nil
	}
}

// WithDynamicYAMLFile adds a Loader layer which handles yaml file configurations and tracks all its changes.
func WithDynamicYAMLFile(
	file string,
	onUpdateSuccess CallbackFunc,
	onUpdateError CallbackErrFunc,
	yamlFormatterOptions ...YAMLFormatterOption,
) Option {
	return func(cm *ConfigManager) error {
		s := NewFileSource(file)
		cm.AddLoader(NewLoader(NewFileSource(file), NewYAMLFormatter(yamlFormatterOptions...),
			WithWatcher(NewModTimeWatcher(s)),
			WithOnUpdateSuccess(onUpdateSuccess),
			WithOnUpdateError(onUpdateError),
		))

		return nil
	}
}
