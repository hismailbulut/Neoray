package main

import (
	"flag"
	"path/filepath"
	"strconv"
)

type Args struct {
	file           string
	line           int
	column         int
	singleInstance bool

	nvimArgs []string
}

func ParseArgs(args []string) Args {
	options := Args{}
	flag.StringVar(&options.file, "file", "", "Specify a filename to open in neovim. This is useful when -si flag has given.")
	flag.IntVar(&options.line, "line", -1, "Goto line number.")
	flag.IntVar(&options.column, "column", -1, "Goto column number.")
	flag.BoolVar(&options.singleInstance, "singleinstance", false, "If this option has given neoray will open only one instance. All neoray commands will send all flags to already open instance and immediately close.")
	flag.BoolVar(&options.singleInstance, "si", false, "Shortland for singleinstance")
	flag.Parse()
	options.nvimArgs = flag.Args()
	return options
}

// Call this before starting neovim.
func (options Args) ProcessBefore() bool {
	dontStart := false
	if options.singleInstance {
		// First we will check only once because sending and
		// waiting http requests will make neoray opens slower.
		client, err := CreateClient()
		if err == nil {
			if client.SendSignal(SIGNAL_CHECK_CONNECTION) {
				if options.file != "" {
					fullPath, err := filepath.Abs(options.file)
					if err == nil {
						dontStart = client.SendSignal(SIGNAL_OPEN_FILE, fullPath)
					}
				}
				if options.line != -1 {
					dontStart = client.SendSignal(SIGNAL_GOTO_LINE, strconv.Itoa(options.line))
				}
				if options.column != -1 {
					dontStart = client.SendSignal(SIGNAL_GOTO_COLUMN, strconv.Itoa(options.column))
				}

			}
			client.Close()
		} else {
			log_debug("Error when creating client:", err)
		}
	}
	return dontStart
}

// Call this after connected neovim as ui.
func (options Args) ProcessAfter() {
	if options.singleInstance {
		server, err := CreateServer()
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to create TCP listener.")
		} else {
			EditorSingleton.server = server
		}
	}
	if options.file != "" {
		EditorSingleton.nvim.OpenFile(options.file)
	}
	if options.line != -1 {
		EditorSingleton.nvim.GotoLine(options.line)
	}
	if options.column != -1 {
		EditorSingleton.nvim.GotoColumn(options.column)
	}
}
