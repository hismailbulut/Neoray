package main

import (
	"runtime"
	// "time"

	rl "github.com/chunqian/go-raylib/raylib"
)

const font_name = "Cousine"

func init() {
	runtime.LockOSThread()
}

func main() {

	nvim_proc := CreateProcess()

	window := CreateAndShow(800, 600, "window", font_name, 18)
	defer window.Close()

	nvim_proc.StartUI(&window)
	defer nvim_proc.Close()

	rl.SetTargetFPS(60)
	for !rl.WindowShouldClose() {
		window.Update(nvim_proc)
		window.Render()
		// time.Sleep(time.Millisecond * 2)
	}
}
