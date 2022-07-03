//go:build debug
// +build debug

package main

import (
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/hismailbulut/neoray/src/logger"
)

const BUILD_TYPE = logger.Debug

var cpuProfileFile *os.File

// NOTE: All functions, types, constants and variables must exist in utils_release.go,
// Add empty ones to release file if they are not local.

func toggle_cpu_profile() {
	if cpuProfileFile == nil {
		// New profile
		var err error
		cpuProfileFile, err = os.Create("Neoray_cpu_profile.prof")
		if err != nil {
			logger.Log(logger.ERROR, "Failed to open cpu profile file:", err)
			return
		}
		err = pprof.StartCPUProfile(cpuProfileFile)
		if err != nil {
			logger.Log(logger.ERROR, "Failed to start cpu profile:", err)
			cpuProfileFile.Close()
			cpuProfileFile = nil
			return
		}
		logger.Log(logger.DEBUG, "Profiling CPU usage...")
	} else {
		// Finish current profile
		pprof.StopCPUProfile()
		cpuProfileFile.Close()
		cpuProfileFile = nil
		logger.Log(logger.DEBUG, "CPU profile stopped and dumped to file")
	}
}

func dump_heap_profile() {
	heapFile, err := os.Create("Neoray_heap_profile.prof")
	if err != nil {
		logger.Log(logger.ERROR, "Failed to open memory profile file:", err)
		return
	}
	defer heapFile.Close()
	runtime.GC()
	err = pprof.Lookup("heap").WriteTo(heapFile, 0)
	if err != nil {
		logger.Log(logger.DEBUG, "Failed to write memory profile:", err)
	}
	logger.Log(logger.DEBUG, "Heap profile dumped to file")
}

// This assert only works on debug build.
func assert_debug(cond bool, message ...interface{}) {
	if !cond {
		logger.Log(logger.FATAL, "Debug assertion failed:", message)
	}
}
