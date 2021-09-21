// +build !debug

package main

const (
	MINIMUM_LOG_LEVEL       = LEVEL_TRACE
	BUILD_TYPE              = RELEASE
	FONT_ATLAS_DEFAULT_SIZE = 2048
)

func start_pprof() {}

func init_function_time_tracker() {}

func measure_execution_time() func(custom ...string) { return func(custom ...string) {} }

func close_function_time_tracker() {}

// This assert only works on debug build.
func assert_debug(cond bool, message ...interface{}) {}
