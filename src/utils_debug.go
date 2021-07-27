// +build debug

package main

import (
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sync"
	"time"
)

// NOTE: All functions, types, constants and variables must exist in brelease.go,
// Add empty ones to release file if they are not local.

const (
	MINIMUM_LOG_LEVEL       = LOG_LEVEL_DEBUG
	BUILD_TYPE              = DEBUG
	FONT_ATLAS_DEFAULT_SIZE = 256
)

func start_pprof() {
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to start pprof:", err)
		}
	}()
}

type function_measure struct {
	totalCall int64
	totalTime time.Duration
}

var (
	trackerAverages map[string]function_measure
	trackerMutex    sync.Mutex
)

func init_function_time_tracker() {
	trackerAverages = make(map[string]function_measure)
}

func measure_execution_time() func(uname ...string) {
	now := time.Now()
	return func(uname ...string) {
		name := "Unrecognized"
		if len(uname) > 0 {
			name = uname[0]
		} else {
			pc, _, _, ok := runtime.Caller(1)
			if ok {
				name = runtime.FuncForPC(pc).Name()
			}
		}
		elapsed := time.Since(now)
		trackerMutex.Lock()
		defer trackerMutex.Unlock()
		if val, ok := trackerAverages[name]; ok == true {
			val.totalCall++
			val.totalTime += elapsed
			trackerAverages[name] = val
		} else {
			trackerAverages[name] = function_measure{
				totalCall: 1,
				totalTime: elapsed,
			}
		}
	}
}

func close_function_time_tracker() {
	trackerMutex.Lock()
	defer trackerMutex.Unlock()
	for key, val := range trackerAverages {
		logMessage(LOG_LEVEL_DEBUG, LOG_TYPE_PERFORMANCE,
			key, "Calls:", val.totalCall, "Time:", val.totalTime, "Average:", val.totalTime/time.Duration(val.totalCall))
	}
}

// This assert only works on debug build.
func assert_debug(cond bool, message ...interface{}) {
	if !cond {
		logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Debug assertion failed:", message)
	}
}
