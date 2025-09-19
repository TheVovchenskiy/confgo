package confgo

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
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

var _ Watcher = (*fakeWatcher)(nil)

type fakeWatcher struct {
	mu sync.Mutex
	cb func()
}

func (fw *fakeWatcher) Watch(cb func()) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.cb = cb
}

func (fw *fakeWatcher) Stop() error { return nil }

func (fw *fakeWatcher) Trigger() {
	fw.mu.Lock()
	cb := fw.cb
	fw.mu.Unlock()
	if cb != nil {
		cb()
	}
}

type testInnerConfig struct {
	Int    int    `json:"int"`
	String string `json:"string"`
}

type testConfig struct {
	Int      int               `json:"int"`
	IntPtr   *int              `json:"int_ptr"`
	Inner    testInnerConfig   `json:"inner"`
	InnerPtr *testInnerConfig  `json:"inner_ptr"`
	Map      map[string]string `json:"map"`
	Slice    []string          `json:"slice"`
}

var _ Merger = (*testConfigAsMerger)(nil)

type testConfigAsMerger struct {
	testConfig
}

func (c *testConfigAsMerger) Merge(other any) error {
	otherCfg, ok := other.(*testConfigAsMerger)
	if !ok {
		return fmt.Errorf("error converting other config to *testConfigAsMerger")
	}
	c.Int = otherCfg.Int + 1
	return nil
}

var _ Validator = (*testConfigAsValidator)(nil)

type testConfigAsValidator struct {
	testConfig
}

func (c *testConfigAsValidator) Validate() error {
	if c.Int == 123 {
		return fmt.Errorf("test error")
	}
	return nil
}

type testConfigManagerFields struct {
	constructor     ConstructorFunc
	current         any
	loaders         []Loader
	validators      []ValidateFunc
	namedValidators map[string]ValidateFunc
	//mu            sync.RWMutex
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
	testConfigStruct := testConfig{}
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
				dst: &testConfig{
					Int: 1,
				},
				src: testConfig{
					Int: 2,
				},
			},
			want: &testConfig{
				Int: 2,
			},
		},
		{
			name: "single field override",
			args: args{
				dst: &testConfig{
					Int: 1,
				},
				src: &testConfig{
					Int: 2,
				},
			},
			want: &testConfig{
				Int: 2,
			},
		},
		{
			name: "no override by zero value",
			args: args{
				dst: &testConfig{
					Int:    1,
					IntPtr: ptr(123),
				},
				src: &testConfig{
					Int: 2,
				},
			},
			want: &testConfig{
				Int:    2,
				IntPtr: ptr(123),
			},
		},
		{
			name: "zero value field override",
			args: args{
				dst: &testConfig{},
				src: &testConfig{
					Int: 2,
				},
			},
			want: &testConfig{
				Int: 2,
			},
		},
		{
			name: "multiple fields override",
			args: args{
				dst: &testConfig{
					Int:    1,
					IntPtr: ptr(123),
				},
				src: &testConfig{
					Int:    2,
					IntPtr: ptr(321),
				},
			},
			want: &testConfig{
				Int:    2,
				IntPtr: ptr(321),
			},
		},
		{
			name: "inner struct custom merge",
			args: args{
				dst: &testConfig{
					Inner: testInnerConfig{
						Int:    1,
						String: "str",
					},
				},
				src: &testConfig{
					Inner: testInnerConfig{
						Int: 2,
					},
				},
			},
			want: &testConfig{
				Inner: testInnerConfig{
					Int:    2,
					String: "str",
				},
			},
		},
		{
			name: "inner struct pointer custom merge",
			args: args{
				dst: &testConfig{
					InnerPtr: &testInnerConfig{
						Int:    1,
						String: "str",
					},
				},
				src: &testConfig{
					InnerPtr: &testInnerConfig{
						Int: 2,
					},
				},
			},
			want: &testConfig{
				InnerPtr: &testInnerConfig{
					Int:    2,
					String: "str",
				},
			},
		},
		{
			name: "override inner struct nil pointer",
			args: args{
				dst: &testConfig{
					InnerPtr: nil,
				},
				src: &testConfig{
					InnerPtr: &testInnerConfig{
						Int: 2,
					},
				},
			},
			want: &testConfig{
				InnerPtr: &testInnerConfig{
					Int: 2,
				},
			},
		},
		{
			name: "override inner map",
			args: args{
				dst: &testConfig{
					Map: map[string]string{"foo": "bar", "the_one": "to_replace"},
				},
				src: &testConfig{
					Map: map[string]string{"the_one": "with_updated_value"},
				},
			},
			want: &testConfig{
				Map: map[string]string{"foo": "bar", "the_one": "with_updated_value"},
			},
		},
		{
			name: "no override by zero map",
			args: args{
				dst: &testConfig{
					Map: map[string]string{"foo": "bar", "the_one": "to_replace"},
				},
				src: &testConfig{
					Map: nil,
				},
			},
			want: &testConfig{
				Map: map[string]string{"foo": "bar", "the_one": "to_replace"},
			},
		},
		{
			name: "override inner slice",
			args: args{
				dst: &testConfig{
					Slice: []string{"first", "second"},
				},
				src: &testConfig{
					Slice: []string{"third"},
				},
			},
			want: &testConfig{
				Slice: []string{"third"},
			},
		},
		{
			name: "no override by zero slice",
			args: args{
				dst: &testConfig{
					Slice: []string{"first", "second"},
				},
				src: &testConfig{
					Slice: nil,
				},
			},
			want: &testConfig{
				Slice: []string{"first", "second"},
			},
		},
		{
			name: "Merger config",
			args: args{
				dst: &testConfigAsMerger{},
				src: &testConfigAsMerger{testConfig{Int: 1}},
			},
			want: &testConfigAsMerger{testConfig{Int: 2}},
		},
		{
			name: "Merger config with error",
			args: args{
				dst: &testConfigAsMerger{},
				// ConfigManager implementation ensures that merge is called with the same type dst and src,
				// here we just emulate error returning behaviour
				src: &testConfig{Int: 1},
			},
			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				config: &testConfig{Int: 123},
			},
			wantError: false,
		},
		{
			name: "validator config",
			args: args{
				config: &testConfigAsValidator{testConfig{Int: 123}},
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
				config: &testConfig{Int: 123},
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
				config: &testConfig{Int: 123},
			},

			wantError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	tests := []struct {
		name       string
		fields     testConfigManagerFields
		wantErr    bool
		wantConfig any
	}{
		{
			name: "multiple loaders success",
			fields: testConfigManagerFields{
				constructor: func() any { return new(testConfig) },
				loaders: []Loader{
					{Source: &fakeSource{data: []byte(`{"int": 1}`)}, Formatter: NewJSONFormatter()},
					{Source: &fakeSource{data: []byte(`{"inner": {"string": "str"}}`)}, Formatter: NewJSONFormatter()},
				},
			},
			wantConfig: &testConfig{Int: 1, Inner: testInnerConfig{String: "str"}},
		},
		{
			name: "read error",
			fields: testConfigManagerFields{
				constructor: func() any { return new(testConfig) },
				loaders: []Loader{
					{Source: &fakeSource{err: fmt.Errorf("test error")}, Formatter: NewJSONFormatter()},
				},
			},
			wantErr: true,
		},
		{
			name: "unmarshal error",
			fields: testConfigManagerFields{
				constructor: func() any { return new(testConfig) },
				loaders: []Loader{
					{Source: &fakeSource{data: []byte(`{"int": 1}`)}, Formatter: &fakeFormatter{err: fmt.Errorf("test error")}},
				},
			},
			wantErr: true,
		},
		{
			name: "validate error",
			fields: testConfigManagerFields{
				constructor: func() any { return new(testConfig) },
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
				constructor: func() any { return testConfig{} },
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
				constructor: func() any { return &testConfig{Int: 1, Inner: testInnerConfig{String: "test"}} },
			},
			wantErr: true,
		},
		{
			name: "positional validator is nil",
			fields: testConfigManagerFields{
				constructor: func() any { return &testConfig{} },
				validators:  []ValidateFunc{nil},
			},
			wantErr: true,
		},
		{
			name: "named validator is nil",
			fields: testConfigManagerFields{
				constructor:     func() any { return &testConfig{} },
				namedValidators: map[string]ValidateFunc{"test": nil},
			},
			wantErr: true,
		},
		{
			name: "no loaders configured",
			fields: testConfigManagerFields{
				constructor: func() any { return &testConfig{} },
				loaders:     []Loader{},
			},
			wantErr: true,
		},
		{
			name: "loader with nil source",
			fields: testConfigManagerFields{
				constructor: func() any { return &testConfig{} },
				loaders:     []Loader{{Source: nil}},
			},
			wantErr: true,
		},
		{
			name: "loader with nil formatter",
			fields: testConfigManagerFields{
				constructor: func() any { return &testConfig{} },
				loaders:     []Loader{{Source: &fakeSource{}, Formatter: nil}},
			},
			wantErr: true,
		},
		{
			name: "valid",
			fields: testConfigManagerFields{
				constructor:     func() any { return &testConfig{} },
				loaders:         []Loader{{Source: &fakeSource{}, Formatter: &fakeFormatter{}}},
				validators:      []ValidateFunc{func() error { return nil }},
				namedValidators: map[string]ValidateFunc{"test": func() error { return nil }},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := newTestConfigManager(tt.fields)
			if err := cm.validatePreRunState(); (err != nil) != tt.wantErr {
				t.Errorf("validatePreRunState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigManager_runWatchers_RegisterOnlyNonNilWatchers(t *testing.T) {
	events := make(chan string, 3)

	w1 := &fakeWatcher{}
	w2 := &fakeWatcher{}

	cm := newTestConfigManager(testConfigManagerFields{
		constructor: func() any { return &testConfig{} },
		loaders: []Loader{
			{
				Source:    &fakeSource{data: []byte("test")},
				Formatter: &fakeFormatter{data: testConfig{Int: 1}},
				Watcher:   w1,
				OnUpdateSuccess: func() {
					events <- "A:success"
				},
				OnUpdateError: func(err error) {
					events <- "A:error"
				},
			},
			{
				Source:    &fakeSource{data: []byte("test")},
				Formatter: &fakeFormatter{data: testConfig{Int: 1}},
				Watcher:   w2,
				OnUpdateSuccess: func() {
					events <- "B:success"
				},
				OnUpdateError: func(err error) {
					events <- "B:error"
				},
			},
			{
				Source:    &fakeSource{data: []byte("test")},
				Formatter: &fakeFormatter{data: testConfig{Int: 1}},
				Watcher:   nil, // must be ignored
				OnUpdateSuccess: func() {
					events <- "C:success"
				},
				OnUpdateError: func(err error) {
					events <- "C:error"
				},
			},
		},
	})

	cm.runWatchers()

	if w1.cb == nil {
		t.Fatalf("watcher #1 did not get a callback")
	}
	if w2.cb == nil {
		t.Fatalf("watcher #2 did not get a callback")
	}

	w1.Trigger()
	got := <-events
	if !strings.HasPrefix(got, "A:") {
		t.Fatalf("expected event from loader A, got %q", got)
	}

	w2.Trigger()
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

	w := &fakeWatcher{}
	cm := &ConfigManager{
		constructor: func() any { return &testConfig{} },
		loaders: []Loader{
			{
				Source:    &fakeSource{data: []byte("test")},
				Formatter: &fakeFormatter{data: testConfig{Int: 1}},
				Watcher:   w,
				OnUpdateSuccess: func() {
					events <- "X:success"
				},
				OnUpdateError: func(err error) {
					events <- "X:error"
				},
			},
		},
	}

	cm.runWatchers()

	if w.cb == nil {
		t.Fatalf("watcher did not get a callback")
	}

	w.Trigger()
	w.Trigger()

	for i := 0; i < 2; i++ {
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

	w := &fakeWatcher{}
	cm := &ConfigManager{
		constructor: func() any { return &testConfig{} },
		loaders: []Loader{
			{
				Source:          &fakeSource{data: []byte("test")},
				Formatter:       &fakeFormatter{data: testConfig{Int: 1}},
				Watcher:         w,
				OnUpdateSuccess: nil,
				OnUpdateError:   nil,
			},
		},
	}

	cm.runWatchers()

	if w.cb == nil {
		t.Fatalf("watcher did not get a callback")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("callback panicked: %v", r)
		}
	}()
	w.Trigger()
}
