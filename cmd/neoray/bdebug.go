// +build !release

package main

import (
	"net/http"
	_ "net/http/pprof"
	"time"
)

// NOTE: All functions, types, constants and variables must exist in brelease.go,
// Add empty ones to release file.

const (
	MINIMUM_LOG_LEVEL = LOG_LEVEL_DEBUG
	BUILD_TYPE        = DEBUG
)

func start_pprof() {
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to start pprof:", err)
		}
	}()
}

type function_measure struct {
	totalCall int64
	totalTime time.Duration
}

var (
	trackerAverages map[string]function_measure
)

func init_function_time_tracker() {
	trackerAverages = make(map[string]function_measure)
}

func measure_execution_time(name string) func() {
	now := time.Now()
	return func() {
		elapsed := time.Since(now)
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
	for key, val := range trackerAverages {
		log_message(LOG_LEVEL_DEBUG, LOG_TYPE_PERFORMANCE,
			key, "Calls:", val.totalCall, "Time:", val.totalTime, "Average:", val.totalTime/time.Duration(val.totalCall))
	}
}
