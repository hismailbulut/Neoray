package opengl

import (
	"fmt"
	"image"
	"unsafe"

	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/opengl/gl"
)

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
	var id uint32
	// NOTE: There can be multiple textures but only one can bind at a time
	CheckGLError(func() {
		gl.GenTextures(1, &id)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, id)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	})
	texture := Texture{
		id:  id,
		fbo: context.framebuffer,
	}
	texture.Resize(width, height)
	return texture
}

func (texture *Texture) Size() common.Vector2[int] {
	return common.Vec2(texture.width, texture.height)
}

// Texture must bound before resizing
func (texture *Texture) Resize(width, height int) {
	CheckGLError(func() {
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	})
	texture.width = width
	texture.height = height
}

func (texture *Texture) Clear() {
	CheckGLError(func() {
		// Bind framebuffer
		gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, texture.fbo)
		// Init framebuffer with texture
		gl.FramebufferTexture2D(gl.DRAW_FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, texture.id, 0)
	})
	// Check if the framebuffer is complete and ready for draw
	fbo_status := gl.CheckFramebufferStatus(gl.DRAW_FRAMEBUFFER)
	if fbo_status == gl.FRAMEBUFFER_COMPLETE {
		// Clear the texture
		CheckGLError(func() {
			gl.ClearColor(0, 0, 0, 0)
			gl.Clear(gl.COLOR_BUFFER_BIT)
		})
	} else {
		// NOTE: We can just print an error and recreate the texture
		panic(fmt.Errorf("Framebuffer is not complete: %d", fbo_status))
	}
	CheckGLError(func() {
		// Unbind framebuffer
		gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0)
	})
}

func (texture *Texture) Bind() {
	CheckGLError(func() {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texture.id)
	})
}

// Texture must bound before drawing
func (texture *Texture) Draw(image *image.RGBA, dest common.Rectangle[int]) {
	CheckGLError(func() {
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, int32(dest.X), int32(dest.Y), int32(dest.W), int32(dest.H), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&image.Pix[0]))
	})
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
}
