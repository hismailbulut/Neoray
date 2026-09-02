package opengl

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/fontkit"
	"github.com/hismailbulut/Neoray/pkg/logger"
)

const (
	UNSUPPORTED_GLYPH_ID uint64 = 0xffffffffffffffff // "Unsupported"
	UNDERCURL_GLYPH_ID   uint64 = 0xfffffffffffffffe // "Undercurl"

	UNICODE_REPLACEMENT_CHARACTER rune = 0xfffd
)

type Atlas struct {
	fontSize, dpi   float64
	useBoxDrawing   bool
	useBlockDrawing bool
	texture         Texture
	cache           map[uint64]common.Rectangle[int]
	pen             common.Vector2[int]
	cellSize        common.Vector2[int]
}

func (atlas *Atlas) String() string {
	return fmt.Sprintf("Atlas(ID: %d, Font Size: %f, Pen: %v)",
		atlas.texture.id,
		atlas.fontSize,
		atlas.pen,
	)
}

func (context *Context) NewAtlas(size, dpi float64, useBoxDrawing, useBlockDrawing bool) *Atlas {
	atlas := new(Atlas)
	atlas.fontSize = size
	atlas.dpi = dpi
	atlas.useBoxDrawing = useBoxDrawing
	atlas.useBlockDrawing = useBlockDrawing
	// 512 * 512 * RGBA8 = 1mib
	const width = 512
	const height = 512
	// In most cases these size of texture is highly enough
	// But we also grow it if needed
	atlas.texture = context.CreateTexture(width, height)
	atlas.cache = make(map[uint64]common.Rectangle[int])
	return atlas
}

func (atlas *Atlas) FontSize() float64 {
	return atlas.fontSize
}

func (atlas *Atlas) SetFontSize(fontSize, dpi float64) {
	atlas.fontSize = fontSize
	atlas.dpi = dpi
	atlas.Reset()
}

func (atlas *Atlas) SetBoxDrawing(useBoxDrawing, useBlockDrawing bool) {
	atlas.useBoxDrawing = useBoxDrawing
	atlas.useBlockDrawing = useBlockDrawing
	atlas.Reset()
}

func (atlas *Atlas) CalculateCellSizeForFont(font *fontkit.Font) {
	face, err := font.CreateFace(fontkit.FaceParams{
		Size:            atlas.fontSize,
		DPI:             atlas.dpi,
		UseBoxDrawing:   atlas.useBoxDrawing,
		UseBlockDrawing: atlas.useBlockDrawing,
	})
	if err != nil {
		panic(err)
	}
	atlas.cellSize = face.ImageSize()
}

func (atlas *Atlas) CellSize() common.Vector2[int] {
	return atlas.cellSize
}

func (atlas *Atlas) Reset() {
	atlas.texture.Clear()
	atlas.cache = make(map[uint64]common.Rectangle[int])
	atlas.pen = common.Vector2[int]{}
}

func getCharID(char rune, italic, bold, underline, strikethrough bool) uint64 {
	id := uint64(char)
	if italic {
		id = id | uint64(1)<<32
	}
	if bold {
		id = id | uint64(1)<<40
	}
	if underline {
		id = id | uint64(1)<<48
	}
	if strikethrough {
		id = id | uint64(1)<<56
	}
	return id
}

// Draws img to texture and returns position
func (atlas *Atlas) drawImage(img *image.RGBA) common.Rectangle[int] {
	texSize := atlas.TextureSize()
	// Check X
	if atlas.pen.X+img.Rect.Dx() > texSize.Width() {
		atlas.pen.X = 0
		atlas.pen.Y += img.Rect.Dy()
	}
	// Check Y
	if atlas.pen.Y+img.Rect.Dy() > texSize.Height() {
		// We must grow the texture
		// TODO: instead of clearing the texture we can create a bigger texture and copy this texture on to it
		// but for now this is easier and doesn't consume too much resource
		atlas.texture.Bind()
		atlas.texture.Resize(texSize.Width()*2, texSize.Height()*2)
		logger.Debug("Atlas", atlas.texture.id, "texture resized to", atlas.TextureSize())
		// Resizing texture also clears it, so we should also clear the cache
		atlas.cache = make(map[uint64]common.Rectangle[int])
		atlas.pen = common.Vector2[int]{}
	}
	// draw image to current pen
	dest := common.Rect(atlas.pen.X, atlas.pen.Y, img.Rect.Dx(), img.Rect.Dy())
	// We should bind texture before drawing to it
	atlas.texture.Bind()
	atlas.texture.Draw(img, dest)
	// increment pen
	atlas.pen.X += img.Rect.Dx()
	return dest
}

func (atlas *Atlas) drawChar(face *fontkit.Face, id uint64, char rune, underline, strikethrough bool) common.Rectangle[int] {
	img := face.RenderChar(char, underline, strikethrough, atlas.cellSize)
	if img == nil {
		logger.Error("face.RenderChar() failed for glyph:", char, underline, strikethrough)
		return common.ZeroRectangleINT
	}
	pos := atlas.drawImage(img)
	atlas.cache[id] = pos
	return pos
}

// For the first time draws and caches undercurl image, returns image pos and true representing first time
// After that uses cached image and returns false
// The third bool value is whether the atlas has resized, draw everything again if it resized
func (atlas *Atlas) Undercurl(font *fontkit.Font) (common.Rectangle[int], bool, bool) {
	pos, ok := atlas.cache[UNDERCURL_GLYPH_ID]
	if ok {
		return pos, false, false
	}
	// Draw and cache
	face, err := font.CreateFace(fontkit.FaceParams{
		Size:            atlas.fontSize,
		DPI:             atlas.dpi,
		UseBoxDrawing:   atlas.useBoxDrawing,
		UseBlockDrawing: atlas.useBlockDrawing,
	})
	if err != nil {
		panic(fmt.Errorf("face creation failed: %s", err))
	}
	img := face.RenderUndercurl(atlas.cellSize)
	texSize := atlas.TextureSize()
	pos = atlas.drawImage(img)
	atlas.cache[UNDERCURL_GLYPH_ID] = pos
	return pos, true, !texSize.Equals(atlas.TextureSize())
}

// Use this if the font doesn't contain replacement character
func (atlas *Atlas) unsupported() common.Rectangle[int] {
	pos, ok := atlas.cache[UNSUPPORTED_GLYPH_ID]
	if ok {
		return pos
	}
	// Draw a simple rectangle representing an unsupported glyph
	img := image.NewRGBA(image.Rect(0, 0, atlas.cellSize.Width(), atlas.cellSize.Height()))
	gap := int(math.Ceil(float64(atlas.cellSize.Width()) / 10.0))
	x0, y0 := gap, gap
	x1 := max((atlas.cellSize.Width()-1)-gap, 0)
	y1 := atlas.cellSize.Height() - gap
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			// TODO: font characters line width
			if x == x0 || x == x1 || y == y0 || y == y1 {
				img.SetRGBA(x, y, color.RGBA{255, 255, 255, 255})
			}
		}
	}
	pos = atlas.drawImage(img)
	atlas.cache[UNSUPPORTED_GLYPH_ID] = pos
	return pos
}

// Returns position of the character in atlas and whether the atlas has been resized
// If atlas has resized then everything should be rendered again because atlas texture clears
func (atlas *Atlas) GetCharPos(font *fontkit.Font, char rune, bold, italic, underline, strikethrough bool) (common.Rectangle[int], bool) {
	id := getCharID(char, italic, bold, underline, strikethrough)
	pos, ok := atlas.cache[id]
	if ok {
		return pos, false
	}
	faceParams := fontkit.FaceParams{
		Size:            atlas.fontSize,
		DPI:             atlas.dpi,
		UseBoxDrawing:   atlas.useBoxDrawing,
		UseBlockDrawing: atlas.useBlockDrawing,
	}
	face, err := font.CreateFaceForGlyph(faceParams, char)
	if err != nil {
		// Use any font
		face, err = font.CreateFace(faceParams)
		if err != nil {
			panic(fmt.Errorf("face creation failed: %s", err))
		}
	}
	prevTextureSize := atlas.TextureSize()
	if font.ContainsGlyph(char) {
		pos = atlas.drawChar(face, id, char, underline, strikethrough)
	} else if font.ContainsGlyph(UNICODE_REPLACEMENT_CHARACTER) {
		pos = atlas.drawChar(face, UNSUPPORTED_GLYPH_ID, UNICODE_REPLACEMENT_CHARACTER, false, false)
	} else {
		pos = atlas.unsupported()
	}
	return pos, !prevTextureSize.Equals(atlas.TextureSize())
}

func (atlas *Atlas) TextureSize() common.Vector2[int] {
	return atlas.texture.Size()
}

// Normalization required when updating texture position to the gpu
func (atlas *Atlas) Normalize(pos common.Rectangle[int]) common.Rectangle[float32] {
	return atlas.texture.Normalize(pos)
}

func (atlas *Atlas) BindTexture() {
	atlas.texture.Bind()
}

func (atlas *Atlas) Delete() {
	atlas.texture.Delete()
}
