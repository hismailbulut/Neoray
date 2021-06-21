package main

import (
	"fmt"
	"sync"

	"github.com/neovim/go-client/nvim"
)

const (
	OPTION_CURSOR_ANIM  string = "neoray_cursor_animation_time"
	OPTION_TRANSPARENCY string = "neoray_framebuffer_transparency"
	OPTION_TARGET_TPS   string = "neoray_target_ticks_per_second"
	OPTION_POPUP_MENU   string = "neoray_popup_menu_enabled"
	OPTION_WINDOW_STATE string = "neoray_window_startup_state"

	// Keybindings
	OPTION_KEY_FULLSCRN string = "neoray_key_toggle_fullscreen"
	OPTION_ZOOMIN_KEY   string = "neoray_key_increase_fontsize"
	OPTION_ZOOMOUT_KEY  string = "neoray_key_decrease_fontsize"
)

type NvimProcess struct {
	handle       *nvim.Nvim
	update_mutex *sync.Mutex
	update_stack [][][]interface{}
}

func CreateNvimProcess() NvimProcess {
	defer measure_execution_time("CreateNvimProcess")()
	proc := NvimProcess{
		update_mutex: &sync.Mutex{},
		update_stack: make([][][]interface{}, 0),
	}

	args := []string{"--embed"}
	args = append(args, EditorArgs.nvimArgs...)

	nv, err := nvim.NewChildProcess(nvim.ChildProcessArgs(args...))
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NVIM, err)
	}

	proc.handle = nv
	proc.requestApiInfo()
	proc.introduce()
	proc.initScripts()

	log_message(LOG_LEVEL_TRACE, LOG_TYPE_NVIM, "Neovim child process created.")

	return proc
}

func (proc *NvimProcess) requestApiInfo() {
	_, err := proc.handle.APIInfo()
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to get api information:", err)
		return
	}
}

func (proc *NvimProcess) introduce() {
	// Short name for the connected client
	name := TITLE
	// Dictionary describing the version
	version := &nvim.ClientVersion{
		Major:      VERSION_MAJOR,
		Minor:      VERSION_MINOR,
		Patch:      VERSION_PATCH,
		Prerelease: "dev",
		Commit:     "main",
	}
	// Client type
	typ := "ui"
	// Builtin methods in the client
	methods := make(map[string]*nvim.ClientMethod, 0)
	// Arbitrary string:string map of informal client properties
	attributes := make(nvim.ClientAttributes, 1)
	attributes["website"] = WEBPAGE
	attributes["license"] = LICENSE
	err := proc.handle.SetClientInfo(name, version, typ, methods, attributes)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to set client information:", err)
	}
}

func (proc *NvimProcess) initScripts() {
	// Set a variable that users can define their neoray specific customization.
	proc.handle.SetVar("neoray", 1)
}

func (proc *NvimProcess) StartUI() {
	defer measure_execution_time("StartUI")()

	options := make(map[string]interface{})
	options["rgb"] = true
	options["ext_linegrid"] = true

	proc.handle.AttachUI(EditorSingleton.columnCount, EditorSingleton.rowCount, options)

	proc.handle.RegisterHandler("redraw",
		func(updates ...[]interface{}) {
			proc.update_mutex.Lock()
			defer proc.update_mutex.Unlock()
			proc.update_stack = append(proc.update_stack, updates)
		})

	go func() {
		if err := proc.handle.Serve(); err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Neovim child process closed with errors:", err)
			return
		}
		log_message(LOG_LEVEL_TRACE, LOG_TYPE_NVIM, "Neovim child process closed.")
		EditorSingleton.quitRequestedChan <- true
	}()

	log_message(LOG_LEVEL_TRACE, LOG_TYPE_NVIM,
		"UI Connected. Rows:", EditorSingleton.rowCount, "Cols:", EditorSingleton.columnCount)

	proc.requestOptions()
}

func (proc *NvimProcess) requestOptions() {
	var f32var float32
	var intvar int
	var strvar string
	if proc.handle.Var(OPTION_CURSOR_ANIM, &f32var) == nil {
		EditorSingleton.cursor.animLifetime = f32var
	}
	if proc.handle.Var(OPTION_TRANSPARENCY, &f32var) == nil {
		EditorSingleton.framebufferTransparency = f32var
	}
	if proc.handle.Var(OPTION_TARGET_TPS, &intvar) == nil {
		if intvar > 0 {
			EditorSingleton.targetTPS = intvar
		}
	}
	if proc.handle.Var(OPTION_POPUP_MENU, &intvar) == nil {
		if intvar == 0 {
			popupMenuEnabled = false
		} else {
			popupMenuEnabled = true
		}
	}
	if proc.handle.Var(OPTION_KEY_FULLSCRN, &strvar) == nil {
		toggleFullscreenKey = strvar
	}
	if proc.handle.Var(OPTION_ZOOMIN_KEY, &strvar) == nil {
		zoomInKey = strvar
	}
	if proc.handle.Var(OPTION_ZOOMOUT_KEY, &strvar) == nil {
		zoomOutKey = strvar
	}
	if proc.handle.Var(OPTION_WINDOW_STATE, &strvar) == nil {
		EditorSingleton.window.SetState(strvar)
	}
}

func (proc *NvimProcess) ExecuteVimScript(script string, args ...interface{}) {
	cmd := fmt.Sprintf(script, args...) + "\n"
	if err := proc.handle.Command(cmd); err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to execute vimscript:", err)
	}
}

func (proc *NvimProcess) Mode() string {
	mode, err := proc.handle.Mode()
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to get mode name:", mode)
		return ""
	}
	return mode.Mode
}

func (proc *NvimProcess) Echo(text string, args ...interface{}) {
	formatted := fmt.Sprintf(text, args...)
	proc.ExecuteVimScript(":echomsg '%s'", formatted)
}

func (proc *NvimProcess) EchoError(text string, args ...interface{}) {
	formatted := fmt.Sprintf(text, args...)
	proc.handle.WritelnErr(formatted)
}

func (proc *NvimProcess) Cut() {
	switch proc.Mode() {
	case "n":
		proc.FeedKeys("v\"*yx")
		break
	case "v":
		proc.FeedKeys("\"*ygvd")
		break
	}
}

func (proc *NvimProcess) Copy() {
	switch proc.Mode() {
	case "n":
		proc.FeedKeys("v\"*y")
		break
	case "v":
		proc.FeedKeys("\"*y")
		break
	}
}

func (proc *NvimProcess) WriteAtCursor(str string) {
	switch proc.Mode() {
	case "i":
		proc.Input(str)
		break
	case "n":
		proc.FeedKeys("i")
		proc.Input(str)
		proc.FeedKeys("<ESC>")
		break
	case "v":
		proc.FeedKeys("<ESC>i")
		proc.Input(str)
		proc.FeedKeys("<ESC>gv")
		break
	}
}

func (proc *NvimProcess) SelectAll() {
	switch proc.Mode() {
	case "i", "v":
		proc.FeedKeys("<ESC>ggVG")
		break
	case "n":
		proc.FeedKeys("ggVG")
		break
	}
}

func (proc *NvimProcess) OpenFile(file string) {
	proc.ExecuteVimScript(":edit %s", file)
}

func (proc *NvimProcess) GotoLine(line int) {
	proc.ExecuteVimScript("call cursor(%d, 0)", line)
}

func (proc *NvimProcess) GotoColumn(col int) {
	proc.ExecuteVimScript("call cursor(0, %d)", col)
}

func (proc *NvimProcess) FeedKeys(keys string) {
	keycode, err := proc.handle.ReplaceTermcodes(keys, true, true, true)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to replace termcodes:", err)
		return
	}
	err = proc.handle.FeedKeys(keycode, "m", true)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to feed keys:", err)
	}
}

func (proc *NvimProcess) Input(keycode string) {
	written, err := proc.handle.Input(keycode)
	if err != nil {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send input keys:", err)
	}
	if written != len(keycode) {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send some keys.")
	}
}

func (proc *NvimProcess) InputMouse(button, action, modifier string, grid, row, column int) {
	err := proc.handle.InputMouse(button, action, modifier, grid, row, column)
	if err != nil {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send mouse input:", err)
	}
}

// Call CalculateCellSize before this function.
func (proc *NvimProcess) RequestResize() {
	EditorSingleton.calculateCellCount()
	proc.handle.TryResizeUI(EditorSingleton.columnCount, EditorSingleton.rowCount)
	EditorSingleton.waitingResize = true
}

func (proc *NvimProcess) Close() {
	proc.handle.Close()
}
