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
	secondChat := []ChatHistory{{Role: "user", Content: "userInput2"}, {Role: "assistant", Content: "assistantOutput2"}}
	thirdUser := []ChatHistory{{Role: "user", Content: "userInput3"}}
	tests := []struct {
		name string
		args args
		want []ChatHistory
	}{
		{name: "无去重", args: args{
			chat:        append([]ChatHistory{}, firstChat...),
			currMessage: append([]ChatHistory{}, sendUser...)},
			want: append(append([]ChatHistory{}, firstChat...), sendUser...)},
		{name: "单次去重", args: args{
			chat:        append([]ChatHistory{}, firstChat...),
			currMessage: append(append([]ChatHistory{}, firstChat...), sendUser...)},
			want: append(append([]ChatHistory{}, firstChat...), sendUser...)},
		{name: "两次去重", args: args{
			chat:        append(append([]ChatHistory{}, firstChat...), secondChat...),
			currMessage: append(append([]ChatHistory{}, secondChat...), thirdUser...)},
			want: append(append(append([]ChatHistory{}, firstChat...), secondChat...), thirdUser...)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fillHistory(tt.args.chat, tt.args.currMessage, 3); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fillHistory() = %v, want %v", got, tt.want)
			}
		})
	}
}
