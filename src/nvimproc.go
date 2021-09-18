package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/neovim/go-client/nvim"
)

const (
	// All options here are deprecated and will be removed soon
	OPTION_CURSOR_ANIM_DEP  = "neoray_cursor_animation_time"
	OPTION_TRANSPARENCY_DEP = "neoray_background_transparency"
	OPTION_TARGET_TPS_DEP   = "neoray_target_ticks_per_second"
	OPTION_CONTEXT_MENU_DEP = "neoray_context_menu_enabled"
	OPTION_WINDOW_STATE_DEP = "neoray_window_startup_state"
	OPTION_WINDOW_SIZE_DEP  = "neoray_window_startup_size"
	OPTION_KEY_FULLSCRN_DEP = "neoray_key_toggle_fullscreen"
	OPTION_KEY_ZOOMIN_DEP   = "neoray_key_increase_fontsize"
	OPTION_KEY_ZOOMOUT_DEP  = "neoray_key_decrease_fontsize"
)

const (
	// New options
	OPTION_CURSOR_ANIM    = "CursorAnimTime"
	OPTION_TRANSPARENCY   = "Transparency"
	OPTION_TARGET_TPS     = "TargetTPS"
	OPTION_CONTEXT_MENU   = "ContextMenuOn"
	OPTION_CONTEXT_BUTTON = "ContextButton"
	OPTION_BOX_DRAWING    = "BoxDrawingOn"
	OPTION_WINDOW_STATE   = "WindowState"
	OPTION_WINDOW_SIZE    = "WindowSize"
	// Keybindings
	OPTION_KEY_FULLSCRN = "KeyFullscreen"
	OPTION_KEY_ZOOMIN   = "KeyZoomIn"
	OPTION_KEY_ZOOMOUT  = "KeyZoomOut"
)

// Add all options here
var OptionsList = []string{
	OPTION_CURSOR_ANIM,
	OPTION_TRANSPARENCY,
	OPTION_TARGET_TPS,
	OPTION_CONTEXT_MENU,
	OPTION_CONTEXT_BUTTON,
	OPTION_BOX_DRAWING,
	OPTION_WINDOW_STATE,
	OPTION_WINDOW_SIZE,
	OPTION_KEY_FULLSCRN,
	OPTION_KEY_ZOOMIN,
	OPTION_KEY_ZOOMOUT,
}

var NeorayOptionSet_Source string = `
function NeorayOptionSet(...)
	if a:0 < 2
		echoerr 'NeoraySet needs at least 2 arguments.'
		return
	endif
	call call(function("rpcnotify"), [CHANID, "NeorayOptionSet"] + a:000)
endfunction

function NeorayCompletion(A, L, P)
	return OPTIONLIST
endfunction

command -nargs=+ -complete=customlist,NeorayCompletion NeoraySet call NeorayOptionSet(<f-args>)
`

type NvimProcess struct {
	handle        *nvim.Nvim
	eventReceived AtomicBool
	eventMutex    *sync.Mutex
	eventStack    [][][]interface{}
	optionChanged AtomicBool
	optionMutex   *sync.Mutex
	optionStack   [][]string
}

func CreateNvimProcess() NvimProcess {
	defer measure_execution_time()()

	proc := NvimProcess{
		eventMutex:  &sync.Mutex{},
		eventStack:  make([][][]interface{}, 0),
		optionMutex: &sync.Mutex{},
		optionStack: make([][]string, 0),
	}

	args := append([]string{"--embed"}, editorParsedArgs.others...)

	var err error
	proc.handle, err = nvim.NewChildProcess(
		nvim.ChildProcessArgs(args...),
		nvim.ChildProcessCommand(editorParsedArgs.execPath))
	if err != nil {
		logMessage(LEVEL_FATAL, TYPE_NVIM, "Failed to start neovim instance:", err)
	}

	logMessage(LEVEL_DEBUG, TYPE_NVIM,
		"Neovim started with command:", editorParsedArgs.execPath, mergeStringArray(args))

	return proc
}

// We are initializing some callback functions here because CreateNvimProcess
// copies actual process struct and we lost pointer of it if these functions
// are called in CreateNvimProcess
func (proc *NvimProcess) init() {
	proc.requestApiInfo()
	proc.registerScripts()
}

func (proc *NvimProcess) requestApiInfo() {
	defer measure_execution_time()()

	info, err := proc.handle.APIInfo()
	if err != nil {
		logMessage(LEVEL_FATAL, TYPE_NVIM, "Failed to get api information:", err)
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
		logMessage(LEVEL_FATAL, TYPE_NVIM,
			"Neoray needs at least 0.4.0 version of neovim. Please update your neovim to a newer version.")
	}

	vStr := fmt.Sprintf("%d.%d.%d", vMajor, vMinor, vPatch)
	logMessage(LEVEL_TRACE, TYPE_NVIM, "Neovim version", vStr)
}

func (proc *NvimProcess) registerScripts() {
	// Set a variable that users can define their neoray specific customization.
	proc.handle.SetVar("neoray", 1)
	// Replace channel ids in the template
	source := strings.ReplaceAll(NeorayOptionSet_Source, "CHANID", strconv.Itoa(proc.handle.ChannelID()))
	// Create option list string
	listStr := "["
	for i := 0; i < len(OptionsList); i++ {
		listStr += "'" + OptionsList[i] + "'"
		if i < len(OptionsList)-1 {
			listStr += ","
		}
	}
	listStr += "]"
	// Replace list in source
	source = strings.Replace(source, "OPTIONLIST", listStr, 1)
	// Trim whitespaces
	source = strings.TrimSpace(source)
	// Execute script
	_, err := proc.handle.Exec(source, false)
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NVIM, "Failed to execute NeorayOptionSet_Source:", err)
		return
	}
	// Register handler
	proc.handle.RegisterHandler("NeorayOptionSet",
		func(args ...string) {
			// arg 0 is the name of the option, others are arguments
			proc.optionMutex.Lock()
			defer proc.optionMutex.Unlock()
			proc.optionStack = append(proc.optionStack, args)
			proc.optionChanged.Set(true)
		})
}

func (proc *NvimProcess) startUI(rows, cols int) {
	defer measure_execution_time()()

	options := map[string]interface{}{
		"rgb":          true,
		"ext_linegrid": true,
	}

	if editorParsedArgs.multiGrid {
		options["ext_multigrid"] = true
		logMessage(LEVEL_DEBUG, TYPE_NVIM, "Multigrid enabled.")
	}

	if err := proc.handle.AttachUI(cols, rows, options); err != nil {
		logMessage(LEVEL_FATAL, TYPE_NVIM, "AttachUI failed:", err)
	}

	proc.handle.RegisterHandler("redraw",
		func(updates ...[]interface{}) {
			proc.eventMutex.Lock()
			defer proc.eventMutex.Unlock()
			proc.eventStack = append(proc.eventStack, updates)
			proc.eventReceived.Set(true)
		})

	go func() {
		if err := proc.handle.Serve(); err != nil {
			logMessage(LEVEL_ERROR, TYPE_NVIM, "Neovim child process closed with errors:", err)
			return
		}
		logMessage(LEVEL_TRACE, TYPE_NVIM, "Neovim child process closed.")
		singleton.quitRequested <- true
	}()

	proc.introduce()
	logMessage(LEVEL_DEBUG, TYPE_NVIM, "Attached to neovim as an ui client.")
}

func (proc *NvimProcess) introduce() {
	// Short name for the connected client
	name := TITLE
	// Dictionary describing the version
	version := &nvim.ClientVersion{
		Major: VERSION_MAJOR,
		Minor: VERSION_MINOR,
		Patch: VERSION_PATCH,
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
		logMessage(LEVEL_FATAL, TYPE_NVIM, "Failed to set client information:", err)
	}
}

func (proc *NvimProcess) update() {
	proc.checkOptions()
}

func (proc *NvimProcess) checkOptions() {
	if proc.optionChanged.Get() {
		proc.optionMutex.Lock()
		defer proc.optionMutex.Unlock()
		for _, opt := range proc.optionStack {
			switch opt[0] {
			case OPTION_CURSOR_ANIM:
				value, err := strconv.ParseFloat(opt[1], 32)
				if err != nil {
					logMessage(LEVEL_WARN, TYPE_NVIM, OPTION_CURSOR_ANIM, "value isn't valid.")
					break
				}
				logMessage(LEVEL_DEBUG, TYPE_NVIM, "Option", OPTION_CURSOR_ANIM, "is", opt[1])
				singleton.options.cursorAnimTime = float32(value)
				break
			case OPTION_TRANSPARENCY:
				value, err := strconv.ParseFloat(opt[1], 32)
				if err != nil {
					logMessage(LEVEL_WARN, TYPE_NVIM, OPTION_TRANSPARENCY, "value isn't valid.")
					break
				}
				logMessage(LEVEL_DEBUG, TYPE_NVIM, "Option", OPTION_TRANSPARENCY, "is", opt[1])
				singleton.options.transparency = f32clamp(float32(value), 0, 1)
				if singleton.mainLoopRunning {
					singleton.fullDraw()
				}
				break
			case OPTION_TARGET_TPS:
				value, err := strconv.Atoi(opt[1])
				if err != nil {
					logMessage(LEVEL_WARN, TYPE_NVIM, OPTION_TARGET_TPS, "value isn't valid.")
					break
				}
				logMessage(LEVEL_DEBUG, TYPE_NVIM, "Option", OPTION_TARGET_TPS, "is", value)
				singleton.options.targetTPS = value
				if singleton.mainLoopRunning {
					singleton.resetTicker()
				}
				break
			case OPTION_CONTEXT_MENU:
				value, err := strconv.ParseBool(opt[1])
				if err != nil {
					logMessage(LEVEL_WARN, TYPE_NVIM, OPTION_CONTEXT_MENU, "value isn't valid.")
					break
				}
				logMessage(LEVEL_DEBUG, TYPE_NVIM, "Option", OPTION_CONTEXT_MENU, "is", value)
				singleton.options.contextMenuEnabled = value
				break
			case OPTION_CONTEXT_BUTTON:
				if len(opt) >= 3 {
					logMessage(LEVEL_DEBUG, TYPE_NVIM, "Option", OPTION_CONTEXT_BUTTON, "name is", opt[1], "and command is", opt[2])
					// NOTE: If we pass opt[2] to execCommand directly, compiler does not copy the string
					// and tries to access deleted slice and generates index out of range.
					cmd := opt[2]
					singleton.contextMenu.AddButton(ContextButton{
						name: opt[1],
						fn:   func() { proc.execCommand(cmd) },
					})
				} else {
					logMessage(LEVEL_WARN, TYPE_NVIM, "Not enough argument for option", OPTION_CONTEXT_BUTTON)
				}
			case OPTION_BOX_DRAWING:
				value, err := strconv.ParseBool(opt[1])
				if err != nil {
					logMessage(LEVEL_WARN, TYPE_NVIM, OPTION_BOX_DRAWING, "value isn't valid.")
					break
				}
				logMessage(LEVEL_DEBUG, TYPE_NVIM, "Option", OPTION_BOX_DRAWING, "is", value)
				singleton.options.boxDrawingEnabled = value
				if singleton.mainLoopRunning {
					singleton.renderer.clearAtlas()
				}
				break
			case OPTION_WINDOW_STATE:
				logMessage(LEVEL_DEBUG, TYPE_NVIM, "Option", OPTION_WINDOW_STATE, "is", opt[1])
				singleton.window.setState(opt[1])
				break
			case OPTION_WINDOW_SIZE:
				width, height, ok := parseSizeString(opt[1])
				if !ok {
					logMessage(LEVEL_WARN, TYPE_NVIM, OPTION_WINDOW_SIZE, "value isn't valid.")
					break
				}
				logMessage(LEVEL_DEBUG, TYPE_NVIM, "Option", OPTION_WINDOW_SIZE, "is", width, height)
				singleton.window.setSize(width, height, true)
				break
			case OPTION_KEY_FULLSCRN:
				logMessage(LEVEL_DEBUG, TYPE_NVIM, "Option", OPTION_KEY_FULLSCRN, "is", opt[1])
				singleton.options.keyToggleFullscreen = opt[1]
				break
			case OPTION_KEY_ZOOMIN:
				logMessage(LEVEL_DEBUG, TYPE_NVIM, "Option", OPTION_KEY_ZOOMIN, "is", opt[1])
				singleton.options.keyIncreaseFontSize = opt[1]
				break
			case OPTION_KEY_ZOOMOUT:
				logMessage(LEVEL_DEBUG, TYPE_NVIM, "Option", OPTION_KEY_ZOOMOUT, "is", opt[1])
				singleton.options.keyDecreaseFontSize = opt[1]
				break
			default:
				logMessage(LEVEL_WARN, TYPE_NVIM, "Invalid option", opt)
			}
		}
		proc.optionStack = proc.optionStack[0:0]
		proc.optionChanged.Set(false)
	}
}

// DEPRECATED
func (proc *NvimProcess) checkDeprecatedOptions() {
	defer measure_execution_time()()
	options := &singleton.options
	var s string
	var f float32
	var i int
	var b bool
	if proc.handle.Var(OPTION_CURSOR_ANIM_DEP, &f) == nil {
		if f != options.cursorAnimTime {
			logMessage(LEVEL_WARN, TYPE_NVIM, "Deprecated option", OPTION_CURSOR_ANIM_DEP, "is", f)
			options.cursorAnimTime = f
		}
	}
	if proc.handle.Var(OPTION_TRANSPARENCY_DEP, &f) == nil {
		if f != options.transparency {
			logMessage(LEVEL_WARN, TYPE_NVIM, "Deprecated option", OPTION_TRANSPARENCY_DEP, "is", f)
			options.transparency = f32clamp(f, 0, 1)
		}
	}
	if proc.handle.Var(OPTION_TARGET_TPS_DEP, &i) == nil {
		if i != options.targetTPS {
			logMessage(LEVEL_WARN, TYPE_NVIM, "Deprecated option", OPTION_TARGET_TPS_DEP, "is", i)
			options.targetTPS = i
		}
	}
	if proc.handle.Var(OPTION_CONTEXT_MENU_DEP, &b) == nil {
		if b != options.contextMenuEnabled {
			logMessage(LEVEL_WARN, TYPE_NVIM, "Deprecated option", OPTION_CONTEXT_MENU_DEP, "is", b)
			options.contextMenuEnabled = b
		}
	}
	if proc.handle.Var(OPTION_KEY_FULLSCRN_DEP, &s) == nil {
		if s != options.keyToggleFullscreen {
			logMessage(LEVEL_WARN, TYPE_NVIM, "Deprecated option", OPTION_KEY_FULLSCRN_DEP, "is", s)
			options.keyToggleFullscreen = s
		}
	}
	if proc.handle.Var(OPTION_KEY_ZOOMIN_DEP, &s) == nil {
		if s != options.keyIncreaseFontSize {
			logMessage(LEVEL_WARN, TYPE_NVIM, "Deprecated option", OPTION_KEY_ZOOMIN_DEP, "is", s)
			options.keyIncreaseFontSize = s
		}
	}
	if proc.handle.Var(OPTION_KEY_ZOOMOUT_DEP, &s) == nil {
		if s != options.keyDecreaseFontSize {
			logMessage(LEVEL_WARN, TYPE_NVIM, "Deprecated option", OPTION_KEY_ZOOMOUT_DEP, "is", s)
			options.keyDecreaseFontSize = s
		}
	}
	// Window startup size
	if proc.handle.Var(OPTION_WINDOW_SIZE_DEP, &s) == nil {
		logMessage(LEVEL_WARN, TYPE_NVIM, "Deprecated option", OPTION_WINDOW_SIZE_DEP, "is", s)
		// Parse the string
		width, height, ok := parseSizeString(s)
		if ok {
			singleton.window.setSize(width, height, true)
		} else {
			logMessage(LEVEL_WARN, TYPE_NVIM, "Could not parse size value:", s)
		}
	}
	// Window startup state
	if proc.handle.Var(OPTION_WINDOW_STATE_DEP, &s) == nil {
		logMessage(LEVEL_WARN, TYPE_NVIM, "Deprecated option", OPTION_WINDOW_STATE_DEP, "is", s)
		singleton.window.setState(s)
	}
}

func (proc *NvimProcess) execCommand(format string, args ...interface{}) bool {
	cmd := fmt.Sprintf(format, args...)
	logMessage(LEVEL_DEBUG, TYPE_NVIM, "Executing command: [", cmd, "]")
	err := proc.handle.Command(cmd)
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NVIM, "Command execution failed: [", cmd, "] err:", err)
		return false
	}
	return true
}

func (proc *NvimProcess) currentMode() string {
	mode, err := proc.handle.Mode()
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NVIM, "Failed to get current mode name:", err)
		return ""
	}
	return mode.Mode
}

func (proc *NvimProcess) echoMsg(format string, args ...interface{}) {
	formatted := fmt.Sprintf(format, args...)
	proc.execCommand("echomsg '%s'", formatted)
}

func (proc *NvimProcess) echoErr(format string, args ...interface{}) {
	formatted := fmt.Sprintf(format, args...)
	proc.handle.WritelnErr(formatted)
}

func (proc *NvimProcess) getRegister(register string) string {
	var content string
	err := proc.handle.Call("getreg", &content, register)
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NVIM, "Api call getreg() failed:", err)
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

// Pastes text at cursor.
func (proc *NvimProcess) paste(str string) {
	err := proc.handle.Call("nvim_paste", nil, str, true, -1)
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NVIM, "Api call nvim_paste() failed:", err)
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
	proc.execCommand("edit %s", file)
}

func (proc *NvimProcess) gotoLine(line int) {
	logMessage(LEVEL_DEBUG, TYPE_NVIM, "Goto Line:", line)
	proc.handle.Call("cursor", nil, line, 0)
}

func (proc *NvimProcess) gotoColumn(col int) {
	logMessage(LEVEL_DEBUG, TYPE_NVIM, "Goto Column:", col)
	proc.handle.Call("cursor", nil, 0, col)
}

func (proc *NvimProcess) feedKeys(keys string) {
	keycode, err := proc.handle.ReplaceTermcodes(keys, true, true, true)
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NVIM, "Failed to replace termcodes:", err)
		return
	}
	err = proc.handle.FeedKeys(keycode, "m", true)
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NVIM, "Failed to feed keys:", err)
	}
}

func (proc *NvimProcess) input(keycode string) {
	written, err := proc.handle.Input(keycode)
	if err != nil {
		logMessage(LEVEL_WARN, TYPE_NVIM, "Failed to send input keys:", err)
	}
	if written != len(keycode) {
		logMessage(LEVEL_WARN, TYPE_NVIM, "Failed to send some keys.")
	}
}

func (proc *NvimProcess) inputMouse(button, action, modifier string, grid, row, column int) {
	err := proc.handle.InputMouse(button, action, modifier, grid, row, column)
	if err != nil {
		logMessage(LEVEL_WARN, TYPE_NVIM, "Failed to send mouse input:", err)
	}
}

func (proc *NvimProcess) requestResize(rows, cols int) {
	assert(rows > 0 && cols > 0, "requested resize with zero parameter")
	err := proc.handle.TryResizeUI(cols, rows)
	if err != nil {
		logMessage(LEVEL_ERROR, TYPE_NVIM, "Failed to send resize request:", err)
		return
	}
}

func (proc *NvimProcess) Close() {
	// NOTE: We are always trying to close neovim even though it closes itself before us.
	err := proc.handle.Close()
	if err != nil {
		logMessage(LEVEL_WARN, TYPE_NVIM, "Failed to close neovim child process:", err)
	}
}
