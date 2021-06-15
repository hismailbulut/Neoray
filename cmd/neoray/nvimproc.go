package main

import (
	"fmt"
	"sync"

	"github.com/neovim/go-client/nvim"
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

	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NVIM, "Neovim child process created.")

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
	name := NEORAY_NAME
	// Dictionary describing the version
	version := &nvim.ClientVersion{
		Major:      NEORAY_VERSION_MAJOR,
		Minor:      NEORAY_VERSION_MINOR,
		Patch:      NEORAY_VERSION_PATCH,
		Prerelease: "dev",
		Commit:     "main",
	}
	// Client type
	typ := "ui"
	// Builtin methods in the client
	methods := make(map[string]*nvim.ClientMethod, 0)
	// Arbitrary string:string map of informal client properties
	attributes := make(nvim.ClientAttributes, 1)
	attributes["website"] = NEORAY_WEBPAGE
	attributes["license"] = NEORAY_LICENSE
	err := proc.handle.SetClientInfo(name, version, typ, methods, attributes)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NVIM, "Failed to set client information:", err)
	}

	// Set a variable that users can define their neoray specific customization.
	proc.handle.SetVar("neoray", 1)

	// id := proc.handle.ChannelID()
	// proc.ExecuteVimScript("au UIEnter * call rpcnotify(%d, \"NeorayUIEnter\")\n", id)
	// proc.handle.RegisterHandler("NeorayUIEnter", func() {
	// proc.update_mutex.Lock()
	// defer proc.update_mutex.Unlock()
	// name := make([][]interface{}, 1, 1)
	// name[0] = make([]interface{}, 1, 1)
	// name[0][0] = "UIEnter"
	// proc.update_stack = append(proc.update_stack, name)
	// })
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
		log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NVIM, "Neovim child process closed.")
		EditorSingleton.quitRequestedChan <- true
	}()

	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NVIM,
		"UI Connected. Rows:", EditorSingleton.rowCount, "Cols:", EditorSingleton.columnCount)
}

func (proc *NvimProcess) ExecuteVimScript(script string, args ...interface{}) {
	s := fmt.Sprintf(script, args...)
	log_debug("Excuting:", s)
	if err := proc.handle.Command(s); err != nil {
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

func (proc *NvimProcess) WriteAfterCursor(str string) {
	switch proc.Mode() {
	case "i":
		proc.SendKeyCode(str)
		break
	case "n":
		proc.FeedKeys("i")
		proc.SendKeyCode(str)
		proc.FeedKeys("<ESC>")
		break
	case "v":
		proc.FeedKeys("<ESC>i")
		proc.SendKeyCode(str)
		proc.FeedKeys("<ESC>gv")
		break
	}
}

func (proc *NvimProcess) SelectAll() {
	switch proc.Mode() {
	case "i":
		proc.FeedKeys("<ESC>ggVG")
		break
	case "n":
		proc.FeedKeys("ggVG")
		break
	case "v":
		proc.FeedKeys("<ESC>ggVG")
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

func (proc *NvimProcess) SendKeyCode(keycode string) {
	written, err := proc.handle.Input(keycode)
	if err != nil {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send input keys:", err)
	}
	if written != len(keycode) {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send some keys.")
	}
}

func (proc *NvimProcess) SendButton(button, action, modifier string, grid, row, column int) {
	err := proc.handle.InputMouse(button, action, modifier, grid, row, column)
	if err != nil {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NVIM, "Failed to send mouse input:", err)
	}
}

// Call CalculateCellSize before this function.
func (proc *NvimProcess) RequestResize() {
	EditorSingleton.CalculateCellCount()
	proc.handle.TryResizeUI(EditorSingleton.columnCount, EditorSingleton.rowCount)
	EditorSingleton.waitingResize = true
}

func (proc *NvimProcess) Close() {
	proc.handle.Close()
}
