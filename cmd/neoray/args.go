package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sqweek/dialog"
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
	help := false
	fs := flag.NewFlagSet("usage", flag.ContinueOnError)
	fs.StringVar(&options.file, "file", "", "Specify a filename to open in neovim. This is useful when -si flag has given.")
	fs.IntVar(&options.line, "line", -1, "Goto line number.")
	fs.IntVar(&options.column, "column", -1, "Goto column number.")
	fs.BoolVar(&options.singleInstance, "singleinstance", false, "If this option has given neoray will open only one instance."+
		" All neoray commands will send all flags to already open instance and immediately close.")
	fs.BoolVar(&options.singleInstance, "si", false, "Shortland for singleinstance")
	fs.BoolVar(&help, "help", false, "Prints this message and quits.")
	fs.BoolVar(&help, "h", false, "Shortland for help.")
	err := fs.Parse(args)
	if err != nil || help {
		PrintHelp(fs)
		os.Exit(0)
	}
	options.nvimArgs = fs.Args()
	return options
}

func PrintHelp(fs *flag.FlagSet) {
	buf := bytes.NewBufferString("")
	fs.SetOutput(buf)
	fs.PrintDefaults()
	msg := "Neoray is an ui client for neovim.\n"
	msg += "Author 2021 Ismail Bulut.\n"
	msg += fmt.Sprintf("Version %d.%d.%d %s\n",
		VERSION_MAJOR, VERSION_MINOR, VERSION_PATCH, buildTypeString())
	msg += fmt.Sprintf("License %s\n", LICENSE)
	msg += fmt.Sprintf("Webpage %s\n", WEBPAGE)
	msg += "\n"
	usage, err := buf.ReadString('\x00')
	if err != nil {
		log_debug(err)
	}
	dialog.Message(msg + usage).Title("Neoray usage").Info()
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
