package opengl

import (
	"fmt"
	"image"
	"unsafe"

	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/logger"
	"github.com/hismailbulut/Neoray/pkg/opengl/gl"
)

var boundTextureId uint32

type Texture struct {
	id     uint32
	width  int
	height int
	fbo    uint32 // this is like pointer to the framebuffer because framebuffer is in gpu memory
}

func (texture Texture) String() string {
	return fmt.Sprintf("Texture(ID: %d, Width: %d, Height: %d)", texture.id, texture.width, texture.height)
}

func (context *Context) CreateTexture(width, height int) Texture {
	texture := Texture{
		fbo: context.framebuffer,
	}
	// NOTE: There can be multiple textures but only one can bind at a time
	gl.GenTextures(1, &texture.id)
	checkGLError()
	gl.BindTexture(gl.TEXTURE_2D, texture.id)
	checkGLError()
	boundTextureId = texture.id
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	checkGLError()
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	checkGLError()
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	checkGLError()
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	checkGLError()
	texture.Resize(width, height)
	logger.Log(logger.DEBUG, "Texture created:", texture)
	return texture
}

func (texture *Texture) Size() common.Vector2[int] {
	return common.Vec2(texture.width, texture.height)
}

// Texture must bound before resizing
func (texture *Texture) Resize(width, height int) {
	if boundTextureId != texture.id {
		panic("texture must be bound before resize")
	}
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	checkGLError()
	texture.width = width
	texture.height = height
}

func (texture *Texture) Clear() {
	// Bind framebuffer
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, texture.fbo)
	checkGLError()
	// Init framebuffer with texture
	gl.FramebufferTexture2D(gl.DRAW_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, texture.id, 0)
	checkGLError()
	// Check if the framebuffer is complete and ready for draw
	fbo_status := gl.CheckFramebufferStatus(gl.DRAW_FRAMEBUFFER)
	if fbo_status == gl.FRAMEBUFFER_COMPLETE {
		// Clear the texture
		gl.ClearColor(0, 0, 0, 0)
		checkGLError()
		gl.Clear(gl.COLOR_BUFFER_BIT)
		checkGLError()
	} else {
		panic(fmt.Errorf("Framebuffer is not complete: %d", fbo_status))
	}
	// Unbind framebuffer
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0)
	checkGLError()
}

func (texture *Texture) Bind() {
	if boundTextureId == texture.id {
		return
	}
	gl.BindTexture(gl.TEXTURE_2D, texture.id)
	checkGLError()
	boundTextureId = texture.id
}

// Texture must bound before drawing
func (texture *Texture) Draw(image *image.RGBA, dest common.Rectangle[int]) {
	if boundTextureId != texture.id {
		panic("texture must be bound before resize")
	}
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(dest.X), int32(dest.Y), int32(dest.W), int32(dest.H), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&image.Pix[0]))
	checkGLError()
}

// Converts coordinates to opengl understandable coordinates, 0 to 1
func (texture *Texture) Normalize(pos common.Rectangle[int]) common.Rectangle[float32] {
	return common.Rectangle[float32]{
		X: float32(pos.X) / float32(texture.width),
		Y: float32(pos.Y) / float32(texture.height),
		W: float32(pos.W) / float32(texture.width),
		H: float32(pos.H) / float32(texture.height),
	}
}

func (texture *Texture) Delete() {
	gl.DeleteTextures(1, &texture.id)
	texture.id = 0
	texture.width = 0
	texture.height = 0
	texture.fbo = 0
	logger.Log(logger.DEBUG, "Texture deleted:", texture)
}
