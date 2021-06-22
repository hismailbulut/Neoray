package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/sqweek/dialog"
)

const (
	// log levels
	LOG_LEVEL_DEBUG = iota
	LOG_LEVEL_TRACE
	LOG_LEVEL_WARN
	LOG_LEVEL_ERROR
	LOG_LEVEL_FATAL
	// log types
	LOG_TYPE_NVIM
	LOG_TYPE_NEORAY
	LOG_TYPE_RENDERER
	LOG_TYPE_PERFORMANCE
)

func log_message(log_level, log_type int, message ...interface{}) {
	if log_level < MINIMUM_LOG_LEVEL {
		return
	}

	fatal := false
	log_string := " "

	switch log_type {
	case LOG_TYPE_NVIM:
		log_string += "[NEOVIM]"
	case LOG_TYPE_NEORAY:
		log_string += "[NEORAY]"
	case LOG_TYPE_RENDERER:
		log_string += "[RENDERER]"
	case LOG_TYPE_PERFORMANCE:
		log_string += "[PERFORMANCE]"
	default:
		return
	}

	log_string += " "
	switch log_level {
	case LOG_LEVEL_DEBUG:
		log_string += ">>"
	case LOG_LEVEL_TRACE:
		log_string += "TRACE:"
	case LOG_LEVEL_WARN:
		log_string += "WARNING:"
	case LOG_LEVEL_ERROR:
		log_string += "ERROR:"
	case LOG_LEVEL_FATAL:
		log_string += "FATAL:"
		fatal = true
	default:
		return
	}

	log_string += " "
	for _, msg := range message {
		log_string += fmt.Sprint(msg)
		log_string += " "
	}

	if fatal {
		if isDebugBuild() {
			fmt.Printf("\n")
			debug.PrintStack()
			log.Fatalln(log_string)
		} else {
			dialog.Message(log_string).Title("Fatal error").Error()
			os.Exit(1)
		}
	} else {
		log.Println(log_string)
	}
}

func log_debug(message ...interface{}) {
	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, message...)
}

func assert(cond bool, message ...interface{}) {
	if cond == false {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Assertion Failed:", message)
	}
}
