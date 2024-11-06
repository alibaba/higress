package main

import (
	"log"
	"net/http"
	"time"
)

var events = []string{
	": this is a test stream\n\n",

	"data: some text\n",
	"data: another message\n",
	"data: with two lines\n\n",

	"event: userconnect\n",
	"data: {\"username\": \"bobby\", \"time\": \"02:33:48\"}\n\n",

	"event: usermessage\n",
	"data: {\"username\": \"bobby\", \"time\": \"02:34:11\", \"text\": \"Hi everyone.\"}\n\n",

	"event: userdisconnect\n",
	"data: {\"username\": \"bobby\", \"time\": \"02:34:23\"}\n\n",

	"event: usermessage\n",
	"data: {\"username\": \"sean\", \"time\": \"02:34:36\", \"text\": \"Bye, bobby.\"}\n\n",
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("receive request")
		w.Header().Set("Content-Type", "text/event-stream")
		for _, e := range events {
			_, _ = w.Write([]byte(e))
			time.Sleep(1 * time.Second)
			w.(http.Flusher).Flush()
		}
	})
	if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
		panic(err)
	}
}
