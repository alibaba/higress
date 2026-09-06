// Copyright (c) 2026 Alibaba Group Holding Ltd.
// SPDX-License-Identifier: Apache-2.0

// A deterministic Agent fixture: keeps task state in one process, without an LLM.
package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type object = map[string]interface{}

func main() {
	listen := flag.String("listen", ":8080", "listen address")
	identity := flag.String("identity", "kubernetes-agent", "fixture identity")
	flag.Parse()
	var mu sync.Mutex
	tasks := map[string]object{}
	contexts := map[string]int{}
	node, _ := os.Hostname()
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		log.Fatal(err)
	}
	incarnation := fmt.Sprintf("%s-%x", node, nonce)
	write := func(w http.ResponseWriter, r *http.Request, body interface{}) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Demo-Agent", *identity)
		w.Header().Set("X-Demo-Node", incarnation)
		if r.URL.Query().Get("trailers") == "true" {
			w.Header().Set("Trailer", "X-Demo-Trailer")
		}
		json.NewEncoder(w).Encode(body)
		w.Header().Set("X-Demo-Trailer", "complete")
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.Contains(r.URL.Path, "/.well-known/agent") {
			card := object{"name": *identity, "description": "Deterministic A2A test Agent", "version": "1.0.0",
				"capabilities": object{"streaming": true}, "defaultInputModes": []string{"text/plain"},
				"defaultOutputModes": []string{"text/plain"}, "skills": []object{{"id": "echo", "name": "Echo", "description": "Echo input", "tags": []string{"test"}}}}
			if r.URL.Query().Get("version") == "1.0" {
				card["supportedInterfaces"] = []object{{"url": "http://internal-agent:8080/", "protocolBinding": "JSONRPC", "protocolVersion": "1.0"}}
			} else {
				card["protocolVersion"], card["url"], card["preferredTransport"] = "0.3.0", "http://internal-agent:8080/", "JSONRPC"
			}
			write(w, r, card)
			return
		}
		if r.Method == "GET" {
			write(w, r, object{"agent": *identity})
			return
		}
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params object          `json:"params"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20)).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		mu.Lock()
		locked := true
		defer func() {
			if locked {
				mu.Unlock()
			}
		}()
		result := object{}
		method := req.Method
		switch method {
		case "SendMessage", "message/send", "SendStreamingMessage", "message/stream":
			id := fmt.Sprintf("%s-task-%d", incarnation, len(tasks)+1)
			contextID := "context-" + id
			msg, _ := req.Params["message"].(map[string]interface{})
			if provided, _ := msg["contextId"].(string); provided != "" {
				if _, found := contexts[provided]; !found {
					write(w, r, object{"jsonrpc": "2.0", "id": req.ID, "error": object{"code": -32001, "message": "Context not found on this node"}})
					return
				}
				contextID = provided
			}
			contexts[contextID]++
			state := "completed"
			if r.URL.Query().Get("working") == "true" {
				state = "working"
			}
			result = object{"kind": "task", "id": id, "contextId": contextID, "metadata": object{"node": incarnation, "turn": contexts[contextID]}, "status": object{"state": state},
				"artifacts": []object{{"artifactId": "echo", "parts": []object{{"kind": "text", "text": "hello from " + *identity}}}}}
			tasks[id] = result
		case "GetTask", "tasks/get", "CancelTask", "tasks/cancel", "SubscribeToTask", "tasks/resubscribe":
			id, _ := req.Params["id"].(string)
			if id == "" {
				id, _ = req.Params["taskId"].(string)
			}
			var found bool
			result, found = tasks[id]
			if !found {
				write(w, r, object{"jsonrpc": "2.0", "id": req.ID, "error": object{"code": -32001, "message": "Task not found"}})
				return
			}
			if method == "CancelTask" || method == "tasks/cancel" {
				result["status"] = object{"state": "canceled"}
			}
		default:
			write(w, r, object{"jsonrpc": "2.0", "id": req.ID, "error": object{"code": -32601, "message": "Unknown method reached upstream"}})
			return
		}
		if method == "SendStreamingMessage" || method == "message/stream" || method == "SubscribeToTask" || method == "tasks/resubscribe" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("X-Demo-Agent", *identity)
			w.Header().Set("X-Demo-Node", incarnation)
			w.Header().Set("Trailer", "X-Demo-Trailer")
			snapshot := object{}
			for k, v := range result {
				snapshot[k] = v
			}
			result = snapshot
			locked = false
			mu.Unlock()
			for i, state := range []string{"working", "completed"} {
				if i > 0 {
					time.Sleep(600 * time.Millisecond)
				}
				event := object{"kind": "status-update", "taskId": result["id"], "contextId": result["contextId"], "status": object{"state": state}, "final": i == 1}
				if i == 0 {
					event = result
					event["status"] = object{"state": "working"}
				}
				data, _ := json.Marshal(object{"jsonrpc": "2.0", "id": req.ID, "result": event})
				fmt.Fprintf(w, "data: %s\n\n", data)
				w.(http.Flusher).Flush()
			}
			mu.Lock()
			tasks[result["id"].(string)]["status"] = object{"state": "completed"}
			mu.Unlock()
			w.Header().Set("X-Demo-Trailer", "complete")
			return
		}
		write(w, r, object{"jsonrpc": "2.0", "id": req.ID, "result": result})
	})
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	log.Fatal((&http.Server{Addr: *listen, Protocols: protocols}).ListenAndServe())
}
