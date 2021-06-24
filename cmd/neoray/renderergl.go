package main

import (
	_ "embed"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
)

const VertexStructSize = 16 * 4

type Vertex struct {
	// These are vertex positions. May not be changed for
	// most of the vertices in runtime.
	pos F32Vec2 // layout 0
	// These are vertex atlas texture positions for given character
	// and they are changing most of the time.
	tex F32Vec2 // layout 1
	// foreground color
	fg F32Color // layout 2
	// background color
	bg F32Color // layout 3
	// special color
	sp F32Color // layout 4
}

// render subsystem global variables
var (
	rgl_vao uint32
	rgl_vbo uint32
	rgl_ebo uint32

	//go:embed shader.glsl
	rgl_shader_sources string

	rgl_shader_program     uint32
	rgl_atlas_uniform      int32
	rgl_projection_uniform int32

	rgl_vertex_buffer_len  int
	rgl_element_buffer_len int
)

func rgl_init() {
	defer measure_execution_time("RGL_Init")()

	// Initialize opengl
	if err := gl.Init(); err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_RENDERER, "Failed to initialize opengl:", err)
	}

	// Init shaders
	rgl_initShaders()
	gl.UseProgram(rgl_shader_program)

	rgl_checkError("gl use program")

	rgl_atlas_uniform = rgl_getUniformLocation("atlas")
	rgl_projection_uniform = rgl_getUniformLocation("projection")

	// Initialize vao
	gl.CreateVertexArrays(1, &rgl_vao)
	gl.BindVertexArray(rgl_vao)

	// Initialize vbo
	gl.GenBuffers(1, &rgl_vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, rgl_vbo)

	// Initialize ebo
	gl.GenBuffers(1, &rgl_ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, rgl_ebo)

	rgl_checkError("gl bind buffers")

	// position
	offset := 0
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, VertexStructSize, uintptr(offset))
	// texture coords
	offset += 2 * 4
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, VertexStructSize, uintptr(offset))
	// foreground color
	offset += 2 * 4
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 4, gl.FLOAT, false, VertexStructSize, uintptr(offset))
	// background color
	offset += 4 * 4
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointerWithOffset(3, 4, gl.FLOAT, false, VertexStructSize, uintptr(offset))
	// special color
	offset += 4 * 4
	gl.EnableVertexAttribArray(4)
	gl.VertexAttribPointerWithOffset(4, 4, gl.FLOAT, false, VertexStructSize, uintptr(offset))

	rgl_checkError("gl enable attributes")
	// NOTE: If you changed something in Vertex you have to update VertexStructSize!

	if isDebugBuild() {
		// We don't need blending. This is only for Renderer.DebugDrawFontAtlas
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		rgl_checkError("gl enable blending")
	}

	log_message(LOG_LEVEL_TRACE, LOG_TYPE_RENDERER, "Opengl Version:", gl.GoStr(gl.GetString(gl.VERSION)))
}

func rgl_getUniformLocation(name string) int32 {
	uniform_name := gl.Str(name + "\x00")
	loc := gl.GetUniformLocation(rgl_shader_program, uniform_name)
	if loc < 0 {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_RENDERER, "Failed to find uniform", name)
	}
	return loc
}

func rgl_createViewport(w, h int) {
	gl.Viewport(0, 0, int32(w), int32(h))
	projection := orthoProjection(0, 0, float32(w), float32(h), -1, 1)
	gl.UniformMatrix4fv(rgl_projection_uniform, 1, true, &projection[0])
}

func rgl_setAtlasTexture(atlas *Texture) {
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, atlas.id)
}

func rgl_setUndercurlRect(val F32Rect) {
	loc := rgl_getUniformLocation("undercurlRect")
	gl.Uniform4f(loc, val.X, val.Y, val.W, val.H)
}

func rgl_clearScreen(color U8Color) {
	gl.Clear(gl.COLOR_BUFFER_BIT)
	c := color.ToF32Color()
	gl.ClearColor(c.R, c.G, c.B, EditorSingleton.framebufferTransparency)
}

func rgl_updateVertices(data []Vertex) {
	if rgl_vertex_buffer_len != len(data) {
		gl.BufferData(gl.ARRAY_BUFFER, len(data)*VertexStructSize, gl.Ptr(data), gl.STATIC_DRAW)
		rgl_checkError("RGL_UpdateVertices.BufferData")
		rgl_vertex_buffer_len = len(data)
	} else {
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(data)*VertexStructSize, gl.Ptr(data))
		rgl_checkError("RGL_UpdateVertices.BufferSubData")
	}
}

func rgl_updateIndices(data []uint32) {
	if rgl_element_buffer_len != len(data) {
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(data)*4, gl.Ptr(data), gl.STATIC_DRAW)
		rgl_checkError("RGL_UpdateIndices.BufferData")
		rgl_element_buffer_len = len(data)
	} else {
		gl.BufferSubData(gl.ELEMENT_ARRAY_BUFFER, 0, len(data)*4, gl.Ptr(data))
		rgl_checkError("RGL_UpdateIndices.BufferSubData")
	}
}

func rgl_render() {
	gl.DrawElements(gl.TRIANGLES, int32(rgl_element_buffer_len), gl.UNSIGNED_INT, nil)
	rgl_checkError("RGL_Render.DrawElements")
}

func rgl_initShaders() {
	vertexShaderSource, fragmentShaderSource := rgl_loadShaders()
	vertexShader := rgl_compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	fragmentShader := rgl_compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)

	rgl_shader_program = gl.CreateProgram()
	gl.AttachShader(rgl_shader_program, vertexShader)
	gl.AttachShader(rgl_shader_program, fragmentShader)
	gl.LinkProgram(rgl_shader_program)

	var status int32
	gl.GetProgramiv(rgl_shader_program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(rgl_shader_program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(rgl_shader_program, logLength, nil, gl.Str(log))
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_RENDERER,
			"Failed to link shader program:", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
}

func rgl_loadShaders() (string, string) {
	vertexSourceBegin := strings.Index(rgl_shader_sources, "// Vertex Shader")
	fragSourceBegin := strings.Index(rgl_shader_sources, "// Fragment Shader")
	assert(vertexSourceBegin != -1 && fragSourceBegin != -1, "Shaders are not correctly prefixed!")

	var vertexShaderSource string
	var fragmentShaderSource string
	if vertexSourceBegin < fragSourceBegin {
		vertexShaderSource = rgl_shader_sources[vertexSourceBegin:fragSourceBegin]
		fragmentShaderSource = rgl_shader_sources[fragSourceBegin:]
	} else {
		fragmentShaderSource = rgl_shader_sources[fragSourceBegin:vertexSourceBegin]
		vertexShaderSource = rgl_shader_sources[vertexSourceBegin:]
	}

	return vertexShaderSource + "\x00", fragmentShaderSource + "\x00"
}

func rgl_compileShader(source string, shader_type uint32) uint32 {
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
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_RENDERER,
			"Shader compilation failed:", source, log)
	}

	return shader
}

func rgl_checkError(callerName string) {
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
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_RENDERER, "Opengl Error", err, "on", callerName)
			return
		}
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_RENDERER, "Opengl Error", errName, "on", callerName)
	}
}

func rgl_close() {
	gl.DeleteProgram(rgl_shader_program)
	gl.DeleteBuffers(1, &rgl_vbo)
	gl.DeleteVertexArrays(1, &rgl_vao)
}
