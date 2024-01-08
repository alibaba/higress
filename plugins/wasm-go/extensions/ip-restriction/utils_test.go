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
		wantVal bool
		wantErr bool
	}{
		{
			name: "",
			args: args{
				array: gjson.Parse(`["127.0.0.1/30","10.0.0.1"]`).Array(),
			},
			wantVal: true,
			wantErr: false,
		},
		{
			name: "",
			args: args{
				array: gjson.Parse(``).Array(),
			},
			wantVal: false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIPNets(tt.args.array)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIPNets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantVal && got == nil {
				return
			}
			if _, found, _ := got.GetByString("10.0.0.1"); found != tt.wantVal {
				t.Errorf("parseIPNets() got = %v, want %v", found, tt.wantVal)
				return
			}
		})
	}
}
