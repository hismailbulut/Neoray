//go:build !debug
// +build !debug

package bench

import (
	"io"

	"github.com/hismailbulut/Neoray/pkg/logger"
)

const BUILD_TYPE = logger.ReleaseBuild

func IsDebugBuild() bool { return false }

func Begin() func(name ...string)

func PrintResults(out io.Writer)
