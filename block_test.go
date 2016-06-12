package block

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"
)

type testReporter struct {
	failed bool
	msg    []string
}

func (tr *testReporter) Errorf(format string, args ...interface{}) {
	tr.failed = true
	tr.msg = append(tr.msg, fmt.Sprintf(format, args))
}

var blockFuncs = []func(){
	// Infinite for loop
	func() {
		for {
			time.Sleep(time.Second)
		}
	},
	// Select on a channel not referenced by other goroutines.
	func() {
		c := make(chan struct{}, 0)
		select {
		case <-c:
		}
	},
	// Blocked select on channels not referenced by other goroutines.
	func() {
		c := make(chan struct{}, 0)
		c2 := make(chan struct{}, 0)
		select {
		case <-c:
		case c2 <- struct{}{}:
		}
	},
	// Blocking wait on sync.Mutex that isn't referenced by other goroutines.
	func() {
		var mu sync.Mutex
		mu.Lock()
		mu.Lock()
	},
	// Blocking wait on sync.RWMutex that isn't referenced by other goroutines.
	func() {
		var mu sync.RWMutex
		mu.RLock()
		mu.Lock()
	},
	func() {
		var mu sync.Mutex
		mu.Lock()
		c := sync.NewCond(&mu)
		c.Wait()
	},
}

func TestCheck(t *testing.T) {
	checker := &testReporter{}
	for _, fn := range blockFuncs {
		go fn()
	}
	time.Sleep(2 * time.Second)
	Check(checker, 5*time.Second, "testing.RunTests")
	if !checker.failed {
		t.Errorf("didn't catch sleeping goroutine")
	}
	if len(checker.msg) != len(blockFuncs) {
		t.Errorf("didn't catch sleeping goroutine")
	}
	t.Log(checker.msg, "++++")
}

type Logger struct {
}

func (_ *Logger) Errorf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// This example show how to detect block in product env
// In web system, please take this code snippet, and export them as internal http api.
// In task like system, export them as signal handle and run it in standalone goroutine.
// Maybe it's useful to find some yuck problem~
func Example_DetectBlock() {
	logger := &Logger{}              // Logger used by your project
	checkInterval := 5 * time.Second // Check interval, we recommend pass this param from api
	ignorePrefix := ""               // Goroutine Prefix that will be ignored, e.g. `endpoint.CheckBlock`
	Check(logger, checkInterval, ignorePrefix)
}
