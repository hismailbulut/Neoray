// +build debug

package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"sort"
	"strings"
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
	totalCall int
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

// This function is only for beautifully printing average times and its stuff are unnecessary.
func close_function_time_tracker() {
	trackerMutex.Lock()
	defer trackerMutex.Unlock()
	type funcTimeSummary struct {
		name    string
		calls   string
		time    string
		average time.Duration
	}
	maxNameLen := 0
	maxCallLen := 0
	maxTimeLen := 0
	sorted := []funcTimeSummary{}
	for key, val := range trackerAverages {
		sum := funcTimeSummary{
			name:    strings.Replace(key, "main.", "", 1),
			calls:   fmt.Sprint(val.totalCall),
			time:    fmt.Sprint(val.totalTime),
			average: val.totalTime / time.Duration(val.totalCall),
		}
		sorted = append(sorted, sum)
		if len(sum.name) > maxNameLen {
			maxNameLen = len(sum.name)
		}
		callLen := len(sum.calls)
		if callLen > maxCallLen {
			maxCallLen = callLen
		}
		timeLen := len([]rune(sum.time))
		if timeLen > maxTimeLen {
			maxTimeLen = timeLen
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].average > sorted[j].average
	})
	for _, ft := range sorted {
		spaces := maxNameLen - len(ft.name)
		msg := ft.name
		for i := 0; i < spaces; i++ {
			msg += " "
		}
		msg += " Calls: " + ft.calls
		spaces = maxCallLen - len(ft.calls)
		for i := 0; i < spaces; i++ {
			msg += " "
		}
		msg += " Time: " + ft.time
		spaces = maxTimeLen - len([]rune(ft.time))
		for i := 0; i < spaces; i++ {
			msg += " "
		}
		msg += " Avg:"
		logMessage(LOG_LEVEL_DEBUG, LOG_TYPE_PERFORMANCE, msg, ft.average)
	}
}

// This assert only works on debug build.
func assert_debug(cond bool, message ...interface{}) {
	if !cond {
		logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Debug assertion failed:", message)
	}
}
