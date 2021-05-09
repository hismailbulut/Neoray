package main

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
	// "github.com/veandco/go-sdl2/ttf"
)

type Renderer struct {
	handle              *sdl.Renderer
	known_font_textures map[string]*sdl.Texture
	font                Font
	cell_width          int
	cell_height         int
}

func CreateRenderer(window *Window, font Font) Renderer {
	cell_width, cell_height := font.CalculateCellSize()
	renderer := Renderer{
		known_font_textures: make(map[string]*sdl.Texture),
		font:                font,
		cell_width:          cell_width,
		cell_height:         cell_height,
	}
	sdl_renderer, err := sdl.CreateRenderer(window.handle, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize SDL renderer:", err)
	}
	renderer.handle = sdl_renderer
	return renderer
}

func (renderer *Renderer) DrawRectangle(rect sdl.Rect, color sdl.Color) {
	renderer.handle.SetDrawColor(color.R, color.G, color.B, color.A)
	renderer.handle.FillRect(&rect)
}

func (renderer *Renderer) DrawCharacter(x, y int32, fg, bg sdl.Color, char string, italic, bold bool) {
	cell_rect := sdl.Rect{
		X: y * int32(renderer.cell_width),
		Y: x * int32(renderer.cell_height),
		W: int32(renderer.cell_width),
		H: int32(renderer.cell_height),
	}
	if len(char) == 0 {
		renderer.DrawRectangle(cell_rect, bg)
		return
	}
	// Create texture from text surface
	id := fmt.Sprintf("%s%v%v%t%t", char, fg, bg, italic, bold)
	var text_texture *sdl.Texture
	if val, ok := renderer.known_font_textures[id]; ok == true {
		text_texture = val
	} else {
		// Create surface and draw font to it
		font_handle := renderer.font.GetDrawableFont(italic, bold)
		text_surface, err := font_handle.RenderUTF8Shaded(char, fg, bg)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, err)
			return
		}
		defer text_surface.Free()
		text_texture, err = renderer.handle.CreateTextureFromSurface(text_surface)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, err)
			return
		}
		text_texture.SetBlendMode(sdl.BLENDMODE_NONE)
		renderer.known_font_textures[id] = text_texture
	}
	// Copy texture to main framebuffer
	err := renderer.handle.Copy(text_texture, nil, &cell_rect)
	if err != nil {
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, err)
	}
}

func (renderer *Renderer) DrawCell(grid *Grid, x, y int32) {
	fg := grid.default_fg
	bg := grid.default_bg
	sp := grid.default_sp

	italic := false
	bold := false

	cell := &grid.cells[x][y]

	if cell.attrib_id > 0 {
		// set attribute colors
		attrib := grid.attributes[cell.attrib_id]
		if !is_color_black(attrib.foreground) {
			fg = attrib.foreground
		}
		if !is_color_black(attrib.background) {
			bg = attrib.background
		}
		if !is_color_black(attrib.special) {
			sp = attrib.special
		}
		// font
		italic = attrib.italic
		bold = attrib.bold
		// reverse color if reverse attribute set
		if attrib.reverse {
			fg, bg = bg, fg
		}
		if attrib.underline || attrib.undercurl {
			fg = sp
		}
	}

	// NOTE: temporary, use blend level
	bg.A = BG_TRANSPARENCY
	// character
	renderer.DrawCharacter(x, y, fg, bg, cell.char, italic, bold)

	if int(y) == len(grid.cells[x])-1 {
		renderer.DrawCharacter(x, y+1, fg, bg, " ", false, false)
	}
}

func (renderer *Renderer) Draw(grid *Grid, mode *Mode, cursor *Cursor) {
	// defer measure_execution_time("Renderer.Draw")()

	for x := 0; x < len(grid.cells); x++ {
		// only draw if this row changed
		if grid.changed_rows[x] == true {
			for y := 0; y < len(grid.cells[x]); y++ {
				renderer.DrawCell(grid, int32(x), int32(y))
			}
			grid.changed_rows[x] = false
		}
	}

	cursor.Draw(grid, renderer, mode)
	grid.changed_rows[cursor.X] = true

	renderer.handle.Present()
}

func (renderer *Renderer) Close() {
	renderer.font.Unload()
	renderer.handle.Destroy()

	for _, val := range renderer.known_font_textures {
		val.Destroy()
	}
}
