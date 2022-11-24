// Copyright (c) 2022 Terminus, Inc.
//
// This program is free software: you can use, redistribute, and/or modify
// it under the terms of the GNU Affero General Public License, version 3
// or later ("AGPL"), as published by the Free Software Foundation.
//
// This program is distributed in the hope that it will be useful, but WITHOUT
// ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or
// FITNESS FOR A PARTICULAR PURPOSE.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

var (
	location = "Location"
)

func main() {
	http.HandleFunc("/redirect0", func(w http.ResponseWriter, r *http.Request) {
		_ = map[string]interface{}{
			"client-request-path":   "/upstream-example/redirect0",
			"upstream-receive-path": "/redirect0",
			"upstream-location":     "http://web1:8080",
			"expected-location":     "http://localhost/web1",
		}
		w.Header().Set("Location", "http://web1:8080")
		w.WriteHeader(http.StatusCreated)
		log.Printf("path: %s, Location: %s\n", r.URL.Path, w.Header().Get(location))
	})
	http.HandleFunc("/redirect1", func(w http.ResponseWriter, r *http.Request) {
		_ = map[string]interface{}{
			"client-request-path":   "/upstream-example/redirect1",
			"upstream-receive-path": "/redirect1",
			"upstream-location":     "/user/one/two/three",
			"expected-location":     "http://one.example.com/two/three",
		}
		w.Header().Set(location, "/user/one/two/three")
		w.WriteHeader(http.StatusMovedPermanently)
		log.Printf("path: %s, Location: %s\n", r.URL.Path, w.Header().Get(location))
	})
	http.HandleFunc("/redirect2", func(w http.ResponseWriter, r *http.Request) {
		_ = map[string]interface{}{
			"client-request-path":   "/upstream-example/redirect2",
			"upstream-receive-path": "/redirect2",
			"upstream-location":     "https://baidu.com/s",
			"expected-location":     "https://baidu.com/s",
		}
		w.Header().Set(location, "https://baidu.com/s")
		w.WriteHeader(http.StatusFound)
		log.Printf("path: %s, Location: %s\n", r.URL.Path, w.Header().Get(location))
	})
	http.HandleFunc("/redirect3", func(w http.ResponseWriter, r *http.Request) {
		_ = map[string]interface{}{
			"client-request-path":   "/upstream-example/redirect3",
			"upstream-receive-path": "/redirect3",
			"upstream-location":     "localhost/one/two/three",
			"expected-location":     "localhost/one/two/three",
		}
		w.Header().Set(location, "localhost/one/two/three")
		w.WriteHeader(http.StatusSeeOther)
		log.Printf("path: %s, Location: %s\n", r.URL.Path, w.Header().Get(location))
	})
	http.HandleFunc("/redirect4/a/b/c", func(w http.ResponseWriter, r *http.Request) {
		_ = map[string]interface{}{
			"client-request-path":   "/upstream-example/redirect4/a/b/c",
			"upstream-receive-path": "/redirect4/a/b/c",
			"upstream-location":     "/one/two/three",
			"expected-location":     "/upstream-example/one/two/three",
		}
		w.Header().Set(location, "/one/two/three")
		w.WriteHeader(http.StatusTemporaryRedirect)
		log.Printf("path: %s, Location: %s\n", r.URL.Path, w.Header().Get(location))
	})
	http.HandleFunc("/upstream-example/redirect5", func(w http.ResponseWriter, r *http.Request) {
		_ = map[string]interface{}{
			"client-request-path":   "/upstream-example/redirect5",
			"upstream-receive-path": "/upstream-example/redirect5",
			"upstream-location":     "/one/two/three",
			"expected-location":     "/one/two/three",
		}
		w.Header().Set(location, "/one/two/three")
		w.WriteHeader(http.StatusTemporaryRedirect)
		log.Printf("path: %s, Location: %s\n", r.URL.Path, w.Header().Get(location))
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var m = map[string]interface{}{
			"time":             time.Now().Format(time.RFC3339),
			".Host":            r.Host,
			".URL.Host":        r.URL.Host,
			`.headers["host"]`: r.Header["host"],
			".URL":             r.URL.String(),
			".RequestURI":      r.RequestURI,
			"headers":          r.Header,
			".RemoteAddress":   r.RemoteAddr,
		}
		data, _ := json.MarshalIndent(m, "", "  ")
		log.Println(string(data))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})

	var address = ":8080"
	log.Printf("ListenAndServe %s\n", address)
	if err := http.ListenAndServe(address, http.DefaultServeMux); err != nil {
		log.Fatalf("failure to ListenAndServe %s\n", address)
	}
}
