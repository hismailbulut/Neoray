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
	MINIMUM_LOG_LEVEL       = LEVEL_DEBUG
	BUILD_TYPE              = DEBUG
	FONT_ATLAS_DEFAULT_SIZE = 512
)

func start_pprof() {
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			logMessage(LEVEL_ERROR, TYPE_NEORAY, "Failed to start pprof:", err)
		}
	}()
}

type function_measure struct {
	totalCall int
	totalTime time.Duration
	maxTime   time.Duration
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
			if elapsed > val.maxTime {
				val.maxTime = elapsed
			}
			trackerAverages[name] = val
		} else {
			trackerAverages[name] = function_measure{
				totalCall: 1,
				totalTime: elapsed,
				maxTime:   elapsed,
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
		avg     time.Duration
		average string
		highest string
	}
	maxNameLen := 0
	maxCallLen := 0
	maxTimeLen := 0
	maxAverageLen := 0
	maxHighestLen := 0
	sorted := []funcTimeSummary{}
	for key, val := range trackerAverages {
		sum := funcTimeSummary{
			name:    strings.Replace(key, "main.", "", 1),
			calls:   fmt.Sprint(val.totalCall),
			time:    fmt.Sprint(val.totalTime),
			avg:     val.totalTime / time.Duration(val.totalCall),
			average: fmt.Sprint(val.totalTime / time.Duration(val.totalCall)),
			highest: fmt.Sprint(val.maxTime),
		}
		sorted = append(sorted, sum)
		if len(sum.name) > maxNameLen {
			maxNameLen = len(sum.name)
		}
		if len(sum.calls) > maxCallLen {
			maxCallLen = len(sum.calls)
		}
		if len([]rune(sum.time)) > maxTimeLen {
			maxTimeLen = len([]rune(sum.time))
		}
		if len([]rune(sum.average)) > maxAverageLen {
			maxAverageLen = len([]rune(sum.average))
		}
		if len([]rune(sum.highest)) > maxHighestLen {
			maxHighestLen = len([]rune(sum.highest))
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].avg > sorted[j].avg
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
		if ft.calls != "1" {
			spaces = maxTimeLen - len([]rune(ft.time))
			for i := 0; i < spaces; i++ {
				msg += " "
			}
			msg += " Avg:" + ft.average
			spaces = maxAverageLen - len([]rune(ft.average))
			for i := 0; i < spaces; i++ {
				msg += " "
			}
			msg += " Max:" + ft.highest
		}
		logMessage(LEVEL_DEBUG, TYPE_PERFORMANCE, msg)
	}
}

// This assert only works on debug build.
func assert_debug(cond bool, message ...interface{}) {
	if !cond {
		logMessage(LEVEL_FATAL, TYPE_NEORAY, "Debug assertion failed:", message)
	}
}
