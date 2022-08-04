package opengl

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/hismailbulut/Neoray/pkg/opengl/gl"
)

//go:embed shader.glsl
var EmbeddedShaderSources string

type ShaderProgram uint32

func (program ShaderProgram) UniformLocation(name string) int32 {
	uniform_name := gl.Str(name + "\x00")
	loc := gl.GetUniformLocation(uint32(program), uniform_name)
	if loc < 0 {
		panic(fmt.Errorf("Failed to find uniform: %s", name))
	}
	return loc
}

func (program ShaderProgram) Use() {
	gl.UseProgram(uint32(program))
	checkGLError()
}

func (program ShaderProgram) Destroy() {
	gl.DeleteProgram(uint32(program))
}

func DefaultProgram() ShaderProgram {
	vsSource, gsSource, fsSource := loadDefaultShaders()
	vertShader := compileShader(vsSource, gl.VERTEX_SHADER)
	geomShader := compileShader(gsSource, gl.GEOMETRY_SHADER)
	fragShader := compileShader(fsSource, gl.FRAGMENT_SHADER)

	shader_program := gl.CreateProgram()
	gl.AttachShader(shader_program, vertShader)
	checkGLError()
	gl.AttachShader(shader_program, geomShader)
	checkGLError()
	gl.AttachShader(shader_program, fragShader)
	checkGLError()
	gl.LinkProgram(shader_program)
	checkGLError()

	var status int32
	gl.GetProgramiv(shader_program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(shader_program, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(shader_program, logLength, nil, gl.Str(log))
		panic(fmt.Errorf("Failed to link shader program: %s", log))
	}

	gl.DeleteShader(vertShader)
	gl.DeleteShader(geomShader)
	gl.DeleteShader(fragShader)

	return ShaderProgram(shader_program)
}

func loadDefaultShaders() (string, string, string) {
	vsBegin := strings.Index(EmbeddedShaderSources, "// Vertex Shader")
	gsBegin := strings.Index(EmbeddedShaderSources, "// Geometry Shader")
	fsBegin := strings.Index(EmbeddedShaderSources, "// Fragment Shader")

	if vsBegin == -1 || gsBegin == -1 || fsBegin == -1 {
		panic("Shader sources are not correctly tagged!")
	}

	if vsBegin >= gsBegin || gsBegin >= fsBegin {
		panic("Shader sources are not correctly ordered!")
	}

	vsSource := EmbeddedShaderSources[vsBegin:gsBegin]
	gsSource := EmbeddedShaderSources[gsBegin:fsBegin]
	fsSource := EmbeddedShaderSources[fsBegin:]

	return vsSource + "\x00", gsSource + "\x00", fsSource + "\x00"
}

func compileShader(source string, shader_type uint32) uint32 {
	shader := gl.CreateShader(shader_type)
	cstr, free := gl.Strs(source)
	defer free()
	gl.ShaderSource(shader, 1, cstr, nil)
	checkGLError()
	gl.CompileShader(shader)
	checkGLError()
	var result int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &result)
	if result == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		panic(fmt.Errorf("Shader %s compilation failed: %s\n", shaderName(shader_type), log))
	}
	return shader
}

func shaderName(shader_type uint32) string {
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
