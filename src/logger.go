package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/sqweek/dialog"
)

type LogLevel uint32
type LogType uint32

const (
	// log levels
	LOG_LEVEL_DEBUG LogLevel = iota
	LOG_LEVEL_TRACE
	LOG_LEVEL_WARN
	LOG_LEVEL_ERROR
	// The fatal will be printed to logfile and a
	// fatal popup will be shown. The program exits immediately.
	LOG_LEVEL_FATAL
)

const (
	// Log types, makes easy to understand where the message coming from
	LOG_TYPE_NVIM LogType = iota
	LOG_TYPE_NEORAY
	LOG_TYPE_RENDERER
	LOG_TYPE_PERFORMANCE
)

var (
	verboseFile *os.File = nil
)

func initVerboseFile(filename string) {
	assert(verboseFile == nil, "multiple initialization of log file")
	path, err := filepath.Abs(filename)
	if err != nil {
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to get absolute path:", err)
		return
	}
	verboseFile, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0666)
	if err != nil {
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to create log file:", err)
		return
	}
	// Print informations to log file.
	verboseFile.WriteString(fmt.Sprintln("\nNEORAY", versionString(), buildTypeString(), "LOG", time.Now().UTC()))
}

func shutdownLogger() {
	// If we are panicking print it to logfile.
	// Also the stack trace will be printed after
	// fatal error.
	if pmsg := recover(); pmsg != nil {
		// Create crash report.
		crash_msg := "An unexpected error occured and generated a crash report."
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, crash_msg)
		dialog.Message(crash_msg).Error()
		createCrashReport("PANIC", pmsg)
	}
	// If logfile is initialized then close it.
	if verboseFile != nil {
		verboseFile.Close()
	}
}

func createCrashReport(msg ...interface{}) {
	crash_file, err := os.Create("neoray_crash.log")
	if err == nil {
		defer crash_file.Close()
		crash_file.WriteString("NEORAY " + versionString() + " " + buildTypeString() + " Crash Report\n")
		crash_file.WriteString("Please open an issue in github with this file.\n")
		crash_file.WriteString("The program is crashed because of the following reasons.\n")
		crash_file.WriteString("Message: " + fmt.Sprintln(msg...))
		crash_file.WriteString(fmt.Sprintln(string(debug.Stack())))
	}
}

func logMessage(log_level LogLevel, log_type LogType, message ...interface{}) {
	if log_level < MINIMUM_LOG_LEVEL && verboseFile == nil {
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

	for _, msg := range message {
		log_string += " " + fmt.Sprint(msg)
	}

	if verboseFile != nil {
		verboseFile.WriteString(log_string + "\n")
	}

	log.Println(log_string)

	if fatal {
		dialog.Message(log_string).Error()
		createCrashReport(log_string)
		os.Exit(1)
	}
}

func logDebug(message ...interface{}) {
	logMessage(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, message...)
}

func logfDebug(format string, message ...interface{}) {
	logMessage(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, fmt.Sprintf(format, message...))
}

func assert(cond bool, message ...interface{}) {
	if cond == false {
		logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Assertion Failed:", fmt.Sprint(message...))
	}
}
