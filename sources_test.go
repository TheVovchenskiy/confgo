package confgo

import (
	"reflect"
	"testing"
)

func Test_stringsToBytes(t *testing.T) {
	type args struct {
		s []string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "empty slice",
			args: args{
				s: []string{},
			},
			want: []byte{},
		},
		{
			name: "single line",
			args: args{
				s: []string{"test"},
			},
			want: []byte{'t', 'e', 's', 't'},
		},
		{
			name: "multiple lines",
			args: args{
				s: []string{"1", "2", "3"},
			},
			want: []byte{'1', '\n', '2', '\n', '3'},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringsToBytes(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stringsToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}
