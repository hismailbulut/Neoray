package opengl

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/hismailbulut/Neoray/pkg/bench"
	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/logger"
	"github.com/hismailbulut/Neoray/pkg/opengl/gl"
)

var boundVertexArrayId uint32 // vao id

type Vertex struct {
	// position of this vertex
	Pos common.Rectangle[float32] // layout 0
	// texture position
	Tex1 common.Rectangle[float32] // layout 1
	// second texture position used for multiwidth characters
	Tex2 common.Rectangle[float32] // layout 2
	// foreground color
	Fg common.Color // layout 3
	// background color
	Bg common.Color // layout 4
	// special color
	Sp common.Color // layout 5
}

func (vertex Vertex) String() string {
	return fmt.Sprintf("Vertex(pos: %v, tex1: %v, tex2: %v, fg: %v, bg: %v, sp: %v)",
		vertex.Pos,
		vertex.Tex1,
		vertex.Tex2,
		vertex.Fg,
		vertex.Bg,
		vertex.Sp,
	)
}

const sizeof_Vertex = int32(unsafe.Sizeof(Vertex{})) // 96 bytes

type VertexBuffer struct {
	shader      *ShaderProgram
	vaoid       uint32
	vboid       uint32
	updatedSize int      // Last buffer size updated to GPU
	data        []Vertex // Current buffer in memory (len(data) gives capacity)
}

func (buffer *VertexBuffer) String() string {
	return fmt.Sprintf("VertexBuffer(VAO: %d, VBO: %d, Size: %d, Updated Size: %d)",
		buffer.vaoid,
		buffer.vboid,
		len(buffer.data),
		buffer.updatedSize,
	)
}

func (context *Context) CreateVertexBuffer(size int) *VertexBuffer {
	if size <= 0 {
		panic("vertex buffer size must bigger then zero")
	}
	buffer := new(VertexBuffer)
	buffer.shader = &context.shader
	// Initialize vao
	gl.GenVertexArrays(1, &buffer.vaoid)
	checkGLError()
	gl.BindVertexArray(buffer.vaoid)
	checkGLError()
	boundVertexArrayId = buffer.vaoid
	// Initialize vbo
	gl.GenBuffers(1, &buffer.vboid)
	checkGLError()
	gl.BindBuffer(gl.ARRAY_BUFFER, buffer.vboid)
	checkGLError()
	// Enable attributes
	valueof_Vertex := reflect.ValueOf(Vertex{})
	offset := uintptr(0)
	for i := 0; i < valueof_Vertex.NumField(); i++ {
		attr_size := valueof_Vertex.Field(i).Type().Size()
		gl.EnableVertexAttribArray(uint32(i))
		checkGLError()
		gl.VertexAttribPointerWithOffset(uint32(i), int32(attr_size)/4, gl.FLOAT, false, sizeof_Vertex, offset)
		checkGLError()
		offset += attr_size
	}
	// Create buffer in memory
	buffer.data = make([]Vertex, size)
	logger.Log(logger.DEBUG, "Buffer created:", buffer)
	return buffer
}

// Resize should clear the buffer
func (buffer *VertexBuffer) Resize(size int) {
	if size <= 0 {
		panic("vertex buffer size must bigger then zero")
	}
	if size == len(buffer.data) {
		return
	}
	EndBenchmark := bench.BeginBenchmark()
	defer EndBenchmark("VertexBuffer.Resize")
	// Clear current buffer
	zVertex := Vertex{}
	for i := range buffer.data {
		buffer.data[i] = zVertex
	}
	// Resize
	if cap(buffer.data) > size {
		buffer.data = buffer.data[:size]
	} else {
		remaining := size - len(buffer.data)
		buffer.data = append(buffer.data, make([]Vertex, remaining)...)
	}
}

// OpenGL Specific functions

func (buffer *VertexBuffer) Bind() {
	if boundVertexArrayId == buffer.vaoid {
		return
	}
	gl.BindVertexArray(buffer.vaoid)
	checkGLError()
	gl.BindBuffer(gl.ARRAY_BUFFER, buffer.vboid)
	checkGLError()
	boundVertexArrayId = buffer.vaoid
}

// Updates current buffer to GPU
// Caller responsible to bind buffer
func (buffer *VertexBuffer) Update() {
	if boundVertexArrayId != buffer.vaoid {
		panic("vertex buffer must be bound before update")
	}
	if len(buffer.data) <= 0 {
		panic("empty vertex buffer")
	}
	if buffer.updatedSize != len(buffer.data) {
		gl.BufferData(gl.ARRAY_BUFFER, len(buffer.data)*int(sizeof_Vertex), unsafe.Pointer(&buffer.data[0]), gl.DYNAMIC_DRAW)
		checkGLError()
		buffer.updatedSize = len(buffer.data)
	} else {
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(buffer.data)*int(sizeof_Vertex), unsafe.Pointer(&buffer.data[0]))
		checkGLError()
	}
}

// Caller responsible to bind buffer
// Caller responsible to Flush
func (buffer *VertexBuffer) Render() {
	if boundVertexArrayId != buffer.vaoid {
		panic("vertex buffer must be bound before render")
	}
	if buffer.updatedSize <= 0 {
		panic("buffer size is zero")
	}
	gl.DrawArrays(gl.POINTS, 0, int32(buffer.updatedSize))
	checkGLError()
}

func orthoProjection(top, left, right, bottom, near, far float32) [16]float32 {
	rml, tmb, fmn := (right - left), (top - bottom), (far - near)
	matrix := [16]float32{}
	matrix[0] = 2 / rml
	matrix[5] = 2 / tmb
	matrix[10] = -2 / fmn
	matrix[12] = -(right + left) / rml
	matrix[13] = -(top + bottom) / tmb
	matrix[14] = -(far + near) / fmn
	matrix[15] = 1
	return matrix
}

func (buffer *VertexBuffer) SetProjection(rect common.Rectangle[float32]) {
	projection := orthoProjection(rect.X, rect.Y, rect.W, rect.H, -1, 1)
	loc := buffer.shader.UniformLocation("projection")
	gl.UniformMatrix4fv(loc, 1, true, &projection[0])
}

func (buffer *VertexBuffer) SetUndercurlRect(rect common.Rectangle[float32]) {
	loc := buffer.shader.UniformLocation("undercurlRect")
	gl.Uniform4f(loc, rect.X, rect.Y, rect.W, rect.H)
}

func (buffer *VertexBuffer) Destroy() {
	buffer.shader = nil
	gl.DeleteVertexArrays(1, &buffer.vaoid)
	gl.DeleteBuffers(1, &buffer.vboid)
	buffer.updatedSize = 0
	buffer.data = nil
	logger.Log(logger.DEBUG, "Buffer destroyed:", buffer)
}

// Buffer functions

func (buffer *VertexBuffer) SetIndexPos(index int, pos common.Rectangle[float32]) {
	buffer.data[index].Pos = pos
}

func (buffer *VertexBuffer) SetIndexTex1(index int, tex1 common.Rectangle[float32]) {
	buffer.data[index].Tex1 = tex1
}

func (buffer *VertexBuffer) SetIndexTex2(index int, tex2 common.Rectangle[float32]) {
	buffer.data[index].Tex2 = tex2
}

func (buffer *VertexBuffer) SetIndexFg(index int, fg common.Color) {
	buffer.data[index].Fg = fg
}

func (buffer *VertexBuffer) SetIndexBg(index int, bg common.Color) {
	buffer.data[index].Bg = bg
}

func (buffer *VertexBuffer) SetIndexSp(index int, sp common.Color) {
	buffer.data[index].Sp = sp
}

func (buffer *VertexBuffer) CopyButPos(dst, src int) {
	buffer.data[dst].Tex1 = buffer.data[src].Tex1
	buffer.data[dst].Tex2 = buffer.data[src].Tex2
	buffer.data[dst].Fg = buffer.data[src].Fg
	buffer.data[dst].Bg = buffer.data[src].Bg
	buffer.data[dst].Sp = buffer.data[src].Sp
}

func (buffer *VertexBuffer) VertexAt(index int) Vertex {
	return buffer.data[index]
}
