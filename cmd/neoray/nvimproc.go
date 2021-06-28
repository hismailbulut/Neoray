package main

import (
	"fmt"
	"sync"

	"github.com/neovim/go-client/nvim"
)

const (
	// Options
	OPTION_CURSOR_ANIM  = "neoray_cursor_animation_time"
	OPTION_TRANSPARENCY = "neoray_framebuffer_transparency"
	OPTION_TARGET_TPS   = "neoray_target_ticks_per_second"
	OPTION_POPUP_MENU   = "neoray_popup_menu_enabled"
	OPTION_WINDOW_STATE = "neoray_window_startup_state"
	// Keybindings
	OPTION_KEY_FULLSCRN = "neoray_key_toggle_fullscreen"
	OPTION_KEY_ZOOMIN   = "neoray_key_increase_fontsize"
	OPTION_KEY_ZOOMOUT  = "neoray_key_decrease_fontsize"
)

type NvimProcess struct {
	handle       *nvim.Nvim
	update_mutex *sync.Mutex
	update_stack [][][]interface{}
}

func CreateNvimProcess() NvimProcess {
	defer measure_execution_time()()
	proc := NvimProcess{
		update_mutex: &sync.Mutex{},
		update_stack: make([][][]interface{}, 0),
	}

	args := []string{"--embed"}
	args = append(args, EditorParsedArgs.others...)

	log_debug("Neovim args:", args)

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

func (proc *NvimProcess) startUI() {
	defer measure_execution_time()()

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

	proc.requestVariables()
}

func (proc *NvimProcess) requestVariables() {
	proc.getVimVar(OPTION_CURSOR_ANIM, &EditorSingleton.cursor.animLifetime)
	proc.getVimVar(OPTION_TRANSPARENCY, &EditorSingleton.framebufferTransparency)
	proc.getVimVar(OPTION_TARGET_TPS, &EditorSingleton.targetTPS)
	proc.getVimVar(OPTION_POPUP_MENU, &popupMenuEnabled)
	var state string
	if proc.getVimVar(OPTION_WINDOW_STATE, &state) {
		EditorSingleton.window.SetState(state)
	}
	proc.getVimVar(OPTION_KEY_FULLSCRN, &KEYToggleFullscreen)
	proc.getVimVar(OPTION_KEY_ZOOMIN, &KEYIncreaseFontSize)
	proc.getVimVar(OPTION_KEY_ZOOMOUT, &KEYDecreaseFontSize)
}

func (proc *NvimProcess) getVimVar(name string, variable interface{}) bool {
	if err := proc.handle.Var(name, variable); err != nil {
		log_debug("Failed to get variable:", err)
		return false
	}
	return true
}

func (proc *NvimProcess) executeVimScript(script string, args ...interface{}) {
	cmd := fmt.Sprintf(script, args...)
	err := proc.handle.Command(cmd)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM,
			"Failed to execute vimscript: [", cmd, "] err:", err)
	}
}

func (proc *NvimProcess) currentMode() string {
	mode, err := proc.handle.Mode()
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to get mode name:", err)
		return ""
	}
	return mode.Mode
}

func (proc *NvimProcess) echoMsg(text string, args ...interface{}) {
	formatted := fmt.Sprintf(text, args...)
	proc.executeVimScript(":echomsg '%s'", formatted)
}

func (proc *NvimProcess) echoErr(text string, args ...interface{}) {
	formatted := fmt.Sprintf(text, args...)
	proc.handle.WritelnErr(formatted)
}

// TODO: Clipboard are not working on some linux systems.
// We need to do cut and copy commands ourselves.
func (proc *NvimProcess) cutSelected() {
	switch proc.currentMode() {
	case "n":
		proc.feedKeys("v\"*yx")
		break
	case "v":
		proc.feedKeys("\"*ygvd")
		break
	}
}

func (proc *NvimProcess) copySelected() {
	switch proc.currentMode() {
	case "n":
		proc.feedKeys("v\"*y")
		break
	case "v":
		proc.feedKeys("\"*y")
		break
	}
}

func (proc *NvimProcess) paste(str string) {
	err := proc.handle.Call("nvim_paste", nil, str, true, -1)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Api call nvim_paste() failed:", err)
	}
}

func (proc *NvimProcess) selectAll() {
	switch proc.currentMode() {
	case "i", "v":
		proc.feedKeys("<ESC>ggVG")
		break
	case "n":
		proc.feedKeys("ggVG")
		break
	}
}

func (proc *NvimProcess) openFile(file string) {
	proc.executeVimScript(":edit %s", file)
}

func (proc *NvimProcess) gotoLine(line int) {
	proc.executeVimScript("call cursor(%d, 0)", line)
}

func (proc *NvimProcess) gotoColumn(col int) {
	proc.executeVimScript("call cursor(0, %d)", col)
}

func (proc *NvimProcess) feedKeys(keys string) {
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

func (proc *NvimProcess) input(keycode string) {
	written, err := proc.handle.Input(keycode)
	if err != nil {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send input keys:", err)
	}
	if written != len(keycode) {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send some keys.")
	}
}

func (proc *NvimProcess) inputMouse(button, action, modifier string, grid, row, column int) {
	err := proc.handle.InputMouse(button, action, modifier, grid, row, column)
	if err != nil {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send mouse input:", err)
	}
}

func (proc *NvimProcess) requestResize() {
	EditorSingleton.calculateCellCount()
	proc.handle.TryResizeUI(EditorSingleton.columnCount, EditorSingleton.rowCount)
	EditorSingleton.waitingResize = true
}

func (proc *NvimProcess) Close() {
	proc.handle.Close()
}
