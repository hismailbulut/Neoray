package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/neovim/go-client/nvim"
)

type NvimProcess struct {
	handle       *nvim.Nvim
	update_mutex *sync.Mutex
	update_stack [][][]interface{}
}

func CreateNvimProcess() NvimProcess {
	proc := NvimProcess{
		update_mutex: &sync.Mutex{},
		update_stack: make([][][]interface{}, 0),
	}
	args := []string{
		"--embed",
		// "-u",
		// "NORC",
		// "--noplugin",
	}
	args = append(args, os.Args[1:]...)

	nv, err := nvim.NewChildProcess(
		nvim.ChildProcessServe(true),
		nvim.ChildProcessArgs(args...))
	if err != nil {
		log.Fatalln(err)
	}

	proc.handle = nv
	proc.requestApiInfo()
	proc.introduce()

	fmt.Println("Neovim child process created.")

	return proc
}

func (proc *NvimProcess) requestApiInfo() {
	_, err := proc.handle.APIInfo()
	if err != nil {
		fmt.Println("ERROR: Failed to get api information.", err)
		return
	}
	// for _, info := range apiInfo[1:] {
	//     mapIter := reflect.ValueOf(info).MapRange()
	//     for mapIter.Next() {
	//         fmt.Println("Key:", mapIter.Key(), "Val:", mapIter.Value())
	//     }
	// }
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
		fmt.Println(err)
	}
}

func (proc *NvimProcess) ExecuteVimScript(script string) {
	if err := proc.handle.Command(script); err != nil {
		fmt.Println("VimScript Error:", err)
	}
}

func (proc *NvimProcess) SendKeyCode(keycode string) {
	term_code, err := proc.handle.ReplaceTermcodes(keycode, true, true, true)
	if err != nil {
		fmt.Println("ReplaceTermcodes:", err)
		return
	}
	err = proc.handle.FeedKeys(term_code, "m", false)
	if err != nil {
		fmt.Println("FeedKeys:", err)
	}
}

func (proc *NvimProcess) StartUI(editor *Editor) {
	options := make(map[string]interface{})
	options["rgb"] = true
	options["ext_linegrid"] = true

	col_count := editor.window.width / editor.renderer.cell_width
	row_count := editor.window.height / editor.renderer.cell_height
	proc.handle.AttachUI(col_count, row_count, options)

	proc.handle.RegisterHandler("redraw",
		func(updates ...[]interface{}) {
			proc.update_mutex.Lock()
			proc.update_stack = append(proc.update_stack, updates)
			proc.update_mutex.Unlock()
		})

	fmt.Println("UI Connected.")
}

func (proc *NvimProcess) ResizeUI(editor *Editor) {
	col_count := editor.window.width / editor.renderer.cell_width
	row_count := editor.window.height / editor.renderer.cell_height
	proc.handle.TryResizeUI(col_count, row_count)
}

func (proc *NvimProcess) Close() {
	proc.handle.Close()
}
