//go:build tinygo

package main

import _ "github.com/wasilibs/nottinygc"

// Compiled by nottinygc for delayed free but Envoy doesn't stub it yet,
// luckily nottinygc doesn't actually call the function, so it's fine to
// stub it out.

//export sched_yield
func sched_yield() int32 {
	return 0
}
