package confgo

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

var _ Source = (*fakeSource)(nil)

type fakeSource struct {
	data []byte
	err  error
}

func (s *fakeSource) Read() ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

var _ Formatter = (*fakeFormatter)(nil)

type fakeFormatter struct {
	data any
	err  error
}

func (f *fakeFormatter) Unmarshal(_ []byte, v any) error {
	if f.err != nil {
		return f.err
	}

	vVal := reflect.ValueOf(v)
	if vVal.Kind() != reflect.Ptr {
		return fmt.Errorf("v must be a pointer")
	}

	vElem := vVal.Elem()
	dataVal := reflect.ValueOf(f.data)

	if dataVal.Type() != vElem.Type() {
		return fmt.Errorf("type mismatch: expected %v, got %v", vElem.Type(), dataVal.Type())
	}

	vElem.Set(dataVal)
	return nil
}

type testInnerConfig struct {
	Int    int    `json:"int"`
	String string `json:"string"`
}

type TestConfig struct {
	Int      int               `json:"int" env:"INT"`
	IntPtr   *int              `json:"int_ptr"`
	Inner    testInnerConfig   `json:"inner"`
	InnerPtr *testInnerConfig  `json:"inner_ptr"`
	Map      map[string]string `json:"map"`
	Slice    []string          `json:"slice"`
}

func testConfigConstructor() any {
	return &TestConfig{}
}

var _ Merger = (*TestConfigAsMerger)(nil)

type TestConfigAsMerger struct {
	TestConfig
}

func testConfigAsMergerConstructor() any {
	return &TestConfigAsMerger{}
}

func (c *TestConfigAsMerger) Merge(other any) error {
	otherCfg, ok := other.(*TestConfigAsMerger)
	if !ok {
		return fmt.Errorf("error converting other config to *TestConfigAsMerger")
	}
	if c.Int == 123 {
		return fmt.Errorf("test merge error")
	}
	c.Int = otherCfg.Int + 1
	return nil
}

var _ Validator = (*TestConfigAsValidator)(nil)

type TestConfigAsValidator struct {
	TestConfig
}

func testConfigAsValidatorConstructor() any {
	return &TestConfigAsValidator{}
}

func (c *TestConfigAsValidator) Validate() error {
	if c.Int == 123 {
		return fmt.Errorf("test validation error")
	}
	return nil
}

var (
	_ Validator = (*testConfigAsValidatorAndMerger)(nil)
	_ Merger    = (*testConfigAsValidatorAndMerger)(nil)
)

type testConfigAsValidatorAndMerger struct {
	TestConfig
}

func (c *testConfigAsValidatorAndMerger) Validate() error {
	if c.Inner.String == "invalid value" {
		return fmt.Errorf("test error")
	}
	return nil
}

func (c *testConfigAsValidatorAndMerger) Merge(other any) error {
	otherCfg, ok := other.(*testConfigAsValidatorAndMerger)
	if !ok {
		return fmt.Errorf("error converting other config to *testConfigAsValidatorAndMerger")
	}
	c.Int = otherCfg.Int + 1
	return nil
}

type testConfigManagerFields struct {
	constructor     ConstructorFunc
	current         any
	loaders         []Loader
	validators      []ValidateFunc
	namedValidators map[string]ValidateFunc
	// mu            sync.RWMutex
}

func newTestConfigManager(fields testConfigManagerFields) *ConfigManager {
	return &ConfigManager{
		constructor:     fields.constructor,
		current:         fields.current,
		loaders:         fields.loaders,
		validators:      fields.validators,
		namedValidators: fields.namedValidators,
	}
}

func TestConfigManager_merge(t *testing.T) {
	t.Parallel()

	testConfigStruct := TestConfig{}
	testNonStruct := 1
	testNonConfigStruct := struct{}{}

	type args struct {
		dst any
		src any
	}
	tests := []struct {
		name      string
		fields    testConfigManagerFields
		args      args
		wantError bool
		want      any
	}{
		{
			name: "dst is not a pointer",
			args: args{
				dst: testConfigStruct,
				src: &testConfigStruct,
			},
			wantError: true,
		},
		{
			name: "dst is not a struct",
			args: args{
				dst: &testNonStruct,
				src: &testConfigStruct,
			},
			wantError: true,
		},
		{
			name: "src is not a struct",
			args: args{
				dst: &testConfigStruct,
				src: &testNonStruct,
			},
			wantError: true,
		},
		{
			name: "src dst have different type",
			args: args{
				dst: &testConfigStruct,
				src: &testNonConfigStruct,
			},
			wantError: true,
		},
		{
			name: "src is not a pointer",
			args: args{
				dst: &TestConfig{
					Int: 1,
				},
				src: TestConfig{
					Int: 2,
				},
			},
			want: &TestConfig{
				Int: 2,
			},
		},
		{
			name: "single field override",
			args: args{
				dst: &TestConfig{
					Int: 1,
				},
				src: &TestConfig{
					Int: 2,
				},
			},
			want: &TestConfig{
				Int: 2,
			},
		},
		{
			name: "no override by zero value",
			args: args{
				dst: &TestConfig{
					Int:    1,
					IntPtr: ptr(123),
				},
				src: &TestConfig{
					Int: 2,
				},
			},
			want: &TestConfig{
				Int:    2,
				IntPtr: ptr(123),
			},
		},
		{
			name: "zero value field override",
			args: args{
				dst: &TestConfig{},
				src: &TestConfig{
					Int: 2,
				},
			},
			want: &TestConfig{
				Int: 2,
			},
		},
		{
			name: "multiple fields override",
			args: args{
				dst: &TestConfig{
					Int:    1,
					IntPtr: ptr(123),
				},
				src: &TestConfig{
					Int:    2,
					IntPtr: ptr(321),
				},
			},
			want: &TestConfig{
				Int:    2,
				IntPtr: ptr(321),
			},
		},
		{
			name: "inner struct custom merge",
			args: args{
				dst: &TestConfig{
					Inner: testInnerConfig{
						Int:    1,
						String: "str",
					},
				},
				src: &TestConfig{
					Inner: testInnerConfig{
						Int: 2,
					},
				},
			},
			want: &TestConfig{
				Inner: testInnerConfig{
					Int:    2,
					String: "str",
				},
			},
		},
		{
			name: "inner struct pointer custom merge",
			args: args{
				dst: &TestConfig{
					InnerPtr: &testInnerConfig{
						Int:    1,
						String: "str",
					},
				},
				src: &TestConfig{
					InnerPtr: &testInnerConfig{
						Int: 2,
					},
				},
			},
			want: &TestConfig{
				InnerPtr: &testInnerConfig{
					Int:    2,
					String: "str",
				},
			},
		},
		{
			name: "override inner struct nil pointer",
			args: args{
				dst: &TestConfig{
					InnerPtr: nil,
				},
				src: &TestConfig{
					InnerPtr: &testInnerConfig{
						Int: 2,
					},
				},
			},
			want: &TestConfig{
				InnerPtr: &testInnerConfig{
					Int: 2,
				},
			},
		},
		{
			name: "override inner map",
			args: args{
				dst: &TestConfig{
					Map: map[string]string{"foo": "bar", "the_one": "to_replace"},
				},
				src: &TestConfig{
					Map: map[string]string{"the_one": "with_updated_value"},
				},
			},
			want: &TestConfig{
				Map: map[string]string{"foo": "bar", "the_one": "with_updated_value"},
			},
		},
		{
			name: "no override by zero map",
			args: args{
				dst: &TestConfig{
					Map: map[string]string{"foo": "bar", "the_one": "to_replace"},
				},
				src: &TestConfig{
					Map: nil,
				},
			},
			want: &TestConfig{
				Map: map[string]string{"foo": "bar", "the_one": "to_replace"},
			},
		},
		{
			name: "override inner slice",
			args: args{
				dst: &TestConfig{
					Slice: []string{"first", "second"},
				},
				src: &TestConfig{
					Slice: []string{"third"},
				},
			},
			want: &TestConfig{
				Slice: []string{"third"},
			},
		},
		{
			name: "no override by zero slice",
			args: args{
				dst: &TestConfig{
					Slice: []string{"first", "second"},
				},
				src: &TestConfig{
					Slice: nil,
				},
			},
			want: &TestConfig{
				Slice: []string{"first", "second"},
			},
		},
		{
			name: "Merger config",
			args: args{
				dst: &TestConfigAsMerger{},
				src: &TestConfigAsMerger{TestConfig{Int: 1}},
			},
			want: &TestConfigAsMerger{TestConfig{Int: 2}},
		},
		{
			name: "Merger config with error",
			args: args{
				dst: &TestConfigAsMerger{TestConfig{Int: 123}}, // here we emulate merge error
				src: &TestConfigAsMerger{TestConfig{Int: 122}},
			},
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := newTestConfigManager(tt.fields)
			gotErr := cm.merge(tt.args.dst, tt.args.src)
			if tt.wantError {
				if gotErr == nil {
					t.Errorf("Expected error, got nil instead")
				}
			} else if !reflect.DeepEqual(tt.args.dst, tt.want) {
				t.Errorf("Merged struct is invalid:\n  want: %#v\n  got: %#v", tt.want, tt.args.dst)
			}
		})
	}
}

func TestConfigManager_validate(t *testing.T) {
	t.Parallel()

	type args struct {
		config any
	}
	tests := []struct {
		name      string
		fields    testConfigManagerFields
		args      args
		wantError bool
	}{
		{
			name: "non validator config",
			args: args{
				config: &TestConfig{Int: 123},
			},
			wantError: false,
		},
		{
			name: "validator config",
			args: args{
				config: &TestConfigAsValidator{TestConfig{Int: 123}},
			},
			wantError: true,
		},
		{
			name: "with custom validator",
			fields: testConfigManagerFields{
				validators: []ValidateFunc{func() error {
					return fmt.Errorf("test")
				}},
			},
			args: args{
				config: &TestConfig{Int: 123},
			},
			wantError: true,
		},
		{
			name: "with custom named validator",
			fields: testConfigManagerFields{
				namedValidators: map[string]ValidateFunc{"test": func() error {
					return fmt.Errorf("test")
				}},
			},
			args: args{
				config: &TestConfig{Int: 123},
			},

			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := newTestConfigManager(tt.fields)
			gotErr := cm.validate(tt.args.config)
			if tt.wantError {
				if gotErr == nil {
					t.Errorf("Expected error, got nil instead")
				}
			}
		})
	}
}

func TestConfigManager_reload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		fields     testConfigManagerFields
		wantErr    bool
		wantConfig any
	}{
		{
			name: "multiple loaders success",
			fields: testConfigManagerFields{
				constructor: func() any { return new(TestConfig) },
				loaders: []Loader{
					{Source: &fakeSource{data: []byte(`{"int": 1}`)}, Formatter: NewJSONFormatter()},
					{Source: &fakeSource{data: []byte(`{"inner": {"string": "str"}}`)}, Formatter: NewJSONFormatter()},
				},
			},
			wantConfig: &TestConfig{Int: 1, Inner: testInnerConfig{String: "str"}},
		},
		{
			name: "read error",
			fields: testConfigManagerFields{
				constructor: func() any { return new(TestConfig) },
				loaders: []Loader{
					{Source: &fakeSource{err: fmt.Errorf("test error")}, Formatter: NewJSONFormatter()},
				},
			},
			wantErr: true,
		},
		{
			name: "unmarshal error",
			fields: testConfigManagerFields{
				constructor: func() any { return new(TestConfig) },
				loaders: []Loader{
					{Source: &fakeSource{data: []byte(`{"int": 1}`)}, Formatter: &fakeFormatter{err: fmt.Errorf("test error")}},
				},
			},
			wantErr: true,
		},
		{
			name: "validate error",
			fields: testConfigManagerFields{
				constructor: func() any { return new(TestConfig) },
				loaders: []Loader{
					{Source: &fakeSource{data: []byte(`{"int": 1}`)}, Formatter: NewJSONFormatter()},
				},
				validators: []ValidateFunc{func() error { return fmt.Errorf("test error") }},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := newTestConfigManager(tt.fields)
			if err := cm.reload(); (err != nil) != tt.wantErr {
				t.Fatalf("reload() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr {
				return
			}

			got := cm.Config()
			if !reflect.DeepEqual(got, tt.wantConfig) {
				t.Fatalf("current config mismatch:\n  got:  %#v\n  want: %#v", got, tt.wantConfig)
			}
		})
	}
}

func TestConfigManager_validatePreRunState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		fields  testConfigManagerFields
		wantErr bool
	}{
		{
			name: "constructor is nil",
			fields: testConfigManagerFields{
				constructor: nil,
			},
			wantErr: true,
		},
		{
			name: "constructor returns non ptr value",
			fields: testConfigManagerFields{
				constructor: func() any { return TestConfig{} },
			},
			wantErr: true,
		},
		{
			name: "constructor returns ptr to non struct",
			fields: testConfigManagerFields{
				constructor: func() any { return ptr(1) },
			},
			wantErr: true,
		},
		{
			name: "constructor returns non zero struct",
			fields: testConfigManagerFields{
				constructor: func() any { return &TestConfig{Int: 1, Inner: testInnerConfig{String: "test"}} },
			},
			wantErr: true,
		},
		{
			name: "positional validator is nil",
			fields: testConfigManagerFields{
				constructor: testConfigConstructor,
				validators:  []ValidateFunc{nil},
			},
			wantErr: true,
		},
		{
			name: "named validator is nil",
			fields: testConfigManagerFields{
				constructor:     testConfigConstructor,
				namedValidators: map[string]ValidateFunc{"test": nil},
			},
			wantErr: true,
		},
		{
			name: "no loaders configured",
			fields: testConfigManagerFields{
				constructor: testConfigConstructor,
				loaders:     []Loader{},
			},
			wantErr: true,
		},
		{
			name: "loader with nil source",
			fields: testConfigManagerFields{
				constructor: testConfigConstructor,
				loaders:     []Loader{{Source: nil}},
			},
			wantErr: true,
		},
		{
			name: "loader with nil formatter",
			fields: testConfigManagerFields{
				constructor: testConfigConstructor,
				loaders:     []Loader{{Source: &fakeSource{}, Formatter: nil}},
			},
			wantErr: true,
		},
		{
			name: "valid",
			fields: testConfigManagerFields{
				constructor:     testConfigConstructor,
				loaders:         []Loader{{Source: &fakeSource{}, Formatter: &fakeFormatter{}}},
				validators:      []ValidateFunc{func() error { return nil }},
				namedValidators: map[string]ValidateFunc{"test": func() error { return nil }},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cm := newTestConfigManager(tt.fields)
			if err := cm.validatePreRunState(); (err != nil) != tt.wantErr {
				t.Errorf("validatePreRunState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigManager_runWatchers_RegisterOnlyNonNilWatchers(t *testing.T) {
	t.Parallel()

	events := make(chan string, 3)

	watcher1 := NewTriggerWatcher()
	watcher2 := NewTriggerWatcher()

	cm := newTestConfigManager(testConfigManagerFields{
		constructor: testConfigConstructor,
		loaders: []Loader{
			{
				Source:    &fakeSource{data: []byte("test")},
				Formatter: &fakeFormatter{data: TestConfig{Int: 1}},
				Watcher:   watcher1,
				OnUpdateSuccess: func() {
					events <- "A:success"
				},
				OnUpdateError: func(_ error) {
					events <- "A:error"
				},
			},
			{
				Source:    &fakeSource{data: []byte("test")},
				Formatter: &fakeFormatter{data: TestConfig{Int: 1}},
				Watcher:   watcher2,
				OnUpdateSuccess: func() {
					events <- "B:success"
				},
				OnUpdateError: func(_ error) {
					events <- "B:error"
				},
			},
			{
				Source:    &fakeSource{data: []byte("test")},
				Formatter: &fakeFormatter{data: TestConfig{Int: 1}},
				Watcher:   nil, // must be ignored
				OnUpdateSuccess: func() {
					events <- "C:success"
				},
				OnUpdateError: func(_ error) {
					events <- "C:error"
				},
			},
		},
	})

	cm.runWatchers()

	if watcher1.callback == nil {
		t.Fatalf("watcher #1 did not get a callback")
	}
	if watcher2.callback == nil {
		t.Fatalf("watcher #2 did not get a callback")
	}

	watcher1.Trigger()
	got := <-events
	if !strings.HasPrefix(got, "A:") {
		t.Fatalf("expected event from loader A, got %q", got)
	}

	watcher2.Trigger()
	got = <-events
	if !strings.HasPrefix(got, "B:") {
		t.Fatalf("expected event from loader B, got %q", got)
	}

	select {
	case extra := <-events:
		t.Fatalf("did not expect extra event, got %q", extra)
	default:
		// ok
	}
}

func TestConfigManager_runWatchers_CallbackMayBeTriggeredMultipleTimes(t *testing.T) {
	t.Parallel()

	events := make(chan string, 2)

	watcher := NewTriggerWatcher()
	cm := &ConfigManager{
		constructor: testConfigConstructor,
		loaders: []Loader{
			{
				Source:    &fakeSource{data: []byte("test")},
				Formatter: &fakeFormatter{data: TestConfig{Int: 1}},
				Watcher:   watcher,
				OnUpdateSuccess: func() {
					events <- "X:success"
				},
				OnUpdateError: func(_ error) {
					events <- "X:error"
				},
			},
		},
	}

	cm.runWatchers()

	if watcher.callback == nil {
		t.Fatalf("watcher did not get a callback")
	}

	watcher.Trigger()
	watcher.Trigger()

	for i := range 2 {
		select {
		case got := <-events:
			if !strings.HasPrefix(got, "X:") {
				t.Fatalf("expected event from loader X, got %q", got)
			}
		default:
			t.Fatalf("expected event #%d, but channel is empty", i+1)
		}
	}
}

func TestConfigManager_runWatchers_NoPanicsIfCallbacksNil(t *testing.T) {
	t.Parallel()

	watcher := NewTriggerWatcher()
	cm := &ConfigManager{
		constructor: testConfigConstructor,
		loaders: []Loader{
			{
				Source:          &fakeSource{data: []byte("test")},
				Formatter:       &fakeFormatter{data: TestConfig{Int: 1}},
				Watcher:         watcher,
				OnUpdateSuccess: nil,
				OnUpdateError:   nil,
			},
		},
	}

	cm.runWatchers()

	if watcher.callback == nil {
		t.Fatalf("watcher did not get a callback")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("callback panicked: %v", r)
		}
	}()
	watcher.Trigger()
}

func TestConfigManager_New(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options []Option
		wantErr bool
	}{
		{
			name:    "no options",
			options: []Option{},
			wantErr: false,
		},
		{
			name:    "nil options",
			options: []Option{},
			wantErr: false,
		},
		{
			name:    "with validator",
			options: []Option{WithValidator(func() error { return fmt.Errorf("test error") })},
			wantErr: false,
		},
		{
			name:    "with named validator",
			options: []Option{WithNamedValidator("test", func() error { return fmt.Errorf("test error") })},
			wantErr: false,
		},
		{
			name:    "with env",
			options: []Option{WithEnv},
			wantErr: false,
		},
		{
			name:    "with json file",
			options: []Option{WithJSONFile("test_file.json", nil)},
			wantErr: false,
		},
		{
			name:    "with dynamic json file",
			options: []Option{WithDynamicJSONFile("test_file.json", nil, nil, nil)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := NewConfigManagerFor[TestConfig](tt.options...); (err != nil) != tt.wantErr {
				t.Fatalf("NewConfigManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//nolint:cyclop
func TestConfigManager_Start_Static(t *testing.T) {
	type args struct {
		constructor ConstructorFunc
		options     []Option
	}
	tests := []struct {
		name    string
		args    args
		setup   func(t *testing.T)
		wantErr bool
		want    any
	}{
		{
			name: "no options",
			args: args{
				constructor: testConfigConstructor,
				options:     []Option{},
			},
			wantErr: true,
		},
		{
			name: "json file does not exist",
			args: args{
				constructor: testConfigConstructor,
				options:     []Option{WithJSONFile("test_config.json")},
			},
			wantErr: true,
		},
		{
			name: "json file exits",
			args: args{
				constructor: testConfigConstructor,
				options:     []Option{WithJSONFile("test_config.json")},
			},
			setup: func(t *testing.T) {
				t.Helper()
				cleanup, err := setupJSONConfig("test_config.json", map[string]any{"int": 123})
				if err != nil {
					t.Fatalf("failed to setup json config: %v", err)
				}
				t.Cleanup(cleanup)
			},
			wantErr: false,
			want:    &TestConfig{Int: 123},
		},
		{
			name: "with env",
			args: args{
				constructor: testConfigConstructor,
				options:     []Option{WithEnv},
			},
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("INT", "123")
			},
			wantErr: false,
			want:    &TestConfig{Int: 123},
		},
		{
			name: "with multiple loaders",
			args: args{
				constructor: testConfigConstructor,
				options:     []Option{WithJSONFile("test_config.json"), WithEnv},
			},
			setup: func(t *testing.T) {
				t.Helper()
				cleanup, err := setupJSONConfig("test_config.json", map[string]any{"int": 10})
				if err != nil {
					t.Fatalf("failed to setup json config: %v", err)
				}
				t.Cleanup(cleanup)
				t.Setenv("INT", "1")
			},
			wantErr: false,
			want:    &TestConfig{Int: 1},
		},
		{
			name: "with multiple loaders and custom merge",
			args: args{
				constructor: testConfigAsMergerConstructor,
				options:     []Option{WithJSONFile("test_config.json"), WithEnv},
			},
			setup: func(t *testing.T) {
				t.Helper()
				cleanup, err := setupJSONConfig("test_config.json", map[string]any{"int": 10})
				if err != nil {
					t.Fatalf("failed to setup json config: %v", err)
				}
				t.Cleanup(cleanup)
				t.Setenv("INT", "1")
			},
			wantErr: false,
			want:    &TestConfigAsMerger{TestConfig{Int: 2}},
		},
		{
			name: "with multiple loaders and custom merge error",
			args: args{
				constructor: testConfigAsMergerConstructor,
				options:     []Option{WithJSONFile("test_config.json"), WithEnv},
			},
			setup: func(t *testing.T) {
				t.Helper()
				cleanup, err := setupJSONConfig("test_config.json", map[string]any{"int": 122})
				if err != nil {
					t.Fatalf("failed to setup json config: %v", err)
				}
				t.Cleanup(cleanup)
				t.Setenv("INT", "1")
			},
			wantErr: true,
			want:    &TestConfigAsMerger{TestConfig{Int: 2}},
		},
		{
			name: "with config validation success",
			args: args{
				constructor: testConfigAsValidatorConstructor,
				options:     []Option{WithEnv},
			},
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("INT", "122")
			},
			wantErr: false,
			want:    &TestConfigAsValidator{TestConfig{Int: 122}},
		},
		{
			name: "with config validation error",
			args: args{
				constructor: testConfigAsValidatorConstructor,
				options:     []Option{WithEnv},
			},
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("INT", "123")
			},
			wantErr: true,
		},
		{
			name: "with custom validator success",
			args: args{
				constructor: testConfigConstructor,
				options:     []Option{WithEnv, WithValidator(func() error { return nil })},
			},
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("INT", "123")
			},
			wantErr: false,
			want:    &TestConfig{Int: 123},
		},
		{
			name: "with custom validator error",
			args: args{
				constructor: testConfigConstructor,
				options:     []Option{WithEnv, WithValidator(func() error { return fmt.Errorf("error") })},
			},
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("INT", "123")
			},
			wantErr: true,
		},
		{
			name: "with custom named validator success",
			args: args{
				constructor: testConfigConstructor,
				options:     []Option{WithEnv, WithNamedValidator("test", func() error { return nil })},
			},
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("INT", "123")
			},
			wantErr: false,
			want:    &TestConfig{Int: 123},
		},
		{
			name: "with custom validator error",
			args: args{
				constructor: testConfigConstructor,
				options:     []Option{WithEnv, WithNamedValidator("test", func() error { return fmt.Errorf("error") })},
			},
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("INT", "123")
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}

			cm, err := NewConfigManager(tt.args.constructor, tt.args.options...)
			if err != nil {
				t.Fatalf("NewConfigManager() error = %v, wantErr %v", err, false)
			}
			defer func() {
				if err := cm.Stop(); err != nil {
					t.Fatalf("Stop() error = %v, wantErr %v", err, false)
				}
			}()
			if err := cm.Start(); (err != nil) != tt.wantErr {
				t.Fatalf("Start() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr {
				return
			}

			cfg := cm.Config()
			if !reflect.DeepEqual(cfg, tt.want) {
				t.Fatalf("Start() got = %v, want %v", cfg, tt.want)
			}
		})
	}
}

//nolint:cyclop
func TestConfigManager_Start_DynamicUpdate(t *testing.T) {
	testFile := "test_dynamic_config.json"

	cleanup, err := setupJSONConfig(testFile, map[string]any{"int": 10})
	if err != nil {
		t.Fatalf("failed to setup json config: %v", err)
	}
	t.Cleanup(cleanup)

	cm, err := NewConfigManagerFor[TestConfig]()
	if err != nil {
		t.Fatalf("NewConfigManagerFor[TestConfig]() error = %v, wantErr %v", err, false)
	}

	var successCalled, errorCalled bool
	watcher := NewTriggerWatcher()
	cm.AddLoader(Loader{
		Source:    NewFileSource(testFile),
		Formatter: NewJSONFormatter(),
		Watcher:   watcher,
		OnUpdateSuccess: func() {
			successCalled = true
		},
		OnUpdateError: func(_ error) {
			errorCalled = true
		},
	})

	defer func() {
		if err := cm.Stop(); err != nil {
			t.Fatalf("Stop() error = %v, wantErr %v", err, false)
		}
	}()
	if err := cm.Start(); err != nil {
		t.Fatalf("Start() error = %v, wantErr %v", err, false)
	}

	cfg1 := cm.Config()
	want1 := &TestConfig{Int: 10}
	if !reflect.DeepEqual(cfg1, want1) {
		t.Fatalf("Config() got = %v, want %v", cfg1, want1)
	}
	if successCalled {
		t.Fatalf("Unexpected call of OnUpdateSuccess")
	}
	if errorCalled {
		t.Fatalf("Unexpected call of OnUpdateError")
	}

	if err := updateJSONFile(testFile, map[string]any{"int": 20}); err != nil {
		t.Fatalf("failed to update json config: %v", err)
	}
	watcher.Trigger()

	cfg2 := cm.Config()
	want2 := &TestConfig{Int: 20}
	if !reflect.DeepEqual(cfg2, want2) {
		t.Fatalf("Config() got = %v, want %v", cfg2, want2)
	}
	if !successCalled {
		t.Fatalf("OnUpdateSuccess was not called")
	}
	if errorCalled {
		t.Fatalf("Unexpected call of OnUpdateError")
	}
	successCalled = false

	cleanup()
	watcher.Trigger()

	cfg3 := cm.Config()
	if !reflect.DeepEqual(cfg3, want2) {
		t.Fatalf("Config() got = %v, want %v", cfg3, want2)
	}
	if successCalled {
		t.Fatalf("Unexpected call of OnUpdateSuccess")
	}
	if !errorCalled {
		t.Fatalf("OnUpdateError was not called")
	}
}
