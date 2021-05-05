package main

func main() {
	editor := Editor{}
	editor.Initialize()
	defer editor.Shutdown()
	editor.MainLoop()
}
