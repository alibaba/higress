package main

import (
    "testing"
)

func TestParsePrefixedToolName(t *testing.T) {
    cases := []struct{
        in string
        wantServer string
        wantTool string
        ok bool
    }{
        {"server___tool", "server", "tool", true},
        {"server/tool", "server", "tool", true},
        {"not-prefixed", "", "", false},
        {"server___", "", "", false},
        {"/tool", "", "", false},
    }
    for _, c := range cases {
        s, tname, ok := parsePrefixedToolName(c.in)
        if ok != c.ok {
            t.Fatalf("%s: ok=%v, want %v", c.in, ok, c.ok)
        }
        if !ok {
            continue
        }
        if s != c.wantServer || tname != c.wantTool {
            t.Fatalf("%s: got (%s,%s), want (%s,%s)", c.in, s, tname, c.wantServer, c.wantTool)
        }
    }
}


