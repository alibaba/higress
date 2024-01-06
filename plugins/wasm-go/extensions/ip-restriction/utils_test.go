package main

import (
	"github.com/tidwall/gjson"
	"testing"
)

func Test_parseIPNets(t *testing.T) {
	type args struct {
		array []gjson.Result
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "",
			args: args{
				array: gjson.Parse(`["127.0.0.1/30","10.0.0.1"]`).Array(),
			},
			wantErr: false,
		},
		{
			name: "",
			args: args{
				array: gjson.Parse(``).Array(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIPNets(tt.args.array)
			t.Logf("pasre result: %v", got)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIPNets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
