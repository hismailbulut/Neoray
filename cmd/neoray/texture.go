package main

import (
	"image"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
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
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	RGL_CheckError("CreateTexture.TexParameteri")
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	RGL_CheckError("CreateTexture.TexImage2D")
	texture := Texture{
		id:     texture_id,
		width:  width,
		height: height,
	}
	return texture
}

func (texture *Texture) Bind() {
	gl.BindTexture(gl.TEXTURE_2D, texture.id)
}

func (texture *Texture) Clear() {
	gl.ClearTexImage(texture.id, 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	RGL_CheckError("Texture.Clear")
}

func (texture *Texture) UpdatePartFromImage(image *image.RGBA, dest sdl.Rect) {
	gl.TexSubImage2D(gl.TEXTURE_2D, 0,
		dest.X, dest.Y, dest.W, dest.H,
		gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&image.Pix[0]))
	RGL_CheckError("Texture.UpdatePartFromSurface")
}

func (texture *Texture) GetRectGLCoordinates(rect sdl.Rect) sdl.FRect {
	return sdl.FRect{
		X: float32(rect.X) / float32(texture.width),
		Y: float32(rect.Y) / float32(texture.height),
		W: float32(rect.W) / float32(texture.width),
		H: float32(rect.H) / float32(texture.height),
	}
}

func (texture *Texture) Delete() {
	gl.DeleteTextures(1, &texture.id)
}
