package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime/debug"
	"sync"
	"time"
)

func start_pprof() {
	go func() {
		err := http.ListenAndServe("localhost:6060", nil)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to create pprof server.")
		}
	}()
}

// TODO: Delete all measurement functions in release build.
type FunctionMeasure struct {
	totalCall int64
	totalTime time.Duration
}

var measure_averages map[string]FunctionMeasure
var measure_averages_mutex sync.Mutex

func init_function_time_tracker() {
	measure_averages = make(map[string]FunctionMeasure)
}

func measure_execution_time(name string) func() {
	now := time.Now()
	return func() {
		elapsed := time.Since(now)
		measure_averages_mutex.Lock()
		defer measure_averages_mutex.Unlock()
		if val, ok := measure_averages[name]; ok == true {
			val.totalCall++
			val.totalTime += elapsed
			measure_averages[name] = val
		} else {
			measure_averages[name] = FunctionMeasure{
				totalCall: 1,
				totalTime: elapsed,
			}
		}
	}
}

func close_function_time_tracker() {
	for key, val := range measure_averages {
		log_message(LOG_LEVEL_DEBUG, LOG_TYPE_PERFORMANCE,
			key, "Calls:", val.totalCall, "Time:", val.totalTime, "Average:", val.totalTime/time.Duration(val.totalCall))
	}
}

// TODO: Set log level to warn or error in release build.
const MINIMUM_LOG_LEVEL = LOG_LEVEL_DEBUG
const (
	// log levels
	LOG_LEVEL_DEBUG = iota
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

	log_string := " "
	switch log_type {
	case LOG_TYPE_NVIM:
		log_string += "[NVIM]"
	case LOG_TYPE_NEORAY:
		log_string += "[NEORAY]"
	case LOG_TYPE_RENDERER:
		log_string += "[RENDERER]"
	case LOG_TYPE_PERFORMANCE:
		log_string += "[PERFORMANCE]"
	default:
		return
	}

	fatal := false
	log_string += " "
	switch log_level {
	case LOG_LEVEL_DEBUG:
		log_string += "DEBUG:"
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
		fmt.Printf("\n")
		debug.PrintStack()
		log.Fatalln(log_string)
	} else {
		log.Println(log_string)
	}
}

func log_debug_msg(message ...interface{}) {
	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, message...)
}

func assert(cond bool, message ...interface{}) {
	if cond == false {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Assertion Failed:", message)
	}
}
