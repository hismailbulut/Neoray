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
	OPTION_TRANSPARENCY = "neoray_background_transparency"
	OPTION_TARGET_TPS   = "neoray_target_ticks_per_second"
	OPTION_CONTEXT_MENU = "neoray_context_menu_enabled"
	OPTION_WINDOW_STATE = "neoray_window_startup_state"
	OPTION_WINDOW_SIZE  = "neoray_window_startup_size"
	// Keybindings
	OPTION_KEY_FULLSCRN = "neoray_key_toggle_fullscreen"
	OPTION_KEY_ZOOMIN   = "neoray_key_increase_fontsize"
	OPTION_KEY_ZOOMOUT  = "neoray_key_decrease_fontsize"
)

type NvimProcess struct {
	handle       *nvim.Nvim
	update_mutex *sync.Mutex
	update_stack [][][]interface{}
	timer        float64
}

func CreateNvimProcess() NvimProcess {
	defer measure_execution_time()()

	proc := NvimProcess{
		update_mutex: &sync.Mutex{},
		update_stack: make([][][]interface{}, 0),
	}

	args := append([]string{"--embed"}, editorParsedArgs.others...)

	nv, err := nvim.NewChildProcess(
		nvim.ChildProcessArgs(args...),
		nvim.ChildProcessCommand(editorParsedArgs.execPath))
	if err != nil {
		logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NVIM, "Failed to start neovim instance:", err)
	}
	proc.handle = nv

	logMessage(LOG_LEVEL_DEBUG, LOG_TYPE_NVIM,
		"Neovim started with command:", editorParsedArgs.execPath, mergeStringArray(args))

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
		// Maybe fatal?
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to get api information:", err)
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
		logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NVIM,
			"Neoray needs at least 0.4.0 version of neovim. Please update your neovim to a newer version.")
	}

	vStr := fmt.Sprintf("%d.%d.%d", vMajor, vMinor, vPatch)
	logMessage(LOG_LEVEL_TRACE, LOG_TYPE_NVIM, "Neovim version", vStr)
}

func (proc *NvimProcess) introduce() {
	// Short name for the connected client
	name := TITLE
	// Dictionary describing the version
	version := &nvim.ClientVersion{
		Major: VERSION_MAJOR,
		Minor: VERSION_MINOR,
		Patch: VERSION_PATCH,
		// Commit: "",
	}
	if isDebugBuild() {
		version.Prerelease = "dev"
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
		// Maybe fatal?
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to set client information:", err)
	}
}

func (proc *NvimProcess) startUI() {
	options := make(map[string]interface{})
	options["rgb"] = true
	options["ext_linegrid"] = true

	if editorParsedArgs.multiGrid {
		options["ext_multigrid"] = true
		logDebug("Multigrid enabled.")
	}

	// TODO: calculate size
	if err := proc.handle.AttachUI(60, 20, options); err != nil {
		logMessage(LOG_LEVEL_FATAL, LOG_TYPE_NVIM, "Attaching ui failed:", err)
	}

	proc.handle.RegisterHandler("redraw",
		func(updates ...[]interface{}) {
			proc.update_mutex.Lock()
			defer proc.update_mutex.Unlock()
			proc.update_stack = append(proc.update_stack, updates)
		})

	go func() {
		if err := proc.handle.Serve(); err != nil {
			logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Neovim child process closed with errors:", err)
			return
		}
		logMessage(LOG_LEVEL_TRACE, LOG_TYPE_NVIM, "Neovim child process closed.")
		singleton.quitRequested <- true
	}()

	logMessage(LOG_LEVEL_DEBUG, LOG_TYPE_NVIM, "Attached to neovim as an ui client.")
}

func (proc *NvimProcess) update() {
	// We are checking variables in every second otherwise performance drops
	// TODO: This is an expensive operation and we need to get rid of this.
	proc.timer += singleton.time.delta
	if proc.timer >= 0 {
		proc.requestRuntimeVariables()
		proc.timer -= 1
	}
}

func (proc *NvimProcess) requestStartupVariables() {
	defer measure_execution_time()()
	var s string
	// Window startup size
	if proc.handle.Var(OPTION_WINDOW_SIZE, &s) == nil {
		logDebugMsg(LOG_TYPE_NVIM, "Option", OPTION_WINDOW_SIZE, "is", s)
		// Parse the string
		width, height, ok := parseSizeString(s)
		if ok {
			singleton.window.setSize(width, height, true)
		} else {
			logMessage(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Could not parse size value:", s)
		}
	}
	// Window startup state
	if proc.handle.Var(OPTION_WINDOW_STATE, &s) == nil {
		logDebugMsg(LOG_TYPE_NVIM, "Option", OPTION_WINDOW_STATE, "is", s)
		singleton.window.setState(s)
	}
	// built-in 'mousehide' option
	singleton.options.mouseHide = boolFromInterface(proc.getUnimplementedOption("mousehide"))
}

func (proc *NvimProcess) requestRuntimeVariables() {
	defer measure_execution_time()()
	options := &singleton.options
	var s string
	var f float32
	var i int
	var b bool
	if proc.handle.Var(OPTION_CURSOR_ANIM, &f) == nil {
		if f != options.cursorAnimTime {
			logDebugMsg(LOG_TYPE_NVIM, "Option", OPTION_CURSOR_ANIM, "is", f)
			options.cursorAnimTime = f
		}
	}
	if proc.handle.Var(OPTION_TRANSPARENCY, &f) == nil {
		if f != options.transparency {
			logDebugMsg(LOG_TYPE_NVIM, "Option", OPTION_TRANSPARENCY, "is", f)
			options.transparency = f
			singleton.fullDraw()
		}
	}
	if proc.handle.Var(OPTION_TARGET_TPS, &i) == nil {
		if i != options.targetTPS {
			logDebugMsg(LOG_TYPE_NVIM, "Option", OPTION_TARGET_TPS, "is", i)
			options.targetTPS = i
			singleton.resetTicker()
		}
	}
	if proc.handle.Var(OPTION_CONTEXT_MENU, &b) == nil {
		if b != options.contextMenuEnabled {
			logDebugMsg(LOG_TYPE_NVIM, "Option", OPTION_CONTEXT_MENU, "is", b)
			options.contextMenuEnabled = b
		}
	}
	if proc.handle.Var(OPTION_KEY_FULLSCRN, &s) == nil {
		if s != options.keyToggleFullscreen {
			logDebugMsg(LOG_TYPE_NVIM, "Option", OPTION_KEY_FULLSCRN, "is", s)
			options.keyToggleFullscreen = s
		}
	}
	if proc.handle.Var(OPTION_KEY_ZOOMIN, &s) == nil {
		if s != options.keyIncreaseFontSize {
			logDebugMsg(LOG_TYPE_NVIM, "Option", OPTION_KEY_ZOOMIN, "is", s)
			options.keyIncreaseFontSize = s
		}
	}
	if proc.handle.Var(OPTION_KEY_ZOOMOUT, &s) == nil {
		if s != options.keyDecreaseFontSize {
			logDebugMsg(LOG_TYPE_NVIM, "Option", OPTION_KEY_ZOOMOUT, "is", s)
			options.keyDecreaseFontSize = s
		}
	}
}

// Returns options which implemented in vim but not implemented in neovim.
// Like 'mousehide'. Don't use this as long as the nvim_get_option is working.
func (proc *NvimProcess) getUnimplementedOption(name string) interface{} {
	defer measure_execution_time()()
	eventName := "opt_" + name
	var opt interface{}
	ch := make(chan bool)
	defer close(ch)
	proc.handle.RegisterHandler(eventName, func(val interface{}) {
		opt = val
		ch <- true
	})
	defer proc.handle.Unsubscribe(eventName)
	ok := proc.executeVimScript("call rpcnotify(%d, \"%s\", &%s)",
		proc.handle.ChannelID(), eventName, name)
	if ok {
		<-ch
	}
	return opt
}

func (proc *NvimProcess) executeVimScript(format string, args ...interface{}) bool {
	cmd := fmt.Sprintf(format, args...)
	logMessage(LOG_LEVEL_DEBUG, LOG_TYPE_NVIM, "Executing script: [", cmd, "]")
	err := proc.handle.Command(cmd)
	if err != nil {
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to execute vimscript: [", cmd, "] err:", err)
		return false
	}
	return true
}

func (proc *NvimProcess) currentMode() string {
	mode, err := proc.handle.Mode()
	if err != nil {
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to get current mode name:", err)
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
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Api call getreg() failed:", err)
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
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Api call nvim_paste() failed:", err)
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
	proc.executeVimScript("edit %s", file)
}

func (proc *NvimProcess) gotoLine(line int) {
	logDebug("Goto Line:", line)
	proc.handle.Call("cursor", nil, line, 0)
}

func (proc *NvimProcess) gotoColumn(col int) {
	logDebug("Goto Column:", col)
	proc.handle.Call("cursor", nil, 0, col)
}

func (proc *NvimProcess) feedKeys(keys string) {
	keycode, err := proc.handle.ReplaceTermcodes(keys, true, true, true)
	if err != nil {
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to replace termcodes:", err)
		return
	}
	err = proc.handle.FeedKeys(keycode, "m", true)
	if err != nil {
		logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to feed keys:", err)
	}
}

func (proc *NvimProcess) input(keycode string) {
	written, err := proc.handle.Input(keycode)
	if err != nil {
		logMessage(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send input keys:", err)
	}
	if written != len(keycode) {
		logMessage(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send some keys.")
	}
}

func (proc *NvimProcess) inputMouse(button, action, modifier string, grid, row, column int) {
	err := proc.handle.InputMouse(button, action, modifier, grid, row, column)
	if err != nil {
		logMessage(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send mouse input:", err)
	}
}

func (proc *NvimProcess) requestResize(rows, cols int) {
	if rows > 0 && cols > 0 {
		err := proc.handle.TryResizeUI(cols, rows)
		if err != nil {
			logMessage(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to send resize request:", err)
			return
		}
	}
}

func (proc *NvimProcess) Close() {
	err := proc.handle.Close()
	if err != nil {
		logMessage(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to close neovim child process:", err)
	}
}
