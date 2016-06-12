// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Package blocktest provides tools to detect blocked goroutines in tests.
// To use it, call "defer block.Check(t)()" at the beginning of each
package block

import (
	"runtime"
	"sort"
	"strings"
	"time"
)

// interestingGoroutines returns all goroutines we care about for the purpose
// of block checking. It excludes runtime ones.
func interestingGoroutines(ignorePrefix string) (gs []string) {
	buf := make([]byte, 2<<20)
	buf = buf[:runtime.Stack(buf, true)]
	for _, g := range strings.Split(string(buf), "\n\n") {
		sl := strings.SplitN(g, "\n", 2)
		if len(sl) != 2 {
			continue
		}
		stack := strings.TrimSpace(sl[1])
		if ignorePrefix != "" && strings.HasPrefix(stack, ignorePrefix) {
			continue
		}
		if stack == "" ||
			strings.Contains(stack, "testing.Main(") ||
			strings.Contains(stack, "runtime.goexit") ||
			strings.Contains(stack, "created by runtime.gc") ||
			strings.Contains(stack, "interestingGoroutines") ||
			strings.Contains(stack, "runtime.MHeap_Scavenger") ||
			strings.Contains(stack, "signal.signal_recv") ||
			strings.Contains(stack, "sigterm.handler") ||
			strings.Contains(stack, "runtime_mcall") ||
			strings.Contains(stack, "goroutine in C code") {
			continue
		}
		gs = append(gs, g)
	}
	sort.Strings(gs)
	return
}

// ErrorReporter use to output block result
type ErrorReporter interface {
	Errorf(format string, args ...interface{})
}

// Check snapshots the currently-running goroutines and returns a
// function to be run at the end of tests to see whether any
// goroutines blocked.
func Check(t ErrorReporter, interval time.Duration, ignorePrefix string) func() {
	return func() {
		orig := map[string]bool{}
		for _, g := range interestingGoroutines(ignorePrefix) {
			orig[g] = true
		}
		time.Sleep(interval)
		var block []string
		for _, g := range interestingGoroutines(ignorePrefix) {
			if orig[g] {
				block = append(block, g)
			}
		}
		if len(block) == 0 {
			return
		}
		for _, g := range block {
			t.Errorf("Blocked goroutine: %v", g)
		}
		return
	}
}
