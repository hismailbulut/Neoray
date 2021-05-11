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
	pos        ivec2
	texture_id int
}

type FontAtlas struct {
	textures   []*sdl.Texture
	pos        ivec2
	characters map[string]FontAtlasCharacter
}

type Renderer struct {
	handle      *sdl.Renderer
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
			textures:   make([]*sdl.Texture, 0),
			characters: make(map[string]FontAtlasCharacter),
		},
		cell_width:  cell_width,
		cell_height: cell_height,
	}
	sdl_renderer, err := window.handle.GetRenderer()
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize SDL renderer:", err)
	}
	renderer.handle = sdl_renderer
	if !renderer.handle.RenderTargetSupported() {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Neoray needs render target support.")
	}
	return renderer
}

// Returns the last empty texture id and position, and advances position.
// If texture is full than new texture will be created.
func (renderer *Renderer) GetEmptyAtlasPosition() (int, ivec2) {
	atlas := &renderer.font_atlas
	if len(atlas.textures) == 0 || (atlas.pos.X == 0 && atlas.pos.Y == 0) {
		// Create new atlas texture from surface
		texture, err := renderer.handle.CreateTexture(
			sdl.PIXELFORMAT_ARGB8888, sdl.TEXTUREACCESS_TARGET, FONT_ATLAS_DEFAULT_SIZE, FONT_ATLAS_DEFAULT_SIZE)
		if err != nil {
			log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Font atlas texture creation failed:", err)
		}
		// texture.SetBlendMode(sdl.BLENDMODE_BLEND)
		// append to textures list
		atlas.textures = append(renderer.font_atlas.textures, texture)
	}
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
	return len(atlas.textures) - 1, pos
}

func (renderer *Renderer) GetCharacterTextureAndPosition(char string, italic, bold bool) (*sdl.Texture, sdl.Rect) {
	var texture *sdl.Texture
	var position sdl.Rect
	// generate specific id for this character
	id := fmt.Sprintf("%s%t%t", char, italic, bold)
	if val, ok := renderer.font_atlas.characters[id]; ok == true {
		// use stored texture
		texture = renderer.font_atlas.textures[val.texture_id]
		position = sdl.Rect{
			X: int32(val.pos.X),
			Y: int32(val.pos.Y),
			W: int32(renderer.cell_width),
			H: int32(renderer.cell_height),
		}
	} else {
		// Create this text
		font_handle := renderer.font.GetDrawableFont(italic, bold)
		text_surface, err := font_handle.RenderUTF8Blended(char, COLOR_WHITE)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, err)
			return nil, sdl.Rect{}
		}
		defer text_surface.Free()
		// create texture from text surface
		text_texture, err := renderer.handle.CreateTextureFromSurface(text_surface)
		if err != nil {
			log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, err)
			return nil, sdl.Rect{}
		}
		atlas_texture_id, text_pos := renderer.GetEmptyAtlasPosition()
		atlas_texture := renderer.font_atlas.textures[atlas_texture_id]
		// calculate destination rectangle
		dest_rect := sdl.Rect{
			X: int32(text_pos.X),
			Y: int32(text_pos.Y),
			W: int32(renderer.cell_width),
			H: int32(renderer.cell_height),
		}
		// make height as same percent with width
		if text_surface.W > int32(renderer.cell_width) {
			dest_rect.H = int32(math.Ceil(float64(text_surface.H) / (float64(text_surface.W) / float64(dest_rect.W))))
		}
		// Draw text to empty position of atlas texture
		renderer.handle.SetRenderTarget(atlas_texture)
		renderer.handle.Copy(text_texture, nil, &dest_rect)
		renderer.handle.SetRenderTarget(nil)
		// Append this id to known character map
		renderer.font_atlas.characters[id] = FontAtlasCharacter{text_pos, atlas_texture_id}
		// Set variables for return
		texture = atlas_texture
		position = dest_rect
	}
	return texture, position
}

func (renderer *Renderer) DrawRectangle(rect sdl.Rect, color sdl.Color, batch bool) {
	renderer.handle.SetDrawColor(color.R, color.G, color.B, color.A)
	renderer.handle.FillRect(&rect)
}

func (renderer *Renderer) DrawCell(x, y int32, fg, bg sdl.Color, char string, italic, bold bool) {
	cell_rect := sdl.Rect{
		X: y * int32(renderer.cell_width),
		Y: x * int32(renderer.cell_height),
		W: int32(renderer.cell_width),
		H: int32(renderer.cell_height),
	}
	// Draw Background
	renderer.DrawRectangle(cell_rect, bg, true)
	if len(char) == 0 || char == " " {
		return
	}
	// Get character from atlas
	text_texture, position_rect := renderer.GetCharacterTextureAndPosition(char, italic, bold)
	if text_texture == nil {
		return
	}
	// Set text color
	text_texture.SetColorMod(fg.R, fg.G, fg.B)
	// Copy texture to main framebuffer
	renderer.handle.Copy(text_texture, &position_rect, &cell_rect)
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

func (renderer *Renderer) Draw(grid *Grid, mode *Mode, cursor *Cursor) {
	for x, row := range grid.cells {
		for y := range row {
			renderer.DrawCellWithAttrib(grid, int32(x), int32(y))
		}
	}
	cursor.Draw(grid, renderer, mode)
	// DEBUG Draw Last Font Atlas
	if len(renderer.font_atlas.textures) > 0 {
		last_texture := renderer.font_atlas.textures[len(renderer.font_atlas.textures)-1]
		last_texture_pos := sdl.Rect{
			X: int32((grid.width * renderer.cell_width) - int(FONT_ATLAS_DEFAULT_SIZE)),
			Y: 0,
			W: FONT_ATLAS_DEFAULT_SIZE,
			H: FONT_ATLAS_DEFAULT_SIZE,
		}
		last_texture.SetColorMod(255, 255, 255)
		renderer.handle.Copy(last_texture, nil, &last_texture_pos)
	}
	//
	renderer.handle.Present()
}

func (renderer *Renderer) Close() {
	for _, texture := range renderer.font_atlas.textures {
		texture.Destroy()
	}
	renderer.font.Unload()
	renderer.handle.Destroy()
}
