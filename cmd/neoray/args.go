package main

import (
	"path/filepath"
	"strconv"
	"strings"
)

type Args struct {
	file           string
	line           int
	column         int
	singleInstance bool
}

func ParseArgs(args []string) (Args, []string) {
	options := Args{
		line:   -1,
		column: -1,
	}
	otherOptions := make([]string, 0)
	for _, arg := range args {
		prefix := arg
		equalsIndex := strings.Index(arg, "=")
		if equalsIndex != -1 {
			prefix = arg[:equalsIndex]
		}
		switch prefix {
		case "--file":
			if equalsIndex != -1 {
				options.file = arg[equalsIndex+1:]
			}
			break
		case "--line":
			if equalsIndex != -1 {
				line, err := strconv.Atoi(arg[equalsIndex+1:])
				if err == nil {
					options.line = line
				}
			}
			break
		case "--column":
			if equalsIndex != -1 {
				column, err := strconv.Atoi(arg[equalsIndex+1:])
				if err == nil {
					options.column = column
				}
			}
			break
		case "--single-instance", "-si":
			options.singleInstance = true
			break
		default:
			otherOptions = append(otherOptions, arg)
			break
		}
	}
	return options, otherOptions
}

// Call this before starting neovim.
func (options Args) ProcessBefore() bool {
	dontStart := false
	if options.singleInstance {
		if options.file != "" {
			fullPath, err := filepath.Abs(options.file)
			if err == nil {
				dontStart = SendSignal(FORMAT_OPENFILE, fullPath)
			} else {
				log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "Failed to send openfile signal:", err)
			}
		}
		if options.line != -1 {
			dontStart = SendSignal(FORMAT_GOTOLINE, options.line)
		}
		if options.column != -1 {
			dontStart = SendSignal(FORMAT_GOTOCOL, options.column)
		}
		if !dontStart {
			CreateServer()
		}
	}
	return dontStart
}

// Call this after connected neovim as ui.
func (options Args) ProcessAfter() {
	if options.file != "" {
		EditorSingleton.nvim.ExecuteVimScript(":edit %s", options.file)
	}
	if options.line != -1 {
		EditorSingleton.nvim.ExecuteVimScript("call cursor(%d, 0)", options.line)
	}
	if options.column != -1 {
		EditorSingleton.nvim.ExecuteVimScript("call cursor(0, %d)", options.column)
	}
}
