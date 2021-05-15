package main

import (
	"fmt"
	"math"

	"github.com/veandco/go-sdl2/sdl"
)

var (
	FONT_ATLAS_DEFAULT_SIZE int32 = 256
)

type FontAtlasCharacter struct {
	pos      ivec2
	atlas_id int
}

type FontAtlasPair struct {
	surface *sdl.Surface
	texture Texture
}

type FontAtlas struct {
	pairs      []FontAtlasPair
	pos        ivec2
	characters map[string]FontAtlasCharacter
}

// RAPI_DrawSubTextureColor(atlas.texture, &atlas_char_pos, &cell_rect, fg)
// RAPI_FillRect(rect, color)
type CellDrawInfo struct {
	draw_font bool
	// For font drawings
	atlas     Texture
	atlas_src sdl.Rect
	fg_dest   sdl.Rect
	fg_color  sdl.Color
	// for background (always send)
	bg_dest  sdl.Rect
	bg_color sdl.Color
}

type Renderer struct {
	font        Font
	font_atlas  FontAtlas
	cell_width  int
	cell_height int
	cell_infos  [][]CellDrawInfo
}

func CreateRenderer(window *Window, font Font) Renderer {
	cell_width, cell_height := font.CalculateCellSize()
	renderer := Renderer{
		font: font,
		font_atlas: FontAtlas{
			pairs:      make([]FontAtlasPair, 0),
			characters: make(map[string]FontAtlasCharacter),
		},
		cell_width:  cell_width,
		cell_height: cell_height,
	}

	RAPI_Init(window)
	RAPI_CreateViewport(window.width, window.height)

	return renderer
}

func (renderer *Renderer) Resize(row_count, col_count int) {
	// renderer.cell_infos = make([][]CellDrawInfo, row_count)
	// for i := range renderer.cell_infos {
	//     renderer.cell_infos[i] = make([]CellDrawInfo, col_count)
	// }
}

// Returns the last empty texture id and position, and advances position.
// If texture is full than new texture will be created.
func (renderer *Renderer) GetEmptyAtlasPosition() (int, ivec2) {
	atlas := &renderer.font_atlas
	if len(atlas.pairs) == 0 || (atlas.pos.X == 0 && atlas.pos.Y == 0) {
		// create new atlas texture
		surface, err := sdl.CreateRGBSurfaceWithFormat(
			0, FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE, 32, sdl.PIXELFORMAT_RGBA8888)
		if err != nil {
			log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Font atlas surface creation failed:", err)
		}
		// generate opengl texture
		texture := RAPI_CreateTextureFromSurface(surface)
		pair := FontAtlasPair{surface: surface, texture: texture}
		// append to textures list
		atlas.pairs = append(renderer.font_atlas.pairs, pair)
	}
	// calculate position
	pos := atlas.pos
	atlas.pos.X += renderer.cell_width
	if atlas.pos.X+renderer.cell_width > int(FONT_ATLAS_DEFAULT_SIZE) {
		// New row
		atlas.pos.X = 0
		atlas.pos.Y += renderer.cell_height
	}
	if atlas.pos.Y+renderer.cell_height > int(FONT_ATLAS_DEFAULT_SIZE) {
		// Next time new texture will be created.
		atlas.pos = ivec2{}
	}
	// Return last texture and pos
	return len(atlas.pairs) - 1, pos
}

func (renderer *Renderer) GetCharacterAtlasPosition(char string, italic, bold bool) (FontAtlasPair, sdl.Rect, error) {
	var atlas FontAtlasPair
	var position sdl.Rect
	// generate specific id for this character
	id := fmt.Sprintf("%s%t%t", char, italic, bold)
	if val, ok := renderer.font_atlas.characters[id]; ok == true {
		// use stored texture
		atlas = renderer.font_atlas.pairs[val.atlas_id]
		position = sdl.Rect{
			X: int32(val.pos.X),
			Y: int32(val.pos.Y),
			W: int32(renderer.cell_width),
			H: int32(renderer.cell_height),
		}
		// For Debug
		// if len(renderer.font_atlas.characters) > 25 {
		//     if err := atlas.surface.SaveBMP("surface.bmp"); err != nil {
		//         log_debug_msg(err)
		//     }
		//     log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "BMP Saved.")
		// }
	} else {
		// Create this text
		font_handle := renderer.font.GetDrawableFont(italic, bold)
		text_surface, err := font_handle.RenderUTF8Blended(char, COLOR_WHITE)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, err)
			return FontAtlasPair{}, sdl.Rect{}, err
		}
		defer text_surface.Free()
		// get new position and current atlas texture
		atlas_texture_id, text_pos := renderer.GetEmptyAtlasPosition()
		atlas = renderer.font_atlas.pairs[atlas_texture_id]
		// calculate destination rectangle
		position = sdl.Rect{
			X: int32(text_pos.X),
			Y: int32(text_pos.Y),
			W: int32(renderer.cell_width),
			H: int32(renderer.cell_height),
		}
		// make height as same percent with width
		if text_surface.W > int32(renderer.cell_width) {
			position.H = int32(math.Ceil(float64(text_surface.H) / (float64(text_surface.W) / float64(position.W))))
		}
		// Draw text to empty position of atlas texture
		// TODO: Draw directly to texture
		text_surface.Blit(nil, atlas.surface, &position)
		// Update opengl texture
		atlas.texture.UpdateFromSurface(atlas.surface)
		// append this id to known character map
		renderer.font_atlas.characters[id] = FontAtlasCharacter{text_pos, atlas_texture_id}
	}
	return atlas, position, nil
}

func (renderer *Renderer) DrawRectangle(x, y int32, rect sdl.Rect, color sdl.Color) {
	// renderer.cell_infos[x][y].bg_dest = rect
	// renderer.cell_infos[x][y].bg_color = color
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
	renderer.DrawRectangle(x, y, cell_rect, bg)
	if len(char) == 0 || char == " " {
		// renderer.cell_infos[x][y].draw_font = false
		return
	}
	// get character atlas texture and position
	atlas, atlas_char_pos, err := renderer.GetCharacterAtlasPosition(char, italic, bold)
	if err != nil {
		return
	}
	// draw
	// renderer.cell_infos[x][y].draw_font = true
	// renderer.cell_infos[x][y].atlas = atlas.texture
	// renderer.cell_infos[x][y].atlas_src = atlas_char_pos
	// renderer.cell_infos[x][y].fg_dest = cell_rect
	// renderer.cell_infos[x][y].fg_color = fg
	RAPI_DrawSubTextureColor(atlas.texture, &atlas_char_pos, &cell_rect, fg)
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

	// Send buffer to render subsystem
	// for _, row := range renderer.cell_infos {
	//     for _, cell := range row {
	//         RAPI_FillRect(cell.bg_dest, cell.bg_color)
	//         if cell.draw_font {
	//             RAPI_DrawSubTextureColor(
	//                 cell.atlas, &cell.atlas_src, &cell.fg_dest, cell.fg_color)
	//         }
	//     }
	// }

	// DEBUG: prepare last font atlas for drawing
	if len(renderer.font_atlas.pairs) > 0 {
		atlas := renderer.font_atlas.pairs[len(renderer.font_atlas.pairs)-1]
		atlas_pos := sdl.Rect{
			X: int32((editor.grid.width * renderer.cell_width) - int(FONT_ATLAS_DEFAULT_SIZE)),
			Y: 0,
			W: FONT_ATLAS_DEFAULT_SIZE,
			H: FONT_ATLAS_DEFAULT_SIZE,
		}
		RAPI_DrawTexture(atlas.texture, &atlas_pos)
	}

	// Draw
	RAPI_Render()

	// Swap window surface
	editor.window.handle.GLSwap()
}

func (renderer *Renderer) Close() {
	for _, atlas := range renderer.font_atlas.pairs {
		atlas.surface.Free()
		atlas.texture.Delete()
	}
	renderer.font.Unload()
	RAPI_Close()
}
