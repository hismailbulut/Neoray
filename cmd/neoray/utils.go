package main

import (
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

const MINIMUM_LOG_LEVEL = 0

const (
	// log levels
	LOG_LEVEL_DEBUG = iota
	LOG_LEVEL_WARN
	LOG_LEVEL_ERROR
	LOG_LEVEL_FATAL
	// log types
	LOG_TYPE_NVIM
	LOG_TYPE_NEORAY
	LOG_TYPE_FAST_DEBUG_MESSAGE
)

func log_message(log_level, log_type int, message ...interface{}) {
	if log_level < MINIMUM_LOG_LEVEL {
		return
	}

	log_string := " "
	fast_debug_message := false
	switch log_type {
	case LOG_TYPE_NVIM:
		log_string += "[NVIM]"
	case LOG_TYPE_NEORAY:
		log_string += "[NEORAY]"
	case LOG_TYPE_FAST_DEBUG_MESSAGE:
		log_string += ">>"
		fast_debug_message = true
	default:
		return
	}

	err := false
	fatal := false
	log_string += " "
	if !fast_debug_message {
		switch log_level {
		case LOG_LEVEL_DEBUG:
			log_string += "DEBUG:"
		case LOG_LEVEL_WARN:
			log_string += "WARNING:"
		case LOG_LEVEL_ERROR:
			log_string += "ERROR:"
			err = true
		case LOG_LEVEL_FATAL:
			log_string += "FATAL:"
			fatal = true
		default:
			return
		}
	}

	for _, msg := range message {
		log_string += " "
		log_string += fmt.Sprint(msg)
	}

	if fatal {
		fmt.Printf("\n")
		debug.PrintStack()
		log.Fatalln(log_string)
	} else if err {
		fmt.Printf("\n")
		log.Println(log_string)
		fmt.Printf("\n")
	} else {
		log.Println(log_string)
	}
}

func log_debug_msg(message ...interface{}) {
	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_FAST_DEBUG_MESSAGE, message...)
}

func log_err_if_not_nil(err error) bool {
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, err)
		return true
	}
	return false
}

type vec2 struct {
	X float32
	Y float32
}

type ivec2 struct {
	X int
	Y int
}

func convert_rgb24_to_rgba(color uint32) sdl.Color {
	return sdl.Color{
		// 0x000000rr & 0xff = red: 0xrr
		R: uint8((color >> 16) & 0xff),
		// 0x0000rrgg & 0xff = green: 0xgg
		G: uint8((color >> 8) & 0xff),
		// 0x00rrggbb & 0xff = blue: 0xbb
		B: uint8(color & 0xff),
		A: 255,
	}
}

func convert_rgba_to_rgb24(color sdl.Color) uint32 {
	// 0x00000000
	rgb24 := uint32(color.R)
	// 0x000000rr
	rgb24 = (rgb24 << 8) | uint32(color.G)
	// 0x0000rrgg
	rgb24 = (rgb24 << 8) | uint32(color.B)
	// 0x00rrggbb
	return rgb24
}

func is_color_black(color sdl.Color) bool {
	return color.R == 0 && color.G == 0 && color.B == 0
}

func has_flag_u16(val, flag uint16) bool {
	return val&flag != 0
}

func is_digit(char rune) bool {
	return char >= '0' && char <= '9'
}

func measure_execution_time(name string) func() {
	now := time.Now()
	return func() {
		elapsed := time.Since(now)
		if elapsed > time.Millisecond*10 {
			log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "Function", name, "takes", elapsed)
		}
	}
}
