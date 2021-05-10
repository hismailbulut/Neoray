package main

import (
	"fmt"
	"math"

	"github.com/veandco/go-sdl2/sdl"
)

type Renderer struct {
	handle            *sdl.Renderer
	font_text_storage map[string]*sdl.Texture
	font              Font
	cell_width        int
	cell_height       int
}

func CreateRenderer(window *Window, font Font) Renderer {
	cell_width, cell_height := font.CalculateCellSize()
	renderer := Renderer{
		font_text_storage: make(map[string]*sdl.Texture),
		font:              font,
		cell_width:        cell_width,
		cell_height:       cell_height,
	}
	sdl_renderer, err := window.handle.GetRenderer()
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize SDL renderer:", err)
	}
	renderer.handle = sdl_renderer
	return renderer
}

func (renderer *Renderer) DrawRectangle(rect sdl.Rect, color sdl.Color, batch bool) {
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
	if len(char) == 0 || char == " " {
		renderer.DrawRectangle(cell_rect, bg, true)
		return
	}
	var text_texture *sdl.Texture
	id := fmt.Sprintf("(%s-%t-%t)", char, italic, bold)
	if val, ok := renderer.font_text_storage[id]; ok == true {
		// use stored texture
		text_texture = val
	} else {
		// render text to surface
		font_handle := renderer.font.GetDrawableFont(italic, bold)
		text_surface, err := font_handle.RenderUTF8Blended(char, COLOR_WHITE)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, err)
			return
		}
		defer text_surface.Free()

		// clip this surface if its bigger than our rectangle
		if text_surface.W > cell_rect.W || text_surface.H > cell_rect.H {
			// 		10			8				20
			crop_rect := sdl.Rect{
				X: cell_rect.X,
				Y: cell_rect.Y,
				W: cell_rect.W,
				H: int32(math.Ceil(float64(text_surface.H) / (float64(text_surface.W) / float64(cell_rect.W)))),
			}
			log_debug_msg("Char:", char, "W:", text_surface.W, "H:", text_surface.H,
				"CropR:", crop_rect, "CellR:", cell_rect)
		}

		// create texture from this surface and store it for future use
		text_texture, err = renderer.handle.CreateTextureFromSurface(text_surface)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, err)
			return
		}
		renderer.font_text_storage[id] = text_texture
	}
	// Draw Background
	renderer.DrawRectangle(cell_rect, bg, true)
	// Set text color
	text_texture.SetColorMod(fg.R, fg.G, fg.B)
	// Copy texture to main framebuffer
	renderer.handle.Copy(text_texture, nil, &cell_rect)
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

	// TODO: use user defined transparency
	bg.A = BG_TRANSPARENCY

	// character
	renderer.DrawCharacter(x, y, fg, bg, cell.char, italic, bold)
	if int(y) == len(grid.cells[x])-1 {
		renderer.DrawCharacter(x, y+1, fg, bg, "", false, false)
	}
}

func (renderer *Renderer) Draw(grid *Grid, mode *Mode, cursor *Cursor) {
	// defer measure_execution_time("Renderer.Draw")()

	redrawed_rows := make([]int, 0)

	for x := 0; x < len(grid.cells); x++ {
		// only draw if this row changed
		if grid.changed_rows[x] == true {
			for y := 0; y < len(grid.cells[x]); y++ {
				renderer.DrawCell(grid, int32(x), int32(y))
			}
			grid.changed_rows[x] = false
			redrawed_rows = append(redrawed_rows, x)
		}
	}

	log_debug_msg("Redrawed rows:", redrawed_rows)

	cursor.Draw(grid, renderer, mode)
	grid.changed_rows[cursor.X] = true

	renderer.handle.Present()
}

func (renderer *Renderer) Close() {
	renderer.font.Unload()
	renderer.handle.Destroy()

	for _, val := range renderer.font_text_storage {
		val.Destroy()
	}
}
