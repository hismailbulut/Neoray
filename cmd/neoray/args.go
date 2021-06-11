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

	nvimArgs []string
}

func ParseArgs(args []string) Args {
	options := Args{
		line:   -1,
		column: -1,
	}
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
			options.nvimArgs = append(options.nvimArgs, arg)
			break
		}
	}
	return options
}

// Call this before starting neovim.
func (options Args) ProcessBefore() bool {
	dontStart := false
	if options.singleInstance {
		if !dontStart && options.file != "" {
			fullPath, err := filepath.Abs(options.file)
			if err == nil {
				dontStart = SendSignal(FORMAT_OPENFILE, fullPath)
			}
		}
		if options.line != -1 {
			dontStart = SendSignal(FORMAT_GOTOLINE, options.line)
		}
		if options.column != -1 {
			dontStart = SendSignal(FORMAT_GOTOCOL, options.column)
		}
		if !dontStart {
			log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "No open instance founded, creating server.")
			CreateServer()
		}
	}
	return dontStart
}

// Call this after connected neovim as ui.
func (options Args) ProcessAfter() {
	if options.file != "" {
		EditorSingleton.nvim.ExecuteVimScript(FORMAT_OPENFILE, options.file)
	}
	if options.line != -1 {
		EditorSingleton.nvim.ExecuteVimScript(FORMAT_GOTOLINE, options.line)
	}
	if options.column != -1 {
		EditorSingleton.nvim.ExecuteVimScript(FORMAT_GOTOCOL, options.column)
	}
}
