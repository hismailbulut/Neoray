package opengl

import (
	"fmt"
	"image"

	"github.com/hismailbulut/neoray/src/common"
	"github.com/hismailbulut/neoray/src/fontkit"
)

const (
	UNSUPPORTED_GLYPH_ID = 0xffffffffffffffff // "Unsupported"
	UNDERCURL_GLYPH_ID   = 0xfffffffffffffffe // "Undercurl"
)

type Atlas struct {
	kit             *fontkit.FontKit
	fontSize, dpi   float64
	useBoxDrawing   bool
	useBlockDrawing bool
	texture         Texture
	cache           map[uint64]common.Rectangle[int]
	pen             common.Vector2[int]
}

func (atlas *Atlas) String() string {
	return fmt.Sprintf("Atlas(ID: %d, Font Size: %f, Pen: %v)",
		atlas.texture.id,
		atlas.fontSize,
		atlas.pen,
	)
}

func (context *Context) NewAtlas(kit *fontkit.FontKit, size, dpi float64, useBoxDrawing, useBlockDrawing bool) *Atlas {
	atlas := new(Atlas)
	atlas.kit = kit
	atlas.fontSize = size
	atlas.dpi = dpi
	atlas.useBoxDrawing = useBoxDrawing
	atlas.useBlockDrawing = useBlockDrawing
	// 4096 * 512 = 2Mib
	const width = 4096
	const height = 512
	// In most cases these size of texture is highly enough
	// But we also grow it if needed
	atlas.texture = context.CreateTexture(width, height)
	atlas.cache = make(map[uint64]common.Rectangle[int])
	return atlas
}

func (atlas *Atlas) FontKit() *fontkit.FontKit {
	if atlas.kit != nil {
		return atlas.kit
	}
	return fontkit.Default()
}

func (atlas *Atlas) SetFontKit(kit *fontkit.FontKit) {
	atlas.kit = kit
	atlas.Reset()
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

func (atlas *Atlas) Reset() {
	atlas.texture.Clear()
	for k := range atlas.cache {
		delete(atlas.cache, k)
	}
	atlas.pen = common.Vector2[int]{}
}

func (atlas *Atlas) ImageSize() common.Vector2[int] {
	face, err := atlas.FontKit().DefaultFont().CreateFace(fontkit.FaceParams{
		Size:            atlas.fontSize,
		DPI:             atlas.dpi,
		UseBoxDrawing:   atlas.useBoxDrawing,
		UseBlockDrawing: atlas.useBlockDrawing,
	})
	if err != nil {
		panic(err)
	}
	return face.ImageSize()
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
	textureSize := atlas.texture.Size()
	// Check X
	if atlas.pen.X+img.Rect.Dx() > textureSize.Width() {
		atlas.pen.X = 0
		atlas.pen.Y += img.Rect.Dy()
	}
	// Check Y
	if atlas.pen.Y+img.Rect.Dy() > textureSize.Height() {
		// We must grow the texture
		atlas.texture.Resize(textureSize.Width(), textureSize.Height()*2)
		// Resizing texture also clears it, so we should also clear the cache
		for k := range atlas.cache {
			delete(atlas.cache, k)
		}
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

func (atlas *Atlas) drawChar(face *fontkit.Face, id uint64, char rune, underline, strikethrough bool, imgSize common.Vector2[int]) common.Rectangle[int] {
	img := face.RenderChar(char, underline, strikethrough, imgSize)
	pos := atlas.drawImage(img)
	atlas.cache[id] = pos
	return pos
}

func (atlas *Atlas) suitableFont(char rune, bold, italic bool) (*fontkit.Font, bool) {
	if atlas.FontKit().SuitableFont(bold, italic).ContainsGlyph(char) {
		return atlas.FontKit().SuitableFont(bold, italic), true
	}
	if atlas.FontKit().DefaultFont().ContainsGlyph(char) {
		return atlas.FontKit().DefaultFont(), true
	}
	if fontkit.Default().SuitableFont(bold, italic).ContainsGlyph(char) {
		return fontkit.Default().SuitableFont(bold, italic), true
	}
	if fontkit.Default().DefaultFont().ContainsGlyph(char) {
		return fontkit.Default().DefaultFont(), true
	}
	// Neither of the fonts supports this glyph
	return atlas.FontKit().SuitableFont(bold, italic), false
}

// For the first time draws and caches undercurl image, returns image pos and true representing first time
// After that uses cached image and returns false
func (atlas *Atlas) Undercurl(imgSize common.Vector2[int]) (common.Rectangle[int], bool) {
	pos, ok := atlas.cache[UNDERCURL_GLYPH_ID]
	if ok {
		return pos, false
	}
	// Draw and cache
	face, err := atlas.FontKit().DefaultFont().CreateFace(fontkit.FaceParams{
		Size:            atlas.fontSize,
		DPI:             atlas.dpi,
		UseBoxDrawing:   atlas.useBoxDrawing,
		UseBlockDrawing: atlas.useBlockDrawing,
	})
	if err != nil {
		panic(fmt.Errorf("face creation failed: %s", err))
	}
	img := face.RenderUndercurl(imgSize)
	pos = atlas.drawImage(img)
	atlas.cache[UNDERCURL_GLYPH_ID] = pos
	return pos, true
}

func (atlas *Atlas) unsupported(face *fontkit.Face, char rune, imgSize common.Vector2[int]) common.Rectangle[int] {
	pos, ok := atlas.cache[UNSUPPORTED_GLYPH_ID]
	if ok {
		return pos
	}
	// Draw and cache
	face, err := atlas.FontKit().DefaultFont().CreateFace(fontkit.FaceParams{
		Size:            atlas.fontSize,
		DPI:             atlas.dpi,
		UseBoxDrawing:   atlas.useBoxDrawing,
		UseBlockDrawing: atlas.useBlockDrawing,
	})
	if err != nil {
		panic(fmt.Errorf("face creation failed: %s", err))
	}
	// Draw and cache
	pos = atlas.drawChar(face, UNSUPPORTED_GLYPH_ID, char, false, false, imgSize)
	return pos
}

func (atlas *Atlas) GetCharPos(char rune, bold, italic, underline, strikethrough bool, imgSize common.Vector2[int]) common.Rectangle[int] {
	id := getCharID(char, italic, bold, underline, strikethrough)
	pos, ok := atlas.cache[id]
	if ok {
		return pos
	}
	font, contains := atlas.suitableFont(char, bold, italic)
	face, err := font.CreateFace(fontkit.FaceParams{
		Size:            atlas.fontSize,
		DPI:             atlas.dpi,
		UseBoxDrawing:   atlas.useBoxDrawing,
		UseBlockDrawing: atlas.useBlockDrawing,
	})
	if err != nil {
		panic(fmt.Errorf("face creation failed: %s", err))
	}
	if contains {
		// Draw and cache
		pos = atlas.drawChar(face, id, char, underline, strikethrough, imgSize)
		return pos
	} else {
		// unsupported
		return atlas.unsupported(face, char, imgSize)
	}
}

// Normalization required when updating texture position to the gpu
func (atlas *Atlas) Normalize(pos common.Rectangle[int]) common.Rectangle[float32] {
	return atlas.texture.Normalize(pos)
}

func (atlas *Atlas) BindTexture() {
	atlas.texture.Bind()
}

func (atlas *Atlas) Destroy() {
	atlas.texture.Delete()
}
