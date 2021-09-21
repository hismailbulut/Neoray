// +build debug

package main

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/hismailbulut/neoray/src/measurer"
)

// NOTE: All functions, types, constants and variables must exist in utils_release.go,
// Add empty ones to release file if they are not local.

const (
	MINIMUM_LOG_LEVEL       = LEVEL_DEBUG
	BUILD_TYPE              = DEBUG
	FONT_ATLAS_DEFAULT_SIZE = 512
)

func start_pprof() {
	defer measure_execution_time()()
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			logMessage(LEVEL_ERROR, TYPE_NEORAY, "Failed to start pprof:", err)
		}
	}()
}

func init_function_time_tracker() {
	measurer.Init()
}

func measure_execution_time() func(custom ...string) {
	return measurer.Measure()
}

// This function is only for beautifully printing average times and its stuff are unnecessary.
func close_function_time_tracker() {
	measurer.Close()
}

// This assert only works on debug build.
func assert_debug(cond bool, message ...interface{}) {
	if !cond {
		logMessage(LEVEL_FATAL, TYPE_NEORAY, "Debug assertion failed:", message)
	}
}
