package main

import (
	"fmt"
	"reflect"
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

	args := append([]string{"--embed"}, EditorParsedArgs.others...)

	nv, err := nvim.NewChildProcess(
		nvim.ChildProcessArgs(args...),
		nvim.ChildProcessCommand(EditorParsedArgs.execPath))
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NVIM, "Failed to start neovim instance:", err)
	}

	log_message(LOG_LEVEL_TRACE, LOG_TYPE_NVIM,
		"Neovim started with command:", EditorParsedArgs.execPath, mergeStringArray(args))

	proc.handle = nv

	proc.requestApiInfo()
	proc.introduce()

	// Set a variable that users can define their neoray specific customization.
	proc.handle.SetVar("neoray", 1)

	return proc
}

func (proc *NvimProcess) requestApiInfo() {
	defer measure_execution_time()()

	info, err := proc.handle.APIInfo()
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to get api information:", err)
		return
	}
	// Check the version.
	// info[1] is dictionary of infos and it has a key named 'version',
	// and this key contains a map which has major, minor and patch informations.
	vInfo := reflect.ValueOf(info[1]).MapIndex(reflect.ValueOf("version")).Elem()
	vMajor := vInfo.MapIndex(reflect.ValueOf("major")).Elem().Convert(t_int).Int()
	vMinor := vInfo.MapIndex(reflect.ValueOf("minor")).Elem().Convert(t_int).Int()
	vPatch := vInfo.MapIndex(reflect.ValueOf("patch")).Elem().Convert(t_int).Int()

	if vMinor < 4 {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NVIM,
			"Neoray needs at least 0.4.0 version of neovim. Please update your neovim to a newer version.")
	}

	vStr := fmt.Sprintf("v%d.%d.%d", vMajor, vMinor, vPatch)
	log_message(LOG_LEVEL_TRACE, LOG_TYPE_NVIM, "Neovim version", vStr)
}

func (proc *NvimProcess) introduce() {
	defer measure_execution_time()()

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

// Returns options which implemented in vim but not implemented in neovim.
// Like 'mousehide'. Don't use this as long as the nvim_get_option is working.
func (proc *NvimProcess) getUnimplementedOption(name string) interface{} {
	defer measure_execution_time()()

	eventName := "optc_" + name
	var opt interface{}
	okc := make(chan bool)
	proc.handle.RegisterHandler(eventName, func(val interface{}) {
		opt = val
		okc <- true
	})
	ok := proc.executeVimScript("call rpcnotify(%d, \"%s\", &%s)",
		proc.handle.ChannelID(), eventName, name)
	if ok {
		<-okc
	}
	proc.handle.Unsubscribe(eventName)
	return opt
}

func (proc *NvimProcess) startUI(columns, rows int) {
	defer measure_execution_time()()

	options := make(map[string]interface{})
	options["rgb"] = true
	options["ext_linegrid"] = true

	proc.handle.AttachUI(columns, rows, options)

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
		EditorSingleton.quitRequested.Set(true)
	}()
}

func (proc *NvimProcess) requestVariables() {
	defer measure_execution_time()()
	proc.handle.Var(OPTION_CURSOR_ANIM, &EditorSingleton.options.cursorAnimTime)
	proc.handle.Var(OPTION_TRANSPARENCY, &EditorSingleton.options.transparency)
	proc.handle.Var(OPTION_TARGET_TPS, &EditorSingleton.options.targetTPS)
	proc.handle.Var(OPTION_POPUP_MENU, &EditorSingleton.options.popupMenuEnabled)
	proc.handle.Var(OPTION_KEY_FULLSCRN, &EditorSingleton.options.keyToggleFullscreen)
	proc.handle.Var(OPTION_KEY_ZOOMIN, &EditorSingleton.options.keyIncreaseFontSize)
	proc.handle.Var(OPTION_KEY_ZOOMOUT, &EditorSingleton.options.keyDecreaseFontSize)
	var state string
	if proc.handle.Var(OPTION_WINDOW_STATE, &state) == nil {
		EditorSingleton.window.setState(state)
	}
	EditorSingleton.options.mouseHide = boolFromInterface(proc.getUnimplementedOption("mousehide"))
}

func (proc *NvimProcess) executeVimScript(format string, args ...interface{}) bool {
	cmd := fmt.Sprintf(format, args...)
	log_debug("Executing script: [", cmd, "]")
	err := proc.handle.Command(cmd)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM,
			"Failed to execute vimscript: [", cmd, "] err:", err)
		return false
	}
	return true
}

func (proc *NvimProcess) currentMode() string {
	mode, err := proc.handle.Mode()
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to get current mode name:", err)
		return ""
	}
	return mode.Mode
}

func (proc *NvimProcess) echoMsg(format string, args ...interface{}) {
	formatted := fmt.Sprintf(format, args...)
	proc.executeVimScript("echomsg '%s'", formatted)
}

func (proc *NvimProcess) echoErr(format string, args ...interface{}) {
	formatted := fmt.Sprintf(format, args...)
	proc.handle.WritelnErr(formatted)
}

func (proc *NvimProcess) getRegister(register string) string {
	var content string
	err := proc.handle.Call("getreg", &content, register)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Api call getreg() failed:", err)
	}
	return content
}

// This function cuts current selected text and returns the content.
// Not updates clipboard on every system.
func (proc *NvimProcess) cutSelected() string {
	switch proc.currentMode() {
	case "v", "V":
		proc.feedKeys("\"*ygvd")
		return proc.getRegister("*")
	default:
		return ""
	}
}

// This function copies current selected text and returns the content.
// Not updates clipboard on every system.
func (proc *NvimProcess) copySelected() string {
	switch proc.currentMode() {
	case "v", "V":
		proc.feedKeys("\"*y")
		return proc.getRegister("*")
	default:
		return ""
	}
}

// Pastes text to cursor.
func (proc *NvimProcess) paste(str string) {
	err := proc.handle.Call("nvim_paste", nil, str, true, -1)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Api call nvim_paste() failed:", err)
	}
}

// TODO: We need to check if this buffer is normal buffer.
// Executing this function in non normal buffers may be dangerous.
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
	log_debug("Open file:", file)
	proc.executeVimScript("edit %s", file)
}

func (proc *NvimProcess) gotoLine(line int) {
	log_debug("Goto Line:", line)
	proc.handle.Call("cursor", nil, line, 0)
}

func (proc *NvimProcess) gotoColumn(col int) {
	log_debug("Goto Column:", col)
	proc.handle.Call("cursor", nil, 0, col)
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
	if EditorSingleton.calculateCellCount() {
		log_debug("Resize request. Rows:", EditorSingleton.rowCount, "Columns:", EditorSingleton.columnCount)
		err := proc.handle.TryResizeUI(EditorSingleton.columnCount, EditorSingleton.rowCount)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to send resize request:", err)
			return
		}
		EditorSingleton.waitingResize = true
	}
}

func (proc *NvimProcess) Close() {
	proc.handle.Close()
}
