package main

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hismailbulut/Neoray/pkg/bench"
	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/logger"
	"github.com/neovim/go-client/nvim"
)

const (
	// New options
	OPTION_CURSOR_ANIM    = "CursorAnimTime"
	OPTION_TRANSPARENCY   = "Transparency"
	OPTION_TARGET_TPS     = "TargetTPS"
	OPTION_CONTEXT_MENU   = "ContextMenu"
	OPTION_CONTEXT_BUTTON = "ContextButton"
	OPTION_BOX_DRAWING    = "BoxDrawing"
	OPTION_IMAGE_VIEWER   = "ImageViewer"
	OPTION_WINDOW_STATE   = "WindowState"
	OPTION_WINDOW_SIZE    = "WindowSize"
	// Keybindings
	OPTION_KEY_FULLSCRN = "KeyFullscreen"
	OPTION_KEY_ZOOMIN   = "KeyZoomIn"
	OPTION_KEY_ZOOMOUT  = "KeyZoomOut"
)

//go:embed neoray.vim
var NeorayRuntimeScript string

type NvimProcess struct {
	handle     *nvim.Nvim
	eventChan  chan []interface{}
	optionChan chan []string
	// This is required for when closing neoray. If neoray connected via stdin-out
	// it is responsible for closing nvim, but if neoray connected via tcp, it will
	// not close nvim.
	connectedViaTcp bool
}

func CreateNvimProcess() *NvimProcess {
	proc := &NvimProcess{
		eventChan:  make(chan []interface{}, 256), // Thats enough
		optionChan: make(chan []string, 16),
	}

	if Editor.parsedArgs.address != "" {
		// Try to connect via tcp
		var err error
		proc.handle, err = nvim.Dial(Editor.parsedArgs.address,
			nvim.DialLogf(func(format string, args ...interface{}) {
				logger.LogF(logger.TRACE, format, args...)
			}),
		)
		if err != nil {
			logger.Log(logger.ERROR, "Failed to connect existing neovim instance:", err)
		} else {
			logger.Log(logger.TRACE, "Connected to existing neovim at address:", Editor.parsedArgs.address)
			proc.connectedViaTcp = true
		}
	}

	if !proc.connectedViaTcp {
		// Connect via stdin-stdout
		args := append([]string{"--embed"}, Editor.parsedArgs.others...)
		var err error
		proc.handle, err = nvim.NewChildProcess(
			nvim.ChildProcessArgs(args...),
			nvim.ChildProcessCommand(Editor.parsedArgs.execPath),
		)
		if err != nil {
			logger.Log(logger.FATAL, "Failed to start neovim instance:", err)
		}
		logger.Log(logger.TRACE, "Neovim started with command:", Editor.parsedArgs.execPath, args)
	}

	info, err := proc.handle.APIInfo()
	if err != nil {
		logger.Log(logger.FATAL, "Failed to get api information:", err)
	} else {
		// Check neovim version
		// info[1] is dictionary of infos and it has a key named 'version',
		// and this key contains a map which has major, minor and patch informations

		vInfo := info[1].(map[string]interface{})["version"].(map[string]interface{})
		vMajor := to_int(vInfo["major"])
		vMinor := to_int(vInfo["minor"])
		vPatch := to_int(vInfo["patch"])

		if vMinor < 5 {
			logger.Log(logger.FATAL, "Neoray needs at least 0.5.0 version of neovim. Please update your neovim to a newer version.")
		}

		vStr := fmt.Sprintf("%d.%d.%d", vMajor, vMinor, vPatch)
		logger.Log(logger.TRACE, "Neovim version", vStr)
	}

	// Set a variable that users can define their neoray specific customization.
	proc.handle.SetVar("neoray", 1)

	// Prepare runtime script
	source := NeorayRuntimeScript
	// Replace \r\n to \n (windows)
	source = strings.ReplaceAll(source, "\r\n", "\n")
	// Remove starting lines with # (comments)
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			lines[i] = ""
		}
	}
	source = strings.Join(lines, "\n")
	// Replace channel ids in the template
	source = strings.ReplaceAll(source, "$(CHANID)", strconv.Itoa(proc.handle.ChannelID()))

	// Execute runtime script
	_, err = proc.handle.Exec(source, false)
	if err != nil {
		logger.Log(logger.FATAL, "Failed to execute runtime script:", err)
	}

	// Register NeorayOptionSet
	proc.RegisterHandler(
		"NeorayOptionSet",
		func(args ...string) {
			proc.optionChan <- args
		},
	)

	// Register VimEnter
	// NOTE: We are not using this for now but I added for we may need in future
	proc.RegisterHandler(
		"NeorayVimEnter",
		func() {
			logger.Log(logger.DEBUG, "VimEnter")
		},
	)

	// Register VimLeave
	proc.RegisterHandler(
		"NeorayVimLeave",
		func() {
			logger.Log(logger.DEBUG, "VimLeave")
			Editor.quitChan <- true
		},
	)

	// Register ImageViewer
	proc.RegisterHandler(
		"NeorayViewImage",
		func(imgPath string) (bool, error) {
			if Editor.options.imageViewerEnabled {
				logger.Log(logger.DEBUG, "ViewImage:", imgPath)
				Editor.imageViewer.imageChan <- imgPath
				return true, nil
			} else {
				return false, nil
			}
		},
	)

	return proc
}

func (proc *NvimProcess) RegisterHandler(name string, handler interface{}) {
	err := proc.handle.RegisterHandler(name, handler)
	if err != nil {
		logger.LogF(logger.FATAL, "Failed to register handler for '%s' because error: %v", name, err)
	}
}

func (proc *NvimProcess) StartUI(rows, cols int) {
	proc.RegisterHandler("redraw", func(events ...[]interface{}) {
		for _, event := range events {
			proc.eventChan <- event
		}
	})

	options := map[string]interface{}{
		"rgb":          true,
		"ext_linegrid": true,
	}

	if Editor.parsedArgs.multiGrid {
		options["ext_multigrid"] = true
		logger.Log(logger.DEBUG, "Multigrid enabled.")
	}

	if err := proc.handle.AttachUI(cols, rows, options); err != nil {
		logger.Log(logger.FATAL, "AttachUI failed:", err)
	}

	// Dictionary describing the version
	version := nvim.ClientVersion{
		Major: VERSION_MAJOR,
		Minor: VERSION_MINOR,
		Patch: VERSION_PATCH,
	}
	if bench.IsDebugBuild() {
		version.Prerelease = "dev"
	}
	// Client type
	typ := nvim.UIClientType
	// Builtin methods in the client
	methods := make(map[string]*nvim.ClientMethod, 0)
	// Arbitrary string:string map of informal client properties
	attributes := make(nvim.ClientAttributes, 1)
	attributes["website"] = WEBPAGE
	attributes["license"] = LICENSE
	go func() {
		// NOTE: SetClientInfo api call is blocking, if we don't go this one
		// and some error happens in nvim, nvim will wait until user presses
		// enter but because we blocked here, we will not be able to create
		// window and no user event can be handled Also we can not render the
		// error screen. See issue #33
		// NOTE: Not only this one but most api calls gets blocking
		err := proc.handle.SetClientInfo(NAME, version, typ, methods, attributes)
		if err != nil {
			logger.Log(logger.FATAL, "Failed to set client information:", err)
		}
	}()

	logger.Log(logger.DEBUG, "Attached to neovim as an ui client")
}

// Neoray only has to call this when quiting without closing neovim
func (proc *NvimProcess) disconnect() {
	proc.handle.Unsubscribe("redraw")
	proc.handle.Unsubscribe("NeorayOptionSet")
	proc.handle.Unsubscribe("NeorayVimEnter")
	proc.handle.Unsubscribe("NeorayVimLeave")
	proc.handle.Unsubscribe("NeorayViewImage")
	proc.handle.DetachUI()
}

func (proc *NvimProcess) Update() {
	// We wait for first flush because some of the settings depends on default grid
	// and we only make sure default grid has drawn after the first flush
	if Editor.state >= EditorFirstFlush {
		proc.CheckOptions()
		// If this is the first option check we can show the window after it
		// because all initializations and user settings are done
		if Editor.state < EditorWindowShown {
			Editor.window.Show()
			SetEditorState(EditorWindowShown)
			logger.Log(logger.TRACE, "Window is visible now in", time.Since(StartTime))
		}
	}
}

func (proc *NvimProcess) CheckOptions() {
	for len(proc.optionChan) > 0 {
		option := <-proc.optionChan
		proc.processOption(option)
	}
}

func (proc *NvimProcess) processOption(opt []string) {
	// opt[0] is the name of the option, others are arguments
	switch opt[0] {
	case OPTION_CURSOR_ANIM:
		{
			value, err := strconv.ParseFloat(opt[1], 32)
			if err != nil {
				logger.Log(logger.WARN, OPTION_CURSOR_ANIM, "value isn't valid.")
				break
			}
			logger.Log(logger.DEBUG, "Option", OPTION_CURSOR_ANIM, "is", opt[1])
			Editor.options.cursorAnimTime = float32(value)
		}
	case OPTION_TRANSPARENCY:
		{
			value, err := strconv.ParseFloat(opt[1], 32)
			if err != nil {
				logger.Log(logger.WARN, OPTION_TRANSPARENCY, "value isn't valid.")
				break
			}
			logger.Log(logger.DEBUG, "Option", OPTION_TRANSPARENCY, "is", opt[1])
			Editor.options.transparency = common.Clamp(float32(value), 0, 1)
			MarkForceDraw()
		}
	case OPTION_TARGET_TPS:
		{
			value, err := strconv.Atoi(opt[1])
			if err != nil {
				logger.Log(logger.WARN, OPTION_TARGET_TPS, "value isn't valid.")
				break
			}
			logger.Log(logger.DEBUG, "Option", OPTION_TARGET_TPS, "is", value)
			Editor.options.targetTPS = value
			ResetTicker()
		}
	case OPTION_CONTEXT_MENU:
		{
			value, err := strconv.ParseBool(opt[1])
			if err != nil {
				logger.Log(logger.WARN, OPTION_CONTEXT_MENU, "value isn't valid.")
				break
			}
			logger.Log(logger.DEBUG, "Option", OPTION_CONTEXT_MENU, "is", value)
			Editor.options.contextMenuEnabled = value
		}
	case OPTION_CONTEXT_BUTTON:
		{
			if len(opt) >= 3 {
				cmd := strings.Join(opt[2:], " ")
				logger.Log(logger.DEBUG, "Option", OPTION_CONTEXT_BUTTON, "name is", opt[1], "and command is", cmd)
				Editor.contextMenu.AddButton(ContextButton{
					name: opt[1],
					fn:   func() { proc.execCommand(cmd) },
				})
			} else {
				logger.Log(logger.WARN, "Not enough argument for option", OPTION_CONTEXT_BUTTON)
			}
		}
	case OPTION_BOX_DRAWING:
		{
			value, err := strconv.ParseBool(opt[1])
			if err != nil {
				logger.Log(logger.WARN, OPTION_BOX_DRAWING, "value isn't valid.")
				break
			}
			logger.Log(logger.DEBUG, "Option", OPTION_BOX_DRAWING, "is", value)
			Editor.options.boxDrawingEnabled = value
			// Currently we didn't separate this two options but may be in the future
			Editor.gridManager.SetBoxDrawing(Editor.options.boxDrawingEnabled, Editor.options.boxDrawingEnabled)
		}
	case OPTION_IMAGE_VIEWER:
		{
			value, err := strconv.ParseBool(opt[1])
			if err != nil {
				logger.Log(logger.WARN, OPTION_IMAGE_VIEWER, "value isn't valid.")
				break
			}
			logger.Log(logger.DEBUG, "Option", OPTION_IMAGE_VIEWER, "is", value)
			Editor.options.imageViewerEnabled = value
		}
	case OPTION_WINDOW_STATE:
		{
			logger.Log(logger.DEBUG, "Option", OPTION_WINDOW_STATE, "is", opt[1])
			switch opt[1] {
			case "minimized":
				Editor.window.Minimize()
			case "maximized":
				Editor.window.Maximize()
			case "fullscreen":
				if !Editor.window.IsFullscreen() {
					Editor.window.ToggleFullscreen()
				}
			case "centered":
				Editor.window.Center()
			}
		}
	case OPTION_WINDOW_SIZE:
		{
			cols, rows, ok := func(size string) (int, int, bool) {
				// Size must be in form of '10x10'
				values := strings.Split(size, "x")
				if len(values) != 2 {
					return 0, 0, false
				}
				width, err := strconv.Atoi(values[0])
				if err != nil {
					return 0, 0, false
				}
				height, err := strconv.Atoi(values[1])
				if err != nil {
					return 0, 0, false
				}
				return width, height, true
			}(opt[1])
			if !ok {
				logger.Log(logger.WARN, OPTION_WINDOW_SIZE, "value isn't valid.")
				break
			}
			logger.Log(logger.DEBUG, "Option", OPTION_WINDOW_SIZE, "is", cols, rows)
			ResizeWindowInCellFormat(rows, cols)
		}
	case OPTION_KEY_FULLSCRN:
		{
			logger.Log(logger.DEBUG, "Option", OPTION_KEY_FULLSCRN, "is", opt[1])
			Editor.options.keyToggleFullscreen = opt[1]
		}
	case OPTION_KEY_ZOOMIN:
		{
			logger.Log(logger.DEBUG, "Option", OPTION_KEY_ZOOMIN, "is", opt[1])
			Editor.options.keyIncreaseFontSize = opt[1]
		}
	case OPTION_KEY_ZOOMOUT:
		{
			logger.Log(logger.DEBUG, "Option", OPTION_KEY_ZOOMOUT, "is", opt[1])
			Editor.options.keyDecreaseFontSize = opt[1]
		}
	default:
		logger.Log(logger.WARN, "Invalid option", opt)
	}
}

func (proc *NvimProcess) execCommand(format string, args ...interface{}) bool {
	cmd := fmt.Sprintf(format, args...)
	logger.Log(logger.DEBUG, "Executing command: [", cmd, "]")
	err := proc.handle.Command(cmd)
	if err != nil {
		logger.Log(logger.ERROR, "Command execution failed: [", cmd, "] err:", err)
		return false
	}
	return true
}

func (proc *NvimProcess) currentMode() string {
	mode, err := proc.handle.Mode()
	if err != nil {
		logger.Log(logger.ERROR, "Failed to get current mode name:", err)
		return ""
	}
	return mode.Mode
}

func (proc *NvimProcess) echoMsg(format string, args ...interface{}) {
	formatted := fmt.Sprintf(format, args...)
	go proc.execCommand("echomsg '%s'", formatted)
}

func (proc *NvimProcess) echoErr(format string, args ...interface{}) {
	formatted := fmt.Sprintf(format, args...)
	proc.handle.WritelnErr(formatted)
	// Also log this as an error
	logger.LogF(logger.ERROR, format, args...)
}

func (proc *NvimProcess) getRegister(register string) string {
	var content string
	err := proc.handle.Call("getreg", &content, register)
	if err != nil {
		logger.Log(logger.ERROR, "Api call getreg() failed:", err)
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
	go func() {
		err := proc.handle.Call("nvim_paste", nil, str, true, -1)
		if err != nil {
			logger.Log(logger.ERROR, "Api call nvim_paste() failed:", err)
		}
	}()
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
	logger.Log(logger.DEBUG, "Open file", file)
	go proc.execCommand("edit %s", file)
}

func (proc *NvimProcess) gotoLine(line int) {
	logger.Log(logger.DEBUG, "Goto line", line)
	go proc.handle.Call("cursor", nil, line, 0)
}

func (proc *NvimProcess) gotoColumn(col int) {
	logger.Log(logger.DEBUG, "Goto column", col)
	go proc.handle.Call("cursor", nil, 0, col)
}

func (proc *NvimProcess) feedKeys(keys string) {
	keycode, err := proc.handle.ReplaceTermcodes(keys, true, true, true)
	if err != nil {
		logger.Log(logger.ERROR, "Failed to replace termcodes:", err)
		return
	}
	err = proc.handle.FeedKeys(keycode, "m", true)
	if err != nil {
		logger.Log(logger.ERROR, "Failed to feed keys:", err)
	}
}

func (proc *NvimProcess) input(keycode string) {
	written, err := proc.handle.Input(keycode)
	if err != nil {
		logger.Log(logger.WARN, "Failed to send input keys:", err)
	}
	if written != len(keycode) {
		logger.Log(logger.WARN, "Failed to send some keys.")
	}
}

func (proc *NvimProcess) inputMouse(button, action, modifier string, grid, row, column int) {
	err := proc.handle.InputMouse(button, action, modifier, grid, row, column)
	if err != nil {
		logger.Log(logger.WARN, "Failed to send mouse input:", err)
	}
}

func (proc *NvimProcess) tryResizeUI(rows, cols int) {
	if rows <= 0 || cols <= 0 {
		return
	}
	go func() {
		err := proc.handle.TryResizeUI(cols, rows)
		if err != nil {
			logger.Log(logger.ERROR, "Failed to send resize request:", err)
			return
		}
	}()
}

func (proc *NvimProcess) tryResizeGrid(id, rows, cols int) {
	if rows <= 0 || cols <= 0 {
		return
	}
	go func() {
		err := proc.handle.TryResizeUIGrid(id, cols, rows)
		if err != nil {
			logger.Log(logger.ERROR, "Failed to send resize request:", err)
			return
		}
	}()
}

func (proc *NvimProcess) Close() {
	// Sometimes Close function blocks forever
	// I realized that when using a popular neovim configuration
	// And it only happens when :wq in a lua file
	go func() {
		err := proc.handle.Close()
		if err != nil {
			logger.Log(logger.WARN, "Failed to close neovim client:", err)
		} else {
			logger.Log(logger.DEBUG, "Neovim client closed")
		}
	}()
}
