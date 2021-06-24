package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sqweek/dialog"
)

var usageTemplate = `
Neoray is an ui client for neovim.
Copyright (c) 2021 Ismail Bulut.
Version %d.%d.%d %s
License %s
Webpage %s

Options:

--file
	Filename to open
--line
	Goto line number
--column
	Goto column number
--singleinstance, -si
	Only accept one instance of neoray and send all flags to it.
--verbose
	Specify a file in cwd to verbose debug output
--help, -h
	Prints this message and quits

All other flags will be send to neovim.

Copyrights:

Default font is Cascadia Code, Copyright (c) 2019 - Present,
Microsoft Corporation, licensed under SIL OPEN FONT LICENSE Version 1.1
`

var argUsages = map[string]string{}

type Args struct {
	file           string
	line           int
	column         int
	singleInstance bool
	nvimArgs       []string
}

func ParseArgs(args []string) Args {
	// Init defaults
	options := Args{
		file:           "",
		line:           -1,
		column:         -1,
		singleInstance: false,
	}
	printHelp := false
	var err error
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file":
			flagAssert(len(args) > i+1, "specify filename after --file")
			options.file = args[i+1]
			i++
			break
		case "--line":
			flagAssert(len(args) > i+1, "specify line number after --line")
			options.line, err = strconv.Atoi(args[i+1])
			flagAssert(err == nil, "invalid number after --line")
			i++
			break
		case "--column":
			flagAssert(len(args) > i+1, "specify column number after --column")
			options.column, err = strconv.Atoi(args[i+1])
			flagAssert(err == nil, "invalid number after --column")
			i++
			break
		case "--singleinstance", "-si":
			options.singleInstance = true
			break
		case "--verbose":
			flagAssert(len(args) > i+1, "specify filename after --verbose")
			init_log_file(args[i+1])
			i++
			break
		case "--help", "-h":
			printHelp = true
			break
		default:
			options.nvimArgs = append(options.nvimArgs, args[i])
			break
		}
	}
	if printHelp {
		PrintHelp()
		os.Exit(0)
	}
	return options
}

func flagAssert(cond bool, msgs ...interface{}) {
	if !cond {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, msgs...)
	}
}

func PrintHelp() {
	// About
	msg := fmt.Sprintf(usageTemplate,
		VERSION_MAJOR, VERSION_MINOR, VERSION_PATCH,
		buildTypeString(), LICENSE, WEBPAGE)
	dialog.Message(msg).Title("Neoray").Info()
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
						client.SendSignal(SIGNAL_OPEN_FILE, fullPath)
					}
				}
				if options.line != -1 {
					client.SendSignal(SIGNAL_GOTO_LINE, strconv.Itoa(options.line))
				}
				if options.column != -1 {
					client.SendSignal(SIGNAL_GOTO_COLUMN, strconv.Itoa(options.column))
				}
				dontStart = true
			}
			client.Close()
		} else {
			log_debug("No instance founded.")
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
