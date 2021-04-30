package main

import (
	"fmt"
	"log"
	"os"
	// "sync"

	"github.com/neovim/go-client/nvim"
)

type NvimProcess struct {
	handle *nvim.Nvim
	// update_mutex sync.Mutex
	update_stack [][][]interface{}
}

func CreateProcess() *NvimProcess {
	proc := new(NvimProcess)
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

	return proc
}

func (proc *NvimProcess) requestApiInfo() {
	_, err := proc.handle.APIInfo()
	if err != nil {
		fmt.Println("ERROR: Failed to get api info.")
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
	name := "Neoray"
	// Dictionary describing the version
	version := &nvim.ClientVersion{
		Major:      0,
		Minor:      0,
		Patch:      1,
		Prerelease: "dev",
		Commit:     "main",
	}
	// Client type
	typ := "ui"
	// Builtin methods in the client
	methods := make(map[string]*nvim.ClientMethod, 0)
	// Arbitrary string:string map of informal client properties
	attributes := make(nvim.ClientAttributes, 1)
	attributes["website"] = "github.com/hismailbulut/Neoray"
	attributes["license"] = "GPLv3"

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

func (proc *NvimProcess) StartUI(w *Window) {

	proc.update_stack = make([][][]interface{}, 0)
	options := make(map[string]interface{})

	options["rgb"] = true
	options["ext_linegrid"] = true
	col_count := int(float32(w.width) / w.canvas.cell_width)
	row_count := int(float32(w.height) / w.canvas.cell_height)
	proc.handle.AttachUI(col_count, row_count, options)

	proc.handle.RegisterHandler("redraw",
		func(updates ...[]interface{}) {
			// proc.update_mutex.Lock()
			proc.update_stack = append(proc.update_stack, updates)
			// proc.update_mutex.Unlock()
		})

	proc.handle.RegisterHandler("UIEnter",
		func(updates ...[]interface{}) {
			fmt.Println("UI connected")
		})
}

func (proc *NvimProcess) ResizeUI(w *Window) {
	col_count := int(float32(w.width) / w.canvas.cell_width)
	row_count := int(float32(w.height) / w.canvas.cell_height)
	proc.handle.TryResizeUI(col_count, row_count)
}

func (proc *NvimProcess) Close() {
	proc.handle.Close()
}
