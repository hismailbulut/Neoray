package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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

var (
	verbose_output bool
	log_file       *os.File
)

func init_log_file(name string) {
	assert(!verbose_output, "log file already initialized")
	path, err := filepath.Abs(name)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to get absolute path:", err)
		return
	}
	log_file, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0666)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to create log file:", err)
		return
	}
	verbose_output = true
	// Print time to log file
	log_file.WriteString(fmt.Sprintln("\nNEORAY LOG", time.Now()))
}

func close_log_file() {
	if verbose_output {
		log_file.Close()
	}
}

func log_message(log_level, log_type int, message ...interface{}) {
	if log_level < MINIMUM_LOG_LEVEL && !verbose_output {
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
		log_string += "DEBUG:"
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

	if verbose_output {
		log_file.WriteString(log_string + "\n")
	}
	log.Println(log_string)
	if fatal {
		dialog.Message(log_string).Title("Fatal error").Error()
		panic("fatal error occured")
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
