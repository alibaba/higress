package main

import (
	"reflect"
	"testing"
)

func TestDistinctChat(t *testing.T) {
	type args struct {
		chat        []ChatHistory
		currMessage []ChatHistory
	}
	firstChat := []ChatHistory{{Role: "user", Content: "userInput1"}, {Role: "assistant", Content: "assistantOutput1"}}
	sendUser := []ChatHistory{{Role: "user", Content: "userInput2"}}
	tests := []struct {
		name string
		args args
		want []ChatHistory
	}{
		{name: "填充历史", args: args{
			chat:        append([]ChatHistory{}, firstChat...),
			currMessage: append([]ChatHistory{}, sendUser...)},
			want: append(append([]ChatHistory{}, firstChat...), sendUser...)},
		{name: "无需填充", args: args{
			chat:        append([]ChatHistory{}, firstChat...),
			currMessage: append(append([]ChatHistory{}, firstChat...), sendUser...)},
			want: append(append([]ChatHistory{}, firstChat...), sendUser...)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fillHistory(tt.args.chat, tt.args.currMessage, 3); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fillHistory() = %v, want %v", got, tt.want)
			}
		})
	}
}
