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

// draw call containers
var rapi_vertices []Vertex
var rapi_textures map[uint32]int // opengl texture id to our texture id
var rapi_textures_uniform_location int32
var rapi_texture_count_uniform_location int32
var rapi_projection_matrix_uniform_location int32

var projection_matrix [16]float32

const MAX_TEXTURE_COUNT = 8

type Texture struct {
	id     uint32
	width  int
	height int
}

type Vertex struct {
	X, Y       int32   // layout 0
	TexX, TexY float32 // layout 1
	TexId      int32   // layout 2
	R, G, B, A float32 // layout 3
}

var vertexShaderSource = `
#version 330 core

layout(location = 0) in vec2 pos;
layout(location = 1) in vec2 texCoord;
layout(location = 2) in int texId;
layout(location = 3) in vec4 color;

out vec2 vertexTextureCoord;
out int vertexTextureId;
out vec4 vertexColor;

uniform mat4 projectionMatrix;

void main() {
	gl_Position = vec4(pos, 0, 1) * projectionMatrix;
	vertexTextureCoord = texCoord;
	vertexTextureId = texId;
	vertexColor = color;
}
` + "\x00"

var fragmentShaderSource = `
#version 330 core

in vec2 vertexTextureCoord;
flat in int vertexTextureId;
in vec4 vertexColor;

// max 8 textures at the same time
uniform sampler2D textures[8];
uniform int textureCount;

void main() {
	vec4 color;
	if (textureCount > 0 &&
		(vertexTextureId >= 0 &&
		vertexTextureId < textureCount)) {

		color = texture(textures[vertexTextureId], vertexTextureCoord);
	} else {
		color = vertexColor;
	}
	gl_FragColor = color;
}
` + "\x00"

func RAPI_CheckError() {
	if err := gl.GetError(); err != gl.NO_ERROR {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Opengl Error:", err)
	}
}

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

	// TODO: Use slice
	rapi_textures = make(map[uint32]int)

	// Initialize vao and vbo
	gl.CreateVertexArrays(1, &rapi_vao)
	gl.BindVertexArray(rapi_vao)
	gl.GenBuffers(1, &rapi_vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, rapi_vbo)

	gl.EnableVertexAttribArray(0)
	gl.EnableVertexAttribArray(1)
	gl.EnableVertexAttribArray(2)
	gl.EnableVertexAttribArray(3)

	offset := 0
	// position
	gl.VertexAttribPointerWithOffset(0, 2, gl.INT, false, 2*4, uintptr(offset))
	offset += 2 * 4
	// texture coords
	gl.VertexAttribPointerWithOffset(1, 2, gl.FLOAT, false, 2*4, uintptr(offset))
	offset += 2 * 4
	// texture id
	gl.VertexAttribPointerWithOffset(2, 1, gl.INT, false, 1*4, uintptr(offset))
	offset += 1 * 4
	// color
	gl.VertexAttribPointerWithOffset(3, 4, gl.FLOAT, false, 4*4, uintptr(offset))

	// Init shaders
	RAPI_InitShaders()
	gl.UseProgram(rapi_shader_program)
	uniform_name := gl.Str("textures[8]\x00")
	rapi_textures_uniform_location =
		gl.GetUniformLocation(rapi_shader_program, uniform_name)
	uniform_name = gl.Str("textureCount\x00")
	rapi_texture_count_uniform_location =
		gl.GetUniformLocation(rapi_shader_program, uniform_name)
	uniform_name = gl.Str("projectionMatrix\x00")
	rapi_projection_matrix_uniform_location =
		gl.GetUniformLocation(rapi_shader_program, uniform_name)

	// gl.Enable(gl.BLEND)
	// gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	// gl.Enable(gl.DEBUG_OUTPUT)
	gl.Enable(gl.TEXTURE_2D)
	RAPI_CheckError()
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
	RAPI_CheckError()
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
	RAPI_CheckError()
	return shader
}

func RAPI_CreateViewport(w, h int) {
	// gl.Viewport(0, 0, int32(w), int32(h))
	// Generate orthographic projection matrix
	var right float32 = 0.0
	var left float32 = float32(w)
	var top float32 = 0.0
	var bottom float32 = float32(h)
	var far float32 = 1.0
	var near float32 = 1.0
	rml, tmb, fmn := (right - left), (top - bottom), (far - near)
	projection_matrix = [16]float32{
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
	RAPI_CheckError()
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
	RAPI_CheckError()
}

func (texture *Texture) UpdatePartFromSurface(surface *sdl.Surface, dest *sdl.Rect) {
	gl.BindTexture(gl.TEXTURE_2D, texture.id)
	gl.TexSubImage2D(gl.TEXTURE_2D, 0, dest.X, dest.Y, dest.W, dest.H, gl.RGBA, gl.UNSIGNED_BYTE, surface.Data())
	RAPI_CheckError()
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
	// NOTE: if textures will be deleted at runtime,
	// we need to rearrange list for indexes
	// NOTE: use simple slice instead of map
	if _, ok := rapi_textures[texture.id]; ok == true {
		delete(rapi_textures, texture.id)
	}
}

func RAPI_ClearScreen(color sdl.Color) {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	c := u8color_to_fcolor(color)
	gl.ClearColor(c.R, c.G, c.B, c.A)
}

func RAPI_FillRect(area sdl.Rect, color sdl.Color) {
	c := u8color_to_fcolor(color)
	vertex := Vertex{TexId: -1, R: c.R, G: c.G, B: c.B, A: c.A}
	positions := [6]i32vec2{
		// TODO: Use Element Buffer Objects
		{area.X, area.Y + area.H},          //1
		{area.X + area.W, area.Y + area.H}, //2
		{area.X + area.W, area.Y},          //4
		{area.X + area.W, area.Y},          //4
		{area.X, area.Y},                   //3
		{area.X, area.Y + area.H},          //1
	}
	for _, pos := range positions {
		vertex.X = pos.X
		vertex.Y = pos.Y
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

	var texture_id int
	// Find texture id for this text
	if id, ok := rapi_textures[texture.id]; ok == true {
		texture_id = id
	} else { // add this texture to our list
		texture_id = len(rapi_textures)
		rapi_textures[texture.id] = texture_id
	}

	// Color and texture id is same for this vertices
	c := u8color_to_fcolor(color)
	vertex := Vertex{TexId: int32(texture_id), R: c.R, G: c.G, B: c.B, A: c.A}

	positions := [6]i32vec2{
		// TODO: Use Element Buffer Objects
		{dest.X, dest.Y + dest.H},          //1
		{dest.X + dest.W, dest.Y + dest.H}, //2
		{dest.X + dest.W, dest.Y},          //4
		{dest.X + dest.W, dest.Y},          //4
		{dest.X, dest.Y},                   //3
		{dest.X, dest.Y + dest.H},          //1
	}

	texture_coords := [6]f32vec2{
		// TODO: Use Element Buffer Objects
		{area.X, area.Y + area.H},          //1
		{area.X + area.W, area.Y + area.H}, //2
		{area.X + area.W, area.Y},          //4
		{area.X + area.W, area.Y},          //4
		{area.X, area.Y},                   //3
		{area.X, area.Y + area.H},          //1
	}

	for i := 0; i < 6; i++ {
		vertex.X = positions[i].X
		vertex.Y = positions[i].Y
		vertex.TexX = texture_coords[i].X
		vertex.TexY = texture_coords[i].Y
		rapi_vertices = append(rapi_vertices, vertex)
	}
}

func RAPI_Render() {
	// Upload vertices data
	vertex_size := 4 * 9 // every variable is 4 bytes and there are 9 variables in the vertex struct
	// We are changing data on every draw call and STREAM_DRAW is the hint for this
	log_debug_msg("Len Vertices:", len(rapi_vertices))
	gl.BufferData(gl.ARRAY_BUFFER, len(rapi_vertices)*vertex_size, gl.Ptr(rapi_vertices), gl.STREAM_DRAW)
	RAPI_CheckError()
	log_debug_msg("Passed glBufferData")
	// upload projection matrix
	gl.UniformMatrix4fv(
		rapi_projection_matrix_uniform_location,
		1, false, &projection_matrix[0])
	// set texture count uniform
	log_debug_msg("Texture Count:", len(rapi_textures))
	gl.Uniform1i(
		rapi_texture_count_uniform_location, int32(len(rapi_textures)))
	// Set Textures
	var i int
	for texture_id, index := range rapi_textures {
		if i != index {
			// This is for debug
			log_debug_msg("Index problem on textures.")
		}
		if index < MAX_TEXTURE_COUNT {
			active_texture := gl.TEXTURE0 + uint32(index)
			gl.ActiveTexture(active_texture)
			gl.BindTexture(gl.TEXTURE_2D, texture_id)
			gl.Uniform1i(rapi_textures_uniform_location, int32(active_texture))
			log_debug_msg("Texture Id:", texture_id, "Index:", index)
		} else {
			// NOTE: May draw more than one times
			// but i dont think this will happen
			log_debug_msg("Not enough texture count.")
		}
		i++
	}
	// Draw call
	log_debug_msg("glDrawArrays begin")
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(rapi_vertices)))
	RAPI_CheckError()
	log_debug_msg("Passed glDrawArrays")
	// Clear draw calls
	rapi_vertices = nil
}

func RAPI_Close() {
	gl.DeleteProgram(rapi_shader_program)
	gl.DeleteBuffers(1, &rapi_vbo)
	gl.DeleteVertexArrays(1, &rapi_vao)
	sdl.GLDeleteContext(rapi_context)
}
