
// Neoray package for windows syso files and other assets
// Main must include this package

package assets

import _ "embed"

var (
    // Icons

	//go:embed icons/neovim-16.png
	NeovimIconData16x16 []byte
	//go:embed icons/neovim-32.png
	NeovimIconData32x32 []byte
	//go:embed icons/neovim-48.png
	NeovimIconData48x48 []byte

    // Fonts
    // The regular one is completely packed with nerd fonts.
    // Others are original.

	//go:embed fonts/CaskaydiaCove-Regular.ttf
	Regular []byte
	//go:embed fonts/CascadiaCode-BoldItalic.ttf
	BoldItalic []byte
	//go:embed fonts/CascadiaCode-Italic.ttf
	Italic []byte
	//go:embed fonts/CascadiaCode-Bold.ttf
	Bold []byte
)
