package netrc

import (
	"reflect"
	"testing"
)

func Test_parseNetrc(t *testing.T) {
	type args struct {
		data string
	}
	tests := []struct {
		name string
		args args
		want []netrcLine
	}{
		{
			name: "parses netrc file properly",
			args: args{
				data: "machine test.sample.org\nlogin mylogin@sample.org\npassword mypassword",
			},
			want: []netrcLine{
				{
					machine:  "test.sample.org",
					login:    "mylogin@sample.org",
					password: "mypassword",
				},
			},
		},
		{
			name: "does not parse incomplete netrc",
			args: args{
				data: "machine test.sample.org\npassword mypassword",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseNetrc(tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseNetrc() = %v, want %v", got, tt.want)
			}
		})
	}
}
