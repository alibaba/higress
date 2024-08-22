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
			want: append([]ChatHistory{}, firstChat...)},
		{name: "单次去重", args: args{
			chat:        append([]ChatHistory{}, firstChat...),
			currMessage: append(append([]ChatHistory{}, firstChat...), sendUser...)},
			want: []ChatHistory{}},
		{name: "两次去重", args: args{
			chat:        append(append([]ChatHistory{}, firstChat...), secondChat...),
			currMessage: append(append(append([]ChatHistory{}, firstChat...), secondChat...), thirdUser...)},
			want: []ChatHistory{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DistinctChat(tt.args.chat, tt.args.currMessage); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DistinctChat() = %v, want %v", got, tt.want)
			}
		})
	}
}
