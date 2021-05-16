package main

import (
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/veandco/go-sdl2/sdl"
)

// Texture part
type Texture struct {
	id     uint32
	width  int
	height int
}

func (texture *Texture) Bind() {
	gl.BindTexture(gl.TEXTURE_2D, texture.id)
}

func (texture *Texture) UpdateFromSurface(surface *sdl.Surface) {
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, surface.W, surface.H, 0, gl.RGBA, gl.UNSIGNED_BYTE, surface.Data())
	RGL_CheckError("UpdateFromSurface")
}

func (texture *Texture) UpdatePartFromSurface(surface *sdl.Surface, dest *sdl.Rect) {
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, dest.X, dest.Y, dest.W, dest.H, gl.RGBA, gl.UNSIGNED_BYTE, surface.Data())
	RGL_CheckError("UpdatePartFromSurface")
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

// Renderer part
const VertexStructSize = 9 * 4

type Vertex struct {
	X, Y       float32 // layout 0
	TexX, TexY float32 // layout 1
	R, G, B, A float32 // layout 2
	useTexture float32 // layout 3
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

// render subsystem global variables
var rgl_context sdl.GLContext
var rgl_vao uint32
var rgl_vbo uint32
var rgl_shader_program uint32

var rgl_vertices []Vertex

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

func RGL_CreateTexture(width, height int) Texture {
	var texture_id uint32
	gl.GenTextures(1, &texture_id)
	gl.BindTexture(gl.TEXTURE_2D, texture_id)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, nil)
	texture := Texture{
		id:     texture_id,
		width:  width,
		height: height,
	}
	return texture
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

func RGL_FillRect(area sdl.Rect, color sdl.Color) {
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
		rgl_vertices = append(rgl_vertices, vertex)
	}
}

func RGL_DrawTexture(texture Texture, dest *sdl.Rect) {
	RGL_DrawSubTextureColor(texture, nil, dest, COLOR_WHITE)
}

func RGL_DrawTextureColor(texture Texture, dest *sdl.Rect, color sdl.Color) {
	RGL_DrawSubTextureColor(texture, nil, dest, color)
}

func RGL_DrawSubTextureColor(texture Texture, src *sdl.Rect, dest *sdl.Rect, color sdl.Color) {
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
		rgl_vertices = append(rgl_vertices, vertex)
	}
}

func RGL_UpdateVertexData(data []Vertex) {
	gl.BufferData(gl.ARRAY_BUFFER, len(data)*VertexStructSize, gl.Ptr(data), gl.STREAM_DRAW)
}

func RGL_Render(texture Texture) {
	log_debug_msg("Vertex count:", len(rgl_vertices))
	// Upload vertices data
	gl.BufferData(gl.ARRAY_BUFFER, len(rgl_vertices)*VertexStructSize, gl.Ptr(rgl_vertices), gl.STREAM_DRAW)
	RGL_CheckError("RGL_Render.BufferData")
	// Draw call
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(rgl_vertices)))
	RGL_CheckError("RGL_Render.DrawArrays")
	// Clear draw calls
	// TODO: Vertex data must came from renderer
	// and we have to reuse previous unchanged data
	// instead of recreating every frame
	rgl_vertices = nil
}

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
