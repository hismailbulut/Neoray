package main

import (
	_ "embed"
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
var (
	rgl_vao uint32
	// NOTE: We can use multiple vbo's for every grid and store vertex data per grid.
	rgl_vbo uint32
	// Framebuffer object used for clearing texture
	rgl_fbo uint32
	//go:embed shader.glsl
	rgl_shader_sources    string
	rgl_shader_program    uint32
	rgl_vertex_buffer_len int
)

func rglInit() {
	defer measure_execution_time()()

	logMessage(LEVEL_DEBUG, TYPE_RENDERER, "Initializing opengl.")
	// Initialize opengl
	if err := gl.InitWithProcAddrFunc(glfw.GetProcAddress); err != nil {
		logMessage(LEVEL_FATAL, TYPE_RENDERER, "Failed to initialize opengl:", err)
	}

	// Init shaders
	rglInitShaders()
	gl.UseProgram(rgl_shader_program)
	rglCheckError("use program")

	// Initialize vao
	gl.GenVertexArrays(1, &rgl_vao)
	gl.BindVertexArray(rgl_vao)
	rglCheckError("gen vao")

	// Initialize vbo
	gl.GenBuffers(1, &rgl_vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, rgl_vbo)
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
	gl.GenFramebuffers(1, &rgl_fbo)
	rglCheckError("gen framebuffer")

	logMessage(LEVEL_TRACE, TYPE_RENDERER, "Opengl Version:", gl.GoStr(gl.GetString(gl.VERSION)))
	logMessage(LEVEL_DEBUG, TYPE_RENDERER, "Vendor:", gl.GoStr(gl.GetString(gl.VENDOR)))
	logMessage(LEVEL_DEBUG, TYPE_RENDERER, "Renderer:", gl.GoStr(gl.GetString(gl.RENDERER)))
	logMessage(LEVEL_DEBUG, TYPE_RENDERER, "GLSL:", gl.GoStr(gl.GetString(gl.SHADING_LANGUAGE_VERSION)))
}

func rglGetUniformLocation(name string) int32 {
	uniform_name := gl.Str(name + "\x00")
	loc := gl.GetUniformLocation(rgl_shader_program, uniform_name)
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

func rglSetAtlasTexture(atlas *Texture) {
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, atlas.id)
	rglCheckError("set atlas texture")
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
	if rgl_vertex_buffer_len != len(data) {
		gl.BufferData(gl.ARRAY_BUFFER, len(data)*int(sizeof_Vertex), unsafe.Pointer(&data[0]), gl.STATIC_DRAW)
		rglCheckError("vertex buffer data")
		rgl_vertex_buffer_len = len(data)
	} else {
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(data)*int(sizeof_Vertex), unsafe.Pointer(&data[0]))
		rglCheckError("vertex buffer subdata")
	}
}

func rglRender() {
	gl.DrawArrays(gl.POINTS, 0, int32(rgl_vertex_buffer_len))
	rglCheckError("draw arrays")
	// Since we are not using doublebuffering, we don't need swapping buffers, but we need to flush.
	gl.Flush()
	rglCheckError("flush")
}

func rglInitShaders() {
	vsSource, gsSource, fsSource := rglLoadDefaultShaders()

	vertShader := rglCompileShader(vsSource, gl.VERTEX_SHADER)
	geomShader := rglCompileShader(gsSource, gl.GEOMETRY_SHADER)
	fragShader := rglCompileShader(fsSource, gl.FRAGMENT_SHADER)

	rgl_shader_program = gl.CreateProgram()
	gl.AttachShader(rgl_shader_program, vertShader)
	gl.AttachShader(rgl_shader_program, geomShader)
	gl.AttachShader(rgl_shader_program, fragShader)
	gl.LinkProgram(rgl_shader_program)

	var status int32
	gl.GetProgramiv(rgl_shader_program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(rgl_shader_program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(rgl_shader_program, logLength, nil, gl.Str(log))
		logMessage(LEVEL_FATAL, TYPE_RENDERER, "Failed to link shader program:", log)
	}

	gl.DeleteShader(vertShader)
	gl.DeleteShader(geomShader)
	gl.DeleteShader(fragShader)

	rglCheckError("init shaders")
}

func rglLoadDefaultShaders() (string, string, string) {
	vsBegin := strings.Index(rgl_shader_sources, "// Vertex Shader")
	gsBegin := strings.Index(rgl_shader_sources, "// Geometry Shader")
	fsBegin := strings.Index(rgl_shader_sources, "// Fragment Shader")

	assert(vsBegin != -1 && gsBegin != -1 && fsBegin != -1,
		"Shader sources are not correctly tagged!")

	assert(vsBegin < gsBegin && gsBegin < fsBegin,
		"Shader sources are not correctly ordered!")

	vsSource := rgl_shader_sources[vsBegin:gsBegin]
	gsSource := rgl_shader_sources[gsBegin:fsBegin]
	fsSource := rgl_shader_sources[fsBegin:]

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
	if err := gl.GetError(); err != gl.NO_ERROR {
		var errName string
		switch err {
		case gl.INVALID_ENUM:
			errName = "INVALID_ENUM"
		case gl.INVALID_VALUE:
			errName = "INVALID_VALUE"
		case gl.INVALID_OPERATION:
			errName = "INVALID_OPERATION"
		case gl.STACK_OVERFLOW:
			errName = "STACK_OVERFLOW"
		case gl.STACK_UNDERFLOW:
			errName = "STACK_UNDERFLOW"
		case gl.OUT_OF_MEMORY:
			errName = "OUT_OF_MEMORY"
		case gl.CONTEXT_LOST:
			errName = "CONTEXT_LOST"
		default:
			logMessage(LEVEL_ERROR, TYPE_RENDERER, "Opengl Error", err, "on", callerName)
			return false
		}
		logMessage(LEVEL_ERROR, TYPE_RENDERER, "Opengl Error", errName, "on", callerName)
		return false
	}
	return true
}

func rglClose() {
	gl.DeleteFramebuffers(1, &rgl_fbo)
	gl.DeleteBuffers(1, &rgl_vbo)
	gl.DeleteVertexArrays(1, &rgl_vao)
	gl.DeleteProgram(rgl_shader_program)
	rglCheckError("cleanup")
}
