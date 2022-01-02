package main

import (
	_ "embed"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

type Vertex struct {
	// position of this vertex
	pos F32Rect // layout 0
	// texture position
	tex1 F32Rect // layout 1
	// second texture position used for multiwidth characters
	tex2 F32Rect // layout 2
	// foreground color
	fg F32Color // layout 3
	// background color
	bg F32Color // layout 4
	// special color
	sp F32Color // layout 5
}

const sizeof_Vertex = int32(unsafe.Sizeof(Vertex{}))

// renderer opengl global variables
var RGL struct {
	vao               uint32 // Vertex Array Object
	vbo               uint32 // Vertex Buffer Object
	fbo               uint32 // Framebuffer Object (Only used for clearing textures)
	shader_program    uint32
	vertex_buffer_len int // Length of the vertex data is equals to rendered quads
}

//go:embed shader.glsl
var EmbeddedShaderSources string

func rglInit() {
	defer measure_execution_time()()

	logMessage(LEVEL_DEBUG, TYPE_RENDERER, "Initializing opengl.")
	// Initialize opengl
	if err := gl.InitWithProcAddrFunc(glfw.GetProcAddress); err != nil {
		logMessage(LEVEL_FATAL, TYPE_RENDERER, "Failed to initialize opengl:", err)
	}

	// Init shaders
	rglInitShaders()
	gl.UseProgram(RGL.shader_program)
	rglCheckError("use program")

	// Initialize vao
	gl.GenVertexArrays(1, &RGL.vao)
	gl.BindVertexArray(RGL.vao)
	rglCheckError("gen vao")

	// Initialize vbo
	gl.GenBuffers(1, &RGL.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, RGL.vbo)
	rglCheckError("gen vbo")

	// Enable attributes
	valueof_Vertex := reflect.ValueOf(Vertex{})
	offset := uintptr(0)
	for i := 0; i < valueof_Vertex.NumField(); i++ {
		attr_size := valueof_Vertex.Field(i).Type().Size()
		gl.EnableVertexAttribArray(uint32(i))
		gl.VertexAttribPointerWithOffset(uint32(i), int32(attr_size)/4, gl.FLOAT, false, sizeof_Vertex, offset)
		offset += attr_size
	}

	if isDebugBuild() {
		// We don't need blending. This is only for Renderer.DebugDrawFontAtlas
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		rglCheckError("enable blending")
	}

	// Create framebuffer object
	// We dont need to bind framebuffer because we need it only when clearing texture
	gl.GenFramebuffers(1, &RGL.fbo)
	rglCheckError("gen framebuffer")

	logMessage(LEVEL_TRACE, TYPE_RENDERER, "Opengl Version:", gl.GoStr(gl.GetString(gl.VERSION)))
	logMessage(LEVEL_DEBUG, TYPE_RENDERER, "Vendor:", gl.GoStr(gl.GetString(gl.VENDOR)))
	logMessage(LEVEL_DEBUG, TYPE_RENDERER, "Renderer:", gl.GoStr(gl.GetString(gl.RENDERER)))
	logMessage(LEVEL_DEBUG, TYPE_RENDERER, "GLSL:", gl.GoStr(gl.GetString(gl.SHADING_LANGUAGE_VERSION)))
}

func rglGetUniformLocation(name string) int32 {
	uniform_name := gl.Str(name + "\x00")
	loc := gl.GetUniformLocation(RGL.shader_program, uniform_name)
	if loc < 0 {
		logMessage(LEVEL_FATAL, TYPE_RENDERER, "Failed to find uniform", name)
	}
	return loc
}

func rglCreateViewport(w, h int) {
	gl.Viewport(0, 0, int32(w), int32(h))
	projection := ortho(0, 0, float32(w), float32(h), -1, 1)
	gl.UniformMatrix4fv(rglGetUniformLocation("projection"), 1, true, &projection[0])
	rglCheckError("create viewport")
}

func rglSetUndercurlRect(val F32Rect) {
	loc := rglGetUniformLocation("undercurlRect")
	gl.Uniform4f(loc, val.X, val.Y, val.W, val.H)
}

func rglClearScreen(color U8Color) {
	c := color.toF32()
	gl.ClearColor(c.R, c.G, c.B, singleton.options.transparency)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	rglCheckError("clear color")
}

func rglUpdateVertices(data []Vertex) {
	if RGL.vertex_buffer_len != len(data) {
		gl.BufferData(gl.ARRAY_BUFFER, len(data)*int(sizeof_Vertex), unsafe.Pointer(&data[0]), gl.STATIC_DRAW)
		rglCheckError("vertex buffer data")
		RGL.vertex_buffer_len = len(data)
	} else if len(data) > 0 {
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(data)*int(sizeof_Vertex), unsafe.Pointer(&data[0]))
		rglCheckError("vertex buffer subdata")
	}
}

func rglRender() {
	gl.DrawArrays(gl.POINTS, 0, int32(RGL.vertex_buffer_len))
	// Since we are not using doublebuffering, we don't need swapping buffers, but we need to flush.
	gl.Flush()
	rglCheckError("render")
}

func rglInitShaders() {
	vsSource, gsSource, fsSource := rglLoadDefaultShaders()

	vertShader := rglCompileShader(vsSource, gl.VERTEX_SHADER)
	geomShader := rglCompileShader(gsSource, gl.GEOMETRY_SHADER)
	fragShader := rglCompileShader(fsSource, gl.FRAGMENT_SHADER)

	RGL.shader_program = gl.CreateProgram()
	gl.AttachShader(RGL.shader_program, vertShader)
	gl.AttachShader(RGL.shader_program, geomShader)
	gl.AttachShader(RGL.shader_program, fragShader)
	gl.LinkProgram(RGL.shader_program)

	var status int32
	gl.GetProgramiv(RGL.shader_program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(RGL.shader_program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(RGL.shader_program, logLength, nil, gl.Str(log))
		logMessage(LEVEL_FATAL, TYPE_RENDERER, "Failed to link shader program:", log)
	}

	gl.DeleteShader(vertShader)
	gl.DeleteShader(geomShader)
	gl.DeleteShader(fragShader)

	rglCheckError("init shaders")
}

func rglLoadDefaultShaders() (string, string, string) {
	vsBegin := strings.Index(EmbeddedShaderSources, "// Vertex Shader")
	gsBegin := strings.Index(EmbeddedShaderSources, "// Geometry Shader")
	fsBegin := strings.Index(EmbeddedShaderSources, "// Fragment Shader")

	assert(vsBegin != -1 && gsBegin != -1 && fsBegin != -1,
		"Shader sources are not correctly tagged!")

	assert(vsBegin < gsBegin && gsBegin < fsBegin,
		"Shader sources are not correctly ordered!")

	vsSource := EmbeddedShaderSources[vsBegin:gsBegin]
	gsSource := EmbeddedShaderSources[gsBegin:fsBegin]
	fsSource := EmbeddedShaderSources[fsBegin:]

	assert(vsSource != "" && gsSource != "" && fsSource != "",
		"Loading default shaders failed.")

	return vsSource + "\x00", gsSource + "\x00", fsSource + "\x00"
}

func rglCompileShader(source string, shader_type uint32) uint32 {
	shader := gl.CreateShader(shader_type)
	cstr, free := gl.Strs(source)
	defer free()
	gl.ShaderSource(shader, 1, cstr, nil)
	gl.CompileShader(shader)

	var result int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &result)
	if result == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		logMessage(LEVEL_FATAL, TYPE_RENDERER, "Shader", rglGetShaderName(shader_type), "compilation failed:\n", log)
	}
	rglCheckError("compile shader")

	return shader
}

func rglGetShaderName(shader_type uint32) string {
	switch shader_type {
	case gl.VERTEX_SHADER:
		return "VERTEX SHADER"
	case gl.GEOMETRY_SHADER:
		return "GEOMETRY SHADER"
	case gl.FRAGMENT_SHADER:
		return "FRAGMENT SHADER"
	}
	panic("unknown shader name")
}

// If any opengl error happens, prints error to stdout and returns false
func rglCheckError(callerName string) bool {
	error_code := gl.GetError()
	if error_code == gl.NO_ERROR {
		return true
	}

	var error_name string
	switch error_code {
	case gl.INVALID_ENUM:
		error_name = "INVALID_ENUM"
	case gl.INVALID_VALUE:
		error_name = "INVALID_VALUE"
	case gl.INVALID_OPERATION:
		error_name = "INVALID_OPERATION"
	case gl.STACK_OVERFLOW:
		error_name = "STACK_OVERFLOW"
	case gl.STACK_UNDERFLOW:
		error_name = "STACK_UNDERFLOW"
	case gl.OUT_OF_MEMORY:
		error_name = "OUT_OF_MEMORY"
	case gl.CONTEXT_LOST:
		error_name = "CONTEXT_LOST"
	default:
		error_name = fmt.Sprintf("#%.4x", error_code)
	}

	logMessage(LEVEL_ERROR, TYPE_RENDERER, "Opengl Error", error_name, "on", callerName)
	return false
}

func rglClose() {
	gl.DeleteFramebuffers(1, &RGL.fbo)
	gl.DeleteBuffers(1, &RGL.vbo)
	gl.DeleteVertexArrays(1, &RGL.vao)
	gl.DeleteProgram(RGL.shader_program)
	rglCheckError("cleanup")
}
