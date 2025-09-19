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
