package hacknerd

import (
	_ "embed"
)

// This package contains the data of the Hack fonts. Version 3.003
// The regular one is completely packed with nerd fonts.
// Others are original.

var (
	//go:embed Hack-Regular.ttf
	Regular []byte
	//go:embed Hack-BoldItalic.ttf
	BoldItalic []byte
	//go:embed Hack-Italic.ttf
	Italic []byte
	//go:embed Hack-Bold.ttf
	Bold []byte
)
