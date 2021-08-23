package main

import (
	"image"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
)

var (
	TextureClearColor = [4]byte{0, 0, 0, 0}
)

type Texture struct {
	id     uint32
	width  int
	height int
}

func CreateTexture(width, height int) Texture {
	var texture_id uint32
	gl.GenTextures(1, &texture_id)
	gl.BindTexture(gl.TEXTURE_2D, texture_id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	rglCheckError("texture params")
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	rglCheckError("texture teximage2d")
	texture := Texture{
		id:     texture_id,
		width:  width,
		height: height,
	}
	return texture
}

func (texture *Texture) bind() {
	gl.BindTexture(gl.TEXTURE_2D, texture.id)
}

func (texture *Texture) clear() {
	gl.ClearTexImage(texture.id, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&TextureClearColor[0]))
	rglCheckError("texture clear")
}

func (texture *Texture) updatePart(image *image.RGBA, dest IntRect) {
	gl.TexSubImage2D(gl.TEXTURE_2D, 0,
		int32(dest.X), int32(dest.Y), int32(dest.W), int32(dest.H),
		gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&image.Pix[0]))
	rglCheckError("texture update part")
}

func (texture *Texture) glCoords(pos IntRect) F32Rect {
	return F32Rect{
		X: float32(pos.X) / float32(texture.width),
		Y: float32(pos.Y) / float32(texture.height),
		W: float32(pos.W) / float32(texture.width),
		H: float32(pos.H) / float32(texture.height),
	}
}

func (texture *Texture) Delete() {
	gl.DeleteTextures(1, &texture.id)
}
