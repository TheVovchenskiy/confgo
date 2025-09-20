package confgo

import (
	"reflect"
	"testing"
)

func TestEnvFormatter_parseRawIntoMap(t *testing.T) {
	t.Parallel()

	type args struct {
		raw []byte
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "empty",
			args: args{
				raw: []byte(""),
			},
			want: map[string]string{},
		},
		{
			name: "single",
			args: args{
				raw: []byte("foo=bar"),
			},
			want: map[string]string{"foo": "bar"},
		},
		{
			name: "multiple",
			args: args{
				raw: []byte("foo=bar\nbar=baz"),
			},
			want: map[string]string{"foo": "bar", "bar": "baz"},
		},
		{
			name: "with spaces",
			args: args{
				raw: []byte("foo=bar baz"),
			},
			want: map[string]string{"foo": "bar baz"},
		},
		{
			name: "multiple equal signs",
			args: args{
				raw: []byte("foo=bar=baz"),
			},
			want: map[string]string{"foo": "bar=baz"},
		},
		{
			name: "no equal signs",
			args: args{
				raw: []byte("foo"),
			},
			want: map[string]string{},
		},
		{
			name: "multiple new line signs",
			args: args{
				raw: []byte("\nfoo=bar\n\n\nbar=baz\n"),
			},
			want: map[string]string{"foo": "bar", "bar": "baz"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ef := &EnvFormatter{}
			if got := ef.parseRawIntoMap(tt.args.raw); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRawIntoMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONFormatter_Unmarshal(t *testing.T) {
	type args struct {
		data []byte
		v    any
	}
	tests := []struct {
		name    string
		opts    []JSONFormatterOption
		args    args
		wantErr bool
		want    any
	}{
		{
			name: "empty",
			args: args{
				data: []byte(""),
				v:    &map[string]any{},
			},
			wantErr: true,
		},
		{
			name: "with keys",
			args: args{
				data: []byte(`{"foo": "bar"}`),
				v:    &map[string]any{},
			},
			wantErr: false,
			want:    &map[string]any{"foo": "bar"},
		},
		{
			name: "unknown keys on disallow unknown fields",
			opts: []JSONFormatterOption{JSONDisallowUnknownFields},
			args: args{
				data: []byte(`{"int": 123, "foo": "bar"}`),
				v:    &TestConfig{},
			},
			wantErr: true,
		},
		{
			name: "unknown keys on no disallow unknown fields",
			opts: []JSONFormatterOption{},
			args: args{
				data: []byte(`{"int": 123, "foo": "bar"}`),
				v:    &TestConfig{},
			},
			wantErr: false,
			want:    &TestConfig{Int: 123},
		},
		{
			name: "unmarshal into struct",
			args: args{
				data: []byte(`{"int": 123, "inner": {"string": "test"}}`),
				v:    &TestConfig{},
			},
			wantErr: false,
			want:    &TestConfig{Int: 123, Inner: testInnerConfig{String: "test"}},
		},
		{
			name: "nil option",
			opts: []JSONFormatterOption{nil},
			args: args{
				data: []byte(`{"int": 123, "inner": {"string": "test"}}`),
				v:    &TestConfig{},
			},
			wantErr: false,
			want:    &TestConfig{Int: 123, Inner: testInnerConfig{String: "test"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jf := NewJSONFormatter(tt.opts...)
			if err := jf.Unmarshal(tt.args.data, tt.args.v); (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(tt.args.v, tt.want) {
				t.Fatalf("Unmarshal() got = %v, want %v", tt.args.v, tt.want)
			}
		})
	}
}

func TestYAMLFormatter_Unmarshal(t *testing.T) {
	type args struct {
		data []byte
		v    any
	}
	tests := []struct {
		name    string
		opts    []YAMLFormatterOption
		args    args
		wantErr bool
		want    any
	}{
		{
			name: "empty",
			args: args{
				data: []byte(""),
				v:    &map[string]any{},
			},
			wantErr: true,
		},
		{
			name: "with keys",
			args: args{
				data: []byte(`foo: bar`),
				v:    &map[string]any{},
			},
			wantErr: false,
			want:    &map[string]any{"foo": "bar"},
		},
		{
			name: "unknown keys on disallow unknown fields",
			opts: []YAMLFormatterOption{YAMLDisallowUnknownFields},
			args: args{
				data: []byte("int: 123\nfoo: bar\n"),
				v:    &TestConfig{},
			},
			wantErr: true,
		},
		{
			name: "unknown keys on no disallow unknown fields",
			opts: []YAMLFormatterOption{},
			args: args{
				data: []byte("int: 123\nfoo: bar\n"),
				v:    &TestConfig{},
			},
			wantErr: false,
			want:    &TestConfig{Int: 123},
		},
		{
			name: "unmarshal into struct",
			args: args{
				data: []byte("int: 123\ninner:\n  string: test"),
				v:    &TestConfig{},
			},
			wantErr: false,
			want:    &TestConfig{Int: 123, Inner: testInnerConfig{String: "test"}},
		},
		{
			name: "nil option",
			opts: []YAMLFormatterOption{nil},
			args: args{
				data: []byte("int: 123\ninner:\n  string: test"),
				v:    &TestConfig{},
			},
			wantErr: false,
			want:    &TestConfig{Int: 123, Inner: testInnerConfig{String: "test"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jf := NewYAMLFormatter(tt.opts...)
			if err := jf.Unmarshal(tt.args.data, tt.args.v); (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.wantErr {
				return
			}
			if !reflect.DeepEqual(tt.args.v, tt.want) {
				t.Fatalf("Unmarshal() got = %v, want %v", tt.args.v, tt.want)
			}
		})
	}
}
