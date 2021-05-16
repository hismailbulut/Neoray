package main

import (
	"fmt"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	FONT_ATLAS_DEFAULT_SIZE int = 512
)

type FontAtlas struct {
	texture    Texture
	pos        ivec2
	characters map[string]ivec2
}

type Renderer struct {
	font        Font
	font_atlas  FontAtlas
	cell_width  int
	cell_height int
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

	RAPI_Init(window)
	RAPI_CreateViewport(window.width, window.height)

	renderer.font_atlas.texture = RAPI_CreateTexture(FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE)

	return renderer
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
		// get new position and current atlas texture
		text_pos := renderer.GetEmptyAtlasPosition()
		// calculate destination rectangle
		position = sdl.Rect{
			X: int32(text_pos.X),
			Y: int32(text_pos.Y),
			W: int32(renderer.cell_width),
			H: int32(renderer.cell_height),
		}
		// make height as same percent with width
		if text_surface.W > int32(renderer.cell_width) {
			// position.H = int32(math.Ceil(float64(text_surface.H) / (float64(text_surface.W) / float64(position.W))))
		}
		// Draw text to empty position of atlas texture
		renderer.font_atlas.texture.UpdatePartFromSurface(text_surface, &position)
		// Save this character for further use
		renderer.font_atlas.characters[id] = ivec2{int(position.X), int(position.Y)}
	}
	return position, nil
}

func (renderer *Renderer) DrawRectangle(rect sdl.Rect, color sdl.Color) {
	RAPI_FillRect(rect, color)
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
	// get character atlas texture and position
	atlas_char_pos, err := renderer.GetCharacterAtlasPosition(char, italic, bold)
	if err != nil {
		return
	}
	// draw
	RAPI_DrawSubTextureColor(renderer.font_atlas.texture, &atlas_char_pos, &cell_rect, fg)
}

func (renderer *Renderer) PrepareCellToDraw(grid *Grid, x, y int32) {
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
	RAPI_ClearScreen(editor.grid.default_bg)

	for x, row := range editor.grid.cells {
		for y := range row {
			renderer.PrepareCellToDraw(&editor.grid, int32(x), int32(y))
		}
	}

	// Prepare cursor for drawings
	editor.cursor.Draw(&editor.grid, &editor.renderer, &editor.mode)

	// DEBUG: prepare last font atlas for drawing
	atlas_pos := sdl.Rect{
		X: int32((editor.grid.width * renderer.cell_width) - int(FONT_ATLAS_DEFAULT_SIZE)),
		Y: 0,
		W: int32(FONT_ATLAS_DEFAULT_SIZE),
		H: int32(FONT_ATLAS_DEFAULT_SIZE),
	}
	RAPI_DrawTexture(renderer.font_atlas.texture, &atlas_pos)

	// Draw
	RAPI_Render(renderer.font_atlas.texture)

	// Swap window surface
	editor.window.handle.GLSwap()
}

func (renderer *Renderer) Close() {
	renderer.font_atlas.texture.Delete()
	renderer.font.Unload()
	RAPI_Close()
}
