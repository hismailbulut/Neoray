package main

import (
	_ "embed"
	neoray "github.com/hismailbulut/neoray/src"
)

var (
	//go:embed fonts/CascadiaMono-Regular.ttf
	regular []byte
	//go:embed fonts/CascadiaMono-BoldItalic.otf
	bold_italic []byte
	//go:embed fonts/CascadiaMono-Italic.otf
	italic []byte
	//go:embed fonts/CascadiaMono-Bold.otf
	bold []byte
)

func main() {
	neoray.SetDefaultFontData(regular, bold_italic, italic, bold)
	neoray.Main()
}
