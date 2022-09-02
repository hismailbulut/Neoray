//go:build !debug
// +build !debug

package bench

import "github.com/hismailbulut/Neoray/pkg/logger"

const BUILD_TYPE = logger.ReleaseBuild

func IsDebugBuild() bool { return false }

func ToggleCpuProfile(string) error { return nil }

func DumpHeapProfile(string) error { return nil }

func BeginBenchmark() (EndBenchmark func(name string)) { return func(name string) {} }

func PrintResults() {}
