package opengl

import (
	_ "embed"
	"fmt"
	"runtime"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/hismailbulut/neoray/src/common"
)

type ContextInfo struct {
	Version                string
	Vendor                 string
	Renderer               string
	ShadingLanguageVersion string
	MaxTextureSize         int32
}

type Context struct {
	shader      ShaderProgram // default shader for monospaced font rendering
	framebuffer uint32        // only for clearing textures
}

// Call per window
func New() (*Context, error) {
	// Initialize opengl
	err := gl.Init()
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize opengl: %s", err)
	}

	context := new(Context)

	// Init shaders
	context.shader = DefaultProgram()
	context.shader.Use()

	// Create framebuffer object
	// We dont need to bind framebuffer because we need it only when clearing texture
	CheckGLError(func() {
		gl.GenFramebuffers(1, &context.framebuffer)
	})

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
	CheckGLError(func() {
		gl.Viewport(int32(rect.X), int32(rect.Y), int32(rect.W), int32(rect.H))
	})
}

func (context *Context) ClearScreen(color common.U8Color) {
	c := color.ToF32()
	CheckGLError(func() {
		gl.ClearColor(c.R, c.G, c.B, c.A)
		gl.Clear(gl.COLOR_BUFFER_BIT)
	})
}

func (context *Context) Flush() {
	// Since we are not using doublebuffering, we don't need to swap buffers, but we need to flush.
	CheckGLError(func() {
		gl.Flush()
	})
}

func (context *Context) Destroy() {
	// Delete framebuffer
	gl.DeleteFramebuffers(1, &context.framebuffer)
	// Delete shaders
	context.shader.Destroy()
}

// If any opengl error happens, prints error to stdout and returns false
func CheckGLError(glFunc func()) {
	// Call opengl function
	glFunc()
	// Check for error
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

	// Get caller function name
	callerName := "unknown"
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		callerName = runtime.FuncForPC(pc).Name()
	}

	panic(fmt.Errorf("Opengl Error: %s on %s", errorName, callerName))
}
