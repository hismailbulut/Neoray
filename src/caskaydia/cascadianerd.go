package caskaydia

import (
	_ "embed"
)

// This package contains the data of the CascadiaCode fonts.
// The regular one is completely packed with nerd fonts.
// Others are original.

var (
	//go:embed CaskaydiaCove-Regular.ttf
	Regular []byte
	//go:embed CascadiaCode-BoldItalic.ttf
	BoldItalic []byte
	//go:embed CascadiaCode-Italic.ttf
	Italic []byte
	//go:embed CascadiaCode-Bold.ttf
	Bold []byte
)
