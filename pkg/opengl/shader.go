package opengl

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/hismailbulut/Neoray/pkg/opengl/gl"
)

// Embedded shader sources
var (
	//go:embed shaders/grid.vert
	ShaderSourceGridVert string
	//go:embed shaders/grid.geom
	ShaderSourceGridGeom string
	//go:embed shaders/grid.frag
	ShaderSourceGridFrag string
)

// for reducing gl.UseProgram calls
var currentShaderProgramID uint32

type ShaderType uint32

const (
	VERTEX_SHADER   ShaderType = gl.VERTEX_SHADER
	GEOMETRY_SHADER ShaderType = gl.GEOMETRY_SHADER
	FRAGMENT_SHADER ShaderType = gl.FRAGMENT_SHADER
)

func (st ShaderType) String() string {
	switch st {
	case VERTEX_SHADER:
		return "VERTEX_SHADER"
	case GEOMETRY_SHADER:
		return "GEOMETRY_SHADER"
	case FRAGMENT_SHADER:
		return "FRAGMENT_SHADER"
	}
	panic("unknown shader type")
}

type Shader struct {
	ID   uint32
	Type ShaderType
}

func NewShaderFromSource(shader_type ShaderType, source string) *Shader {
	shader := &Shader{
		ID:   gl.CreateShader(uint32(shader_type)),
		Type: shader_type,
	}
	source_cstr, free := gl.Strs(source + "\x00")
	defer free()
	gl.ShaderSource(shader.ID, 1, source_cstr, nil)
	checkGLError()
	gl.CompileShader(shader.ID)
	checkGLError()
	var result int32
	gl.GetShaderiv(shader.ID, gl.COMPILE_STATUS, &result)
	if result == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader.ID, gl.INFO_LOG_LENGTH, &logLength)
		log := string(make([]byte, logLength))
		gl.GetShaderInfoLog(shader.ID, logLength, nil, gl.Str(log))
		log = strings.Trim(log, "\x00")
		panic(fmt.Errorf("Shader %s compilation failed: %s\n", shader_type, log))
	}
	return shader
}

func (shader *Shader) Delete() {
	gl.DeleteShader(shader.ID)
	shader.ID = 0
	shader.Type = 0
}

type ShaderProgram struct {
	ID       uint32
	uniforms map[string]int32
}

// All shaders can be safely destroyed after program creation
func NewShaderProgram(vert *Shader, geom *Shader, frag *Shader) *ShaderProgram {
	program := &ShaderProgram{
		ID:       gl.CreateProgram(),
		uniforms: make(map[string]int32),
	}
	if vert != nil {
		gl.AttachShader(program.ID, vert.ID)
		checkGLError()
	}
	if geom != nil {
		gl.AttachShader(program.ID, geom.ID)
		checkGLError()
	}
	if frag != nil {
		gl.AttachShader(program.ID, frag.ID)
		checkGLError()
	}
	gl.LinkProgram(program.ID)
	checkGLError()
	var status int32
	gl.GetProgramiv(program.ID, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program.ID, gl.INFO_LOG_LENGTH, &logLength)
		log := string(make([]byte, logLength))
		gl.GetProgramInfoLog(program.ID, logLength, nil, gl.Str(log))
		log = strings.Trim(log, "\x00")
		panic(fmt.Errorf("Failed to link shader program: %s", log))
	}
	return program
}

func (program ShaderProgram) UniformLocation(name string) int32 {
	if location, ok := program.uniforms[name]; ok {
		return location
	}
	uniform_name := gl.Str(name + "\x00")
	loc := gl.GetUniformLocation(uint32(program.ID), uniform_name)
	if loc < 0 {
		panic(fmt.Errorf("Failed to find uniform: %s", name))
	}
	program.uniforms[name] = loc
	return loc
}

func (program ShaderProgram) Use() {
	if program.ID != currentShaderProgramID {
		gl.UseProgram(program.ID)
		checkGLError()
		currentShaderProgramID = program.ID
	}
}

func (program ShaderProgram) Destroy() {
	gl.DeleteProgram(program.ID)
	program.ID = 0
	program.uniforms = nil
}
