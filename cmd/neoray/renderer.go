package main

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	FONT_ATLAS_DEFAULT_SIZE int = 256
)

type FontAtlas struct {
	texture    Texture
	pos        ivec2
	characters map[string]ivec2
}

type Renderer struct {
	font          Font
	font_atlas    FontAtlas
	cell_width    int
	cell_height   int
	window_width  int
	window_height int
	// vertex_data   []Vertex
}

func CreateRenderer(window *Window, font Font) Renderer {
	cell_width, cell_height := font.CalculateCellSize()
	renderer := Renderer{
		font: font,
		font_atlas: FontAtlas{
			characters: make(map[string]ivec2),
		},
		cell_width:  cell_width,
		cell_height: cell_height,
	}

	RGL_Init(window)
	RGL_CreateViewport(window.width, window.height)

	renderer.font_atlas.texture = RGL_CreateTexture(FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE)
	RGL_SetAtlasTexture(&renderer.font_atlas.texture)

	renderer.Resize(window.width, window.height)

	return renderer
}

func (renderer *Renderer) Resize(w, h int) {
	// 6 vertices for every cell
	// renderer.vertex_data = make([]Vertex, w*h*6)
	renderer.window_width = w
	renderer.window_height = h
	RGL_CreateViewport(w, h)
}

func (renderer *Renderer) GetEmptyAtlasPosition() ivec2 {
	atlas := &renderer.font_atlas
	// calculate position
	pos := atlas.pos
	atlas.pos.X += renderer.cell_width
	if atlas.pos.X+renderer.cell_width > int(FONT_ATLAS_DEFAULT_SIZE) {
		// New row
		atlas.pos.X = 0
		atlas.pos.Y += renderer.cell_height
	}
	if atlas.pos.Y+renderer.cell_height > int(FONT_ATLAS_DEFAULT_SIZE) {
		// Fully filled
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Font atlas is full.")
		atlas.pos = ivec2{}
	}
	return pos
}

func (renderer *Renderer) GetCharacterAtlasPosition(char string, italic, bold bool) (sdl.Rect, error) {
	var position sdl.Rect
	// generate specific id for this character
	id := fmt.Sprintf("%s%t%t", char, italic, bold)
	if pos, ok := renderer.font_atlas.characters[id]; ok == true {
		// use stored texture
		position = sdl.Rect{
			X: int32(pos.X),
			Y: int32(pos.Y),
			W: int32(renderer.cell_width),
			H: int32(renderer.cell_height),
		}
	} else {
		// Create this text
		font_handle := renderer.font.GetDrawableFont(italic, bold)
		text_surface, err := font_handle.RenderUTF8Blended(char, COLOR_WHITE)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, err)
			return sdl.Rect{}, err
		}
		defer text_surface.Free()
		// Get empty atlas position
		text_pos := renderer.GetEmptyAtlasPosition()
		position = sdl.Rect{
			X: int32(text_pos.X),
			Y: int32(text_pos.Y),
			W: int32(renderer.cell_width),
			H: int32(renderer.cell_height),
		}
		if text_surface.W > int32(renderer.cell_width) || text_surface.H > int32(renderer.cell_height) {
			// TODO: scale surface or scale texture
			position.W = text_surface.W
			position.H = text_surface.H
		}
		// Draw text to empty position of atlas texture
		renderer.font_atlas.texture.UpdatePartFromSurface(text_surface, &position)
		// Save this character for further use
		renderer.font_atlas.characters[id] = ivec2{int(position.X), int(position.Y)}
	}
	return position, nil
}

func (renderer *Renderer) DrawRectangle(rect sdl.Rect, color sdl.Color) {
	RGL_FillRect(rect, color)
}

func (renderer *Renderer) DrawCell(x, y int32, fg, bg sdl.Color, char string, italic, bold bool) {
	cell_rect := sdl.Rect{
		X: y * int32(renderer.cell_width),
		Y: x * int32(renderer.cell_height),
		W: int32(renderer.cell_width),
		H: int32(renderer.cell_height),
	}
	// draw Background
	renderer.DrawRectangle(cell_rect, bg)
	if len(char) == 0 || char == " " {
		return
	}
	// get character position in atlas texture
	atlas_char_pos, err := renderer.GetCharacterAtlasPosition(char, italic, bold)
	if err != nil {
		return
	}
	// draw
	RGL_DrawSubTextureColor(renderer.font_atlas.texture, &atlas_char_pos, &cell_rect, fg)
}

func (renderer *Renderer) DrawCellWithAttrib(grid *Grid, x, y int32) {
	fg := grid.default_fg
	bg := grid.default_bg
	sp := grid.default_sp
	italic := false
	bold := false
	cell := &grid.cells[x][y]
	// attrib id 0 is default palette
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
		// reverse foreground and background
		if attrib.reverse {
			fg, bg = bg, fg
		}
		// underline and undercurl uses special color for foreground
		if attrib.underline || attrib.undercurl {
			fg = sp
		}
	}
	// TODO: use user defined transparency
	bg.A = BG_TRANSPARENCY
	// Draw cell
	renderer.DrawCell(x, y, fg, bg, cell.char, italic, bold)
}

func (renderer *Renderer) Draw(editor *Editor) {
	RGL_ClearScreen(editor.grid.default_bg)

	for x, row := range editor.grid.cells {
		for y := range row {
			renderer.DrawCellWithAttrib(&editor.grid, int32(x), int32(y))
		}
	}

	// Draw cursor
	editor.cursor.Draw(&editor.grid, &editor.renderer, &editor.mode)

	// DEBUG: draw font atlas to top right
	atlas_pos := sdl.Rect{
		X: int32((editor.grid.width * renderer.cell_width) - int(FONT_ATLAS_DEFAULT_SIZE)),
		Y: 0,
		W: int32(FONT_ATLAS_DEFAULT_SIZE),
		H: int32(FONT_ATLAS_DEFAULT_SIZE),
	}
	RGL_DrawTexture(renderer.font_atlas.texture, &atlas_pos)

	// Render changes
	RGL_Render(renderer.font_atlas.texture)

	// Swap window surface
	editor.window.handle.GLSwap()
}

func (renderer *Renderer) Close() {
	renderer.font_atlas.texture.Delete()
	renderer.font.Unload()
	RGL_Close()
}
