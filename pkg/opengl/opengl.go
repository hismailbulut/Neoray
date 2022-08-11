package opengl

import (
	_ "embed"
	"fmt"
	"unsafe"

	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/opengl/gl"
)

type ContextInfo struct {
	Version                string
	Vendor                 string
	Renderer               string
	ShadingLanguageVersion string
	MaxTextureSize         int32
}

type Context struct {
	shader      *ShaderProgram // default shader for monospaced font rendering
	framebuffer uint32         // only for clearing textures
}

// Call per window
func New(getProcAddress func(name string) unsafe.Pointer) (*Context, error) {
	// Initialize opengl
	err := gl.InitWithProcAddrFunc(getProcAddress)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize opengl: %s", err)
	}

	context := new(Context)

	// Init shaders
	vert := NewShaderFromSource(VERTEX_SHADER, ShaderSourceGridVert)
	geom := NewShaderFromSource(GEOMETRY_SHADER, ShaderSourceGridGeom)
	frag := NewShaderFromSource(FRAGMENT_SHADER, ShaderSourceGridFrag)
	context.shader = NewShaderProgram(vert, geom, frag)
	// Delete shaders because we don't need them after program creation
	vert.Delete()
	geom.Delete()
	frag.Delete()
	// TODO: When we start using multiple shaders this line will be deleted
	context.shader.Use()

	// Create framebuffer object
	// We dont need to bind framebuffer because we need it only when clearing texture
	gl.GenFramebuffers(1, &context.framebuffer)
	checkGLError()

	return context, nil
}

func (context *Context) Info() ContextInfo {
	info := ContextInfo{
		Version:                gl.GoStr(gl.GetString(gl.VERSION)),
		Vendor:                 gl.GoStr(gl.GetString(gl.VENDOR)),
		Renderer:               gl.GoStr(gl.GetString(gl.RENDERER)),
		ShadingLanguageVersion: gl.GoStr(gl.GetString(gl.SHADING_LANGUAGE_VERSION)),
	}
	gl.GetIntegerv(gl.MAX_TEXTURE_SIZE, &info.MaxTextureSize)
	return info
}

func (context *Context) SetViewport(rect common.Rectangle[int]) {
	gl.Viewport(int32(rect.X), int32(rect.Y), int32(rect.W), int32(rect.H))
	checkGLError()
}

func (context *Context) ClearScreen(c common.Color) {
	gl.ClearColor(c.R, c.G, c.B, c.A)
	checkGLError()
	gl.Clear(gl.COLOR_BUFFER_BIT)
	checkGLError()
}

func (context *Context) Flush() {
	// Since we are not using doublebuffering, we don't need to swap buffers, but we need to flush.
	gl.Flush()
	checkGLError()
}

func (context *Context) Destroy() {
	// Delete framebuffer
	gl.DeleteFramebuffers(1, &context.framebuffer)
	// Delete shaders
	context.shader.Destroy()
}

func checkGLError() {
	error_code := gl.GetError()
	if error_code == gl.NO_ERROR {
		return
	}
	var errorName string
	switch error_code {
	case gl.INVALID_ENUM:
		errorName = "INVALID_ENUM"
	case gl.INVALID_VALUE:
		errorName = "INVALID_VALUE"
	case gl.INVALID_OPERATION:
		errorName = "INVALID_OPERATION"
	case gl.STACK_OVERFLOW:
		errorName = "STACK_OVERFLOW"
	case gl.STACK_UNDERFLOW:
		errorName = "STACK_UNDERFLOW"
	case gl.OUT_OF_MEMORY:
		errorName = "OUT_OF_MEMORY"
	case gl.CONTEXT_LOST:
		errorName = "CONTEXT_LOST"
	default:
		errorName = fmt.Sprintf("#%.4x", error_code)
	}
	panic(fmt.Errorf("Opengl Error: %s", errorName))
}
