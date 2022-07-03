// Neoray package for windows syso files.
// Main must include this package.
package assets

import _ "embed"

var (
	//go:embed icons/neovim-16.png
	NeovimIconData16x16 []byte
	//go:embed icons/neovim-32.png
	NeovimIconData32x32 []byte
	//go:embed icons/neovim-48.png
	NeovimIconData48x48 []byte
)
