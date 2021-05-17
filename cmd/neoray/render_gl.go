package main

import (
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

const VertexStructSize = 9 * 4

type Vertex struct {
	X, Y       float32 // layout 0
	TexX, TexY float32 // layout 1
	R, G, B, A float32 // layout 2
	useTexture float32 // layout 3
}

// render subsystem global variables
var rgl_context sdl.GLContext
var rgl_vao uint32
var rgl_vbo uint32
var rgl_shader_program uint32

var rgl_atlas_uniform int32
var rgl_projection_uniform int32

func RGL_Init(window *Window) {
	// Initialize opengl
	context, err := window.handle.GLCreateContext()
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize render context:", err)
	}
	rgl_context = context
	if err = gl.Init(); err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize opengl:", err)
	}

	// Init shaders
	RGL_InitShaders()
	gl.UseProgram(rgl_shader_program)

	uniform_name := gl.Str("textures\x00")
	rgl_atlas_uniform =
		gl.GetUniformLocation(rgl_shader_program, uniform_name)

	uniform_name = gl.Str("projectionMatrix\x00")
	rgl_projection_uniform =
		gl.GetUniformLocation(rgl_shader_program, uniform_name)

	// Initialize vao and vbo
	gl.CreateVertexArrays(1, &rgl_vao)
	gl.BindVertexArray(rgl_vao)
	gl.GenBuffers(1, &rgl_vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, rgl_vbo)

	// position
	offset := 0
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointerWithOffset(0, 2, gl.FLOAT, false, VertexStructSize, uintptr(offset))
	// texture coords
	offset += 2 * 4
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, VertexStructSize, uintptr(offset))
	// color
	offset += 2 * 4
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointerWithOffset(2, 4, gl.FLOAT, false, VertexStructSize, uintptr(offset))
	// useTexture boolean value
	offset += 4 * 4
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointerWithOffset(3, 1, gl.FLOAT, false, VertexStructSize, uintptr(offset))

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.TEXTURE_2D)

	RGL_CheckError("Init")
}

func RGL_CreateViewport(w, h int) {
	gl.Viewport(0, 0, int32(w), int32(h))
	// Generate orthographic projection matrix
	var top float32 = 0.0
	var left float32 = 0.0
	var right float32 = float32(w)
	var bottom float32 = float32(h)
	var near float32 = -1.0
	var far float32 = 1.0
	rml, tmb, fmn := (right - left), (top - bottom), (far - near)
	projection_matrix := [16]float32{
		float32(2. / rml), 0, 0, 0, // 1
		0, float32(2. / tmb), 0, 0, // 2
		0, 0, float32(-2. / fmn), 0, // 3
		float32(-(right + left) / rml), // 4
		float32(-(top + bottom) / tmb),
		float32(-(far + near) / fmn), 1}
	// upload projection matrix
	gl.UniformMatrix4fv(
		rgl_projection_uniform,
		1, true, &projection_matrix[0])
}

func RGL_SetAtlasTexture(atlas *Texture) {
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, atlas.id)
	gl.Uniform1i(rgl_atlas_uniform, gl.TEXTURE0)
}

func RGL_ClearScreen(color sdl.Color) {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	c := u8color_to_fcolor(color)
	gl.ClearColor(c.R, c.G, c.B, c.A)
}

func RGL_Render(atlas Texture, vertex_data []Vertex) {
	// Upload vertex data
	gl.BufferData(gl.ARRAY_BUFFER, len(vertex_data)*VertexStructSize, gl.Ptr(vertex_data), gl.STREAM_DRAW)
	RGL_CheckError("RGL_Render.BufferData")
	// Draw
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(vertex_data)))
	RGL_CheckError("RGL_Render.DrawArrays")
}

var vertexShaderSource = `
#version 330 core

layout(location = 0) in vec2 pos;
layout(location = 1) in vec2 texCoord;
layout(location = 2) in vec4 color;
layout(location = 3) in float useTex;

out vec2 textureCoord;
out vec4 vertexColor;
out float useTexture;

uniform mat4 projectionMatrix;

void main() {
	gl_Position = vec4(pos, 0, 1) * projectionMatrix;
	textureCoord = texCoord;
	useTexture = useTex;
	vertexColor = color;
}
` + "\x00"

var fragmentShaderSource = `
#version 330 core

in vec2 textureCoord;
in vec4 vertexColor;
in float useTexture;

uniform sampler2D atlas;

void main() {
	vec4 color;
	if (useTexture > 0.5) {
		color = texture(atlas, textureCoord);
		color *= vertexColor;
	} else {
		color = vertexColor;
	}
	gl_FragColor = color;
}
` + "\x00"

func RGL_InitShaders() {
	vertexShader := RGL_CompileShader(vertexShaderSource, gl.VERTEX_SHADER)
	fragmentShader := RGL_CompileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
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
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY,
			"Failed to link shader program:", log)
	}
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
}

func RGL_CompileShader(source string, shader_type uint32) uint32 {
	shader := gl.CreateShader(shader_type)
	cstr, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, cstr, nil)
	free()
	gl.CompileShader(shader)
	var result int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &result)
	if result == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY,
			"Shader compilation failed:", source, log)
	}
	return shader
}

func RGL_CheckError(callerName string) {
	if err := gl.GetError(); err != gl.NO_ERROR {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, callerName, ": Opengl Error:", err)
	}
}

func RGL_Close() {
	gl.DeleteProgram(rgl_shader_program)
	gl.DeleteBuffers(1, &rgl_vbo)
	gl.DeleteVertexArrays(1, &rgl_vao)
	sdl.GLDeleteContext(rgl_context)
}
