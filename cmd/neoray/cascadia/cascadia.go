package cascadia

import (
	_ "embed"
)

var (
	//go:embed CascadiaMono-Regular.ttf
	Regular []byte
	//go:embed CascadiaMono-BoldItalic.otf
	BoldItalic []byte
	//go:embed CascadiaMono-Italic.otf
	Italic []byte
	//go:embed CascadiaMono-Bold.otf
	Bold []byte
)
