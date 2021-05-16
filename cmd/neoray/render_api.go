package main

import (
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

// render subsystem global variables
var rapi_context sdl.GLContext
var rapi_vao uint32
var rapi_vbo uint32
var rapi_shader_program uint32

var rapi_vertices []Vertex

var rapi_texture_uniform_location int32
var rapi_projection_matrix_uniform_location int32
var rapi_projection_matrix [16]float32

type Texture struct {
	id     uint32
	width  int
	height int
}

type Vertex struct {
	X, Y       float32 // layout 0
	TexX, TexY float32 // layout 1
	R, G, B, A float32 // layout 2
	useTexture float32 // layout 3
}

const VertexStructSize = 9 * 4

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

func RAPI_Init(window *Window) {
	// Initialize opengl
	context, err := window.handle.GLCreateContext()
	if err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize render context:", err)
	}
	rapi_context = context
	if err = gl.Init(); err != nil {
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY, "Failed to initialize opengl:", err)
	}

	// Init shaders
	RAPI_InitShaders()
	gl.UseProgram(rapi_shader_program)

	uniform_name := gl.Str("textures\x00")
	rapi_texture_uniform_location =
		gl.GetUniformLocation(rapi_shader_program, uniform_name)

	uniform_name = gl.Str("projectionMatrix\x00")
	rapi_projection_matrix_uniform_location =
		gl.GetUniformLocation(rapi_shader_program, uniform_name)

	// Initialize vao and vbo
	gl.CreateVertexArrays(1, &rapi_vao)
	gl.BindVertexArray(rapi_vao)
	gl.GenBuffers(1, &rapi_vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, rapi_vbo)

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
	RAPI_CheckError("After Init")
}

func RAPI_CreateViewport(w, h int) {
	gl.Viewport(0, 0, int32(w), int32(h))
	// Generate orthographic projection matrix
	var top float32 = 0.0
	var left float32 = 0.0
	var right float32 = float32(w)
	var bottom float32 = float32(h)
	var near float32 = -1.0
	var far float32 = 1.0
	rml, tmb, fmn := (right - left), (top - bottom), (far - near)
	rapi_projection_matrix = [16]float32{
		float32(2. / rml), 0, 0, 0, // 1
		0, float32(2. / tmb), 0, 0, // 2
		0, 0, float32(-2. / fmn), 0, // 3
		float32(-(right + left) / rml), // 4
		float32(-(top + bottom) / tmb),
		float32(-(far + near) / fmn), 1}
}

func RAPI_CreateTexture(width, height int) Texture {
	var texture_id uint32
	gl.GenTextures(1, &texture_id)
	gl.BindTexture(gl.TEXTURE_2D, texture_id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	// data is filled with 0 and image will be transparent
	// data := make([]uint8, width*height, width*height)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	texture := Texture{
		id:     texture_id,
		width:  width,
		height: height,
	}
	return texture
}

func RAPI_CreateTextureFromSurface(surface *sdl.Surface) Texture {
	texture := RAPI_CreateTexture(int(surface.W), int(surface.H))
	texture.UpdateFromSurface(surface)
	return texture
}

func (texture *Texture) UpdateFromSurface(surface *sdl.Surface) {
	gl.BindTexture(gl.TEXTURE_2D, texture.id)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, surface.W, surface.H, 0, gl.RGBA, gl.UNSIGNED_BYTE, surface.Data())
	RAPI_CheckError("TexImage2D")
}

func (texture *Texture) UpdatePartFromSurface(surface *sdl.Surface, dest *sdl.Rect) {
	gl.BindTexture(gl.TEXTURE_2D, texture.id)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, dest.X, dest.Y, dest.W, dest.H, gl.RGBA, gl.UNSIGNED_BYTE, surface.Data())
	RAPI_CheckError("TexSubImage2D")
}

func (texture *Texture) GetInternalArea(rect *sdl.Rect) sdl.FRect {
	area := sdl.FRect{}
	area.X = float32(rect.X) / float32(texture.width)
	area.Y = float32(rect.Y) / float32(texture.height)
	area.W = float32(rect.W) / float32(texture.width)
	area.H = float32(rect.H) / float32(texture.height)
	return area
}

func (texture *Texture) Delete() {
	gl.DeleteTextures(1, &texture.id)
}

func RAPI_ClearScreen(color sdl.Color) {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	c := u8color_to_fcolor(color)
	gl.ClearColor(c.R, c.G, c.B, c.A)
}

func RAPI_FillRect(area sdl.Rect, color sdl.Color) {
	c := u8color_to_fcolor(color)
	vertex := Vertex{R: c.R, G: c.G, B: c.B, A: c.A, useTexture: 0}
	positions := [6]i32vec2{
		// TODO: Use Element Buffer Objects
		{area.X, area.Y},                   //0
		{area.X, area.Y + area.H},          //1
		{area.X + area.W, area.Y + area.H}, //2
		{area.X + area.W, area.Y + area.H}, //2
		{area.X + area.W, area.Y},          //3
		{area.X, area.Y},                   //0
	}
	for _, pos := range positions {
		vertex.X = float32(pos.X)
		vertex.Y = float32(pos.Y)
		rapi_vertices = append(rapi_vertices, vertex)
	}
}

func RAPI_DrawTexture(texture Texture, dest *sdl.Rect) {
	RAPI_DrawSubTextureColor(texture, nil, dest, COLOR_WHITE)
}

func RAPI_DrawTextureColor(texture Texture, dest *sdl.Rect, color sdl.Color) {
	RAPI_DrawSubTextureColor(texture, nil, dest, color)
}

func RAPI_DrawSubTextureColor(texture Texture, src *sdl.Rect, dest *sdl.Rect, color sdl.Color) {
	var area sdl.FRect
	if src == nil {
		area = sdl.FRect{X: 0, Y: 0, W: 1, H: 1}
	} else {
		area = texture.GetInternalArea(src)
	}
	if dest == nil {
		dest = &sdl.Rect{X: 0, Y: 0, W: int32(texture.width), H: int32(texture.height)}
	}

	// Color and texture id is same for this vertices
	c := u8color_to_fcolor(color)
	//
	vertex := Vertex{R: c.R, G: c.G, B: c.B, A: c.A, useTexture: 1}

	positions := [6]i32vec2{
		// TODO: Use Element Buffer Objects
		{dest.X, dest.Y},                   //0
		{dest.X, dest.Y + dest.H},          //1
		{dest.X + dest.W, dest.Y + dest.H}, //2
		{dest.X + dest.W, dest.Y + dest.H}, //2
		{dest.X + dest.W, dest.Y},          //3
		{dest.X, dest.Y},                   //0
	}

	texture_coords := [6]f32vec2{
		// TODO: Use Element Buffer Objects
		{area.X, area.Y},                   //0
		{area.X, area.Y + area.H},          //1
		{area.X + area.W, area.Y + area.H}, //2
		{area.X + area.W, area.Y + area.H}, //2
		{area.X + area.W, area.Y},          //3
		{area.X, area.Y},                   //0
	}

	for i := 0; i < 6; i++ {
		vertex.X = float32(positions[i].X)
		vertex.Y = float32(positions[i].Y)
		vertex.TexX = texture_coords[i].X
		vertex.TexY = texture_coords[i].Y
		rapi_vertices = append(rapi_vertices, vertex)
	}
}

func RAPI_Render(texture Texture) {
	// Upload vertices data
	// We are changing data on every draw call and STREAM_DRAW is the hint for this
	gl.BufferData(gl.ARRAY_BUFFER, len(rapi_vertices)*VertexStructSize, gl.Ptr(rapi_vertices), gl.STREAM_DRAW)
	RAPI_CheckError("BufferData")
	// upload projection matrix
	gl.UniformMatrix4fv(
		rapi_projection_matrix_uniform_location,
		1, true, &rapi_projection_matrix[0])
	// bind atlas texture
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture.id)
	gl.Uniform1i(rapi_texture_uniform_location, gl.TEXTURE0)
	// Draw call
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(rapi_vertices)))
	RAPI_CheckError("DrawArrays")
	// Clear draw calls
	rapi_vertices = nil
}

func RAPI_InitShaders() {
	vertexShader := RAPI_CompileShader(vertexShaderSource, gl.VERTEX_SHADER)
	fragmentShader := RAPI_CompileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	rapi_shader_program = gl.CreateProgram()
	gl.AttachShader(rapi_shader_program, vertexShader)
	gl.AttachShader(rapi_shader_program, fragmentShader)
	gl.LinkProgram(rapi_shader_program)
	var status int32
	gl.GetProgramiv(rapi_shader_program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(rapi_shader_program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(rapi_shader_program, logLength, nil, gl.Str(log))
		log_message(LOG_LEVEL_FATAL, LOG_TYPE_NEORAY,
			"Failed to link shader program:", log)
	}
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)
}

func RAPI_CompileShader(source string, shader_type uint32) uint32 {
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

func RAPI_CheckError(callerName string) {
	if err := gl.GetError(); err != gl.NO_ERROR {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, callerName, ": Opengl Error:", err)
	}
}

func RAPI_Close() {
	gl.DeleteProgram(rapi_shader_program)
	gl.DeleteBuffers(1, &rapi_vbo)
	gl.DeleteVertexArrays(1, &rapi_vao)
	sdl.GLDeleteContext(rapi_context)
}
