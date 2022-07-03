//go:build !debug
// +build !debug

package main

import (
	"github.com/hismailbulut/neoray/src/logger"
)

const BUILD_TYPE = logger.Release

func toggle_cpu_profile() {}

func dump_heap_profile() {}

// This assert only works on debug build.
func assert_debug(cond bool, message ...interface{}) {}
