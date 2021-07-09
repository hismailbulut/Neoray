package gomononerd

import (
	_ "embed"
)

// This package contains the data of the Go Mono fonts.
// The regular one is completely packed with nerd fonts.
// Others are original.

var (
	//go:embed GoMono-Regular.ttf
	Regular []byte
	//go:embed GoMono-BoldItalic.ttf
	BoldItalic []byte
	//go:embed GoMono-Italic.ttf
	Italic []byte
	//go:embed GoMono-Bold.ttf
	Bold []byte
)
