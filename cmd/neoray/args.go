package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/hismailbulut/Neoray/pkg/bench"
	"github.com/hismailbulut/Neoray/pkg/fontfinder"
	"github.com/hismailbulut/Neoray/pkg/logger"
	"github.com/sqweek/dialog"
)

var usageTemplate = `
Neoray is a simple and lightweight GUI client for Neovim
Version %d.%d.%d %s
License %s
Webpage %s

Options:

--file <name>
	Filename to open
--line <number>
	Cursor goes to line <number>
--column <number>
	Cursor goes to column <number>
--singleinstance, -si
	Only accepts one instance of neoray and sends all flags to it
--verbose
	Prints verbose debug output to a file
--nvim <path>
	Relative or absolute path to nvim executable
--server <address>
	Connect to existing neovim instance
--multigrid
	Enables multigrid support (experimental)
--list-fonts <file>
	Lists all fonts and writes them to <file>
--version, -v
	Prints only the version and quits
--help, -h
	Prints this message and quits

All other flags will send to neovim
`

type ParsedArgs struct {
	file       string
	line       int
	column     int
	singleInst bool
	execPath   string
	address    string
	multiGrid  bool
	others     []string
}

// Last boolean value specifies if we should quit after parsing
func ParseArgs(args []string) (ParsedArgs, error, bool) {
	// Init defaults
	options := ParsedArgs{
		file:       "",
		line:       -1,
		column:     -1,
		singleInst: false,
		execPath:   "nvim",
		address:    "",
		multiGrid:  false,
		others:     []string{},
	}
	var err error
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--file":
			if i+1 >= len(args) {
				return options, errors.New("specify filename after --file"), false
			}
			options.file = args[i+1]
			i++
		case "--line":
			if i+1 >= len(args) {
				return options, errors.New("specify line number after --line"), false
			}
			options.line, err = strconv.Atoi(args[i+1])
			if err != nil {
				return options, errors.New("invalid line number"), false
			}
			i++
		case "--column":
			if i+1 >= len(args) {
				return options, errors.New("specify column number after --column"), false
			}
			options.column, err = strconv.Atoi(args[i+1])
			if err != nil {
				return options, errors.New("invalid column number"), false
			}
			i++
		case "--singleinstance", "-si":
			options.singleInst = true
		case "--verbose":
			logger.InitFile("Neoray_verbose.log")
		case "--nvim":
			if i+1 >= len(args) {
				return options, errors.New("specify nvim executable after --nvim"), false
			}
			nvimPath, err := filepath.Abs(args[i+1])
			if err == nil {
				options.execPath = nvimPath
			}
			i++
		case "--server":
			if i+1 >= len(args) {
				return options, errors.New("specify server address after --server"), false
			}
			options.address = args[i+1]
			i++
		case "--multigrid":
			options.multiGrid = true
		case "--list-fonts":
			if i+1 >= len(args) {
				return options, errors.New("specify file name after --list-fonts"), false
			}
			fileName := args[i+1]
			ListFonts(fileName)
			return options, nil, true
		case "--version", "-v":
			PrintVersion()
			return options, nil, true
		case "--help", "-h":
			PrintHelp()
			return options, nil, true
		default:
			options.others = append(options.others, args[i])
		}
	}
	return options, nil, false
}

func PrintVersion() {
	version := logger.Version{Major: VERSION_MAJOR, Minor: VERSION_MINOR, Patch: VERSION_PATCH}
	msg := "Neoray " + version.String() + "\n" + "Start with -h option for more information."
	switch runtime.GOOS {
	case "windows":
		dialog.Message(msg).Title("Version").Info()
	default:
		fmt.Println(msg)
	}
}

func PrintHelp() {
	// About
	msg := fmt.Sprintf(usageTemplate,
		VERSION_MAJOR, VERSION_MINOR, VERSION_PATCH,
		bench.BUILD_TYPE, LICENSE, WEBPAGE)
	dialog.Message(msg).Title("Help").Info()
	if runtime.GOOS != "windows" {
		// Also print help to stdout for linux and darwin
		fmt.Println(msg)
	}
}

func ListFonts(fileName string) {
	fontList := fontfinder.List()
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		logger.Log(logger.FATAL, "Could not open", fileName, "for writing")
	}
	defer file.Close()
	// Write to file like table
	s := strings.Builder{}
	table := logger.NewTable([]string{"ID", "Family", "Filename", "Name"})
	for i, font := range fontList {
		table.AddRow([]string{strconv.Itoa(i + 1), font.Family, font.Filename, font.Name})
	}
	table.Render(&s)
	file.WriteString(s.String())
	logger.Log(logger.TRACE, "Font list written to", fileName)
}

// Call this before starting neovim.
func (options ParsedArgs) ProcessBefore() bool {
	if options.singleInst {
		// First we will check only once because sending and
		// waiting http requests will make neoray opens slower.
		client, err := CreateClient()
		if err != nil {
			logger.Log(logger.DEBUG, "No instance found or ipc client creation failed:", err)
			return false
		}
		defer client.Close()
		if options.file != "" {
			fullPath, err := filepath.Abs(options.file)
			if err == nil {
				if !client.Call(IPC_MSG_TYPE_OPEN_FILE, fullPath) {
					return false
				}
			}
		}
		if options.line != -1 {
			if !client.Call(IPC_MSG_TYPE_GOTO_LINE, options.line) {
				return false
			}
		}
		if options.column != -1 {
			if !client.Call(IPC_MSG_TYPE_GOTO_COLUMN, options.column) {
				return false
			}
		}
		return true
	}
	return false
}

// Call this after connected neovim as ui.
func (options ParsedArgs) ProcessAfter() {
	if options.singleInst {
		server, err := CreateServer()
		if err != nil {
			logger.Log(logger.ERROR, "Failed to create ipc server:", err)
		} else {
			Editor.server = server
			logger.Log(logger.TRACE, "Ipc server created")
		}
	}
	if options.file != "" {
		Editor.nvim.openFile(options.file)
	}
	if options.line != -1 {
		Editor.nvim.gotoLine(options.line)
	}
	if options.column != -1 {
		Editor.nvim.gotoColumn(options.column)
	}
}
