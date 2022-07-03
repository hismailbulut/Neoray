package opengl

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/hismailbulut/neoray/src/common"
)

type Vertex struct {
	// position of this vertex
	pos common.Rectangle[float32] // layout 0
	// texture position
	tex1 common.Rectangle[float32] // layout 1
	// second texture position used for multiwidth characters
	tex2 common.Rectangle[float32] // layout 2
	// foreground color
	fg common.F32Color // layout 3
	// background color
	bg common.F32Color // layout 4
	// special color
	sp common.F32Color // layout 5
}

func (vertex Vertex) String() string {
	return fmt.Sprintf("Vertex(pos: %v, tex1: %v, tex2: %v, fg: %v, bg: %v, sp: %v)",
		vertex.pos,
		vertex.tex1,
		vertex.tex2,
		vertex.fg,
		vertex.bg,
		vertex.sp,
	)
}

const sizeof_Vertex = int32(unsafe.Sizeof(Vertex{})) // 96 bytes

type VertexBuffer struct {
	shader *ShaderProgram
	vaoid  uint32
	vboid  uint32
	size   int      // Last buffer size updated to GPU
	buffer []Vertex // Current buffer in memory
}

func (buffer *VertexBuffer) String() string {
	return fmt.Sprintf("VertexBuffer(VAO: %d, VBO: %d, Size: %d, Updated Size: %d)",
		buffer.vaoid,
		buffer.vboid,
		len(buffer.buffer),
		buffer.size,
	)
}

func (context *Context) CreateVertexBuffer(size int) *VertexBuffer {
	vertexBuffer := new(VertexBuffer)
	vertexBuffer.shader = &context.shader
	// Initialize vao
	CheckGLError(func() {
		gl.GenVertexArrays(1, &vertexBuffer.vaoid)
		gl.BindVertexArray(vertexBuffer.vaoid)
	})
	// Initialize vbo
	CheckGLError(func() {
		gl.GenBuffers(1, &vertexBuffer.vboid)
		gl.BindBuffer(gl.ARRAY_BUFFER, vertexBuffer.vboid)
	})
	// Enable attributes
	valueof_Vertex := reflect.ValueOf(Vertex{})
	offset := uintptr(0)
	for i := 0; i < valueof_Vertex.NumField(); i++ {
		attr_size := valueof_Vertex.Field(i).Type().Size()
		CheckGLError(func() {
			gl.EnableVertexAttribArray(uint32(i))
			gl.VertexAttribPointerWithOffset(uint32(i), int32(attr_size)/4, gl.FLOAT, false, sizeof_Vertex, offset)
		})
		offset += attr_size
	}
	// Resize buffer in memory
	vertexBuffer.Resize(size)
	return vertexBuffer
}

// OpenGL Specific functions

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

func (buffer *VertexBuffer) SetProjection(rect common.Rectangle[int]) {
	projection := orthoProjection(0, 0, float32(rect.W), float32(rect.H), -1, 1)
	loc := buffer.shader.UniformLocation("projection")
	gl.UniformMatrix4fv(loc, 1, true, &projection[0])
}

func (buffer *VertexBuffer) SetUndercurlRect(rect common.Rectangle[float32]) {
	loc := buffer.shader.UniformLocation("undercurlRect")
	gl.Uniform4f(loc, rect.X, rect.Y, rect.W, rect.H)
}

func (buffer *VertexBuffer) Bind() {
	CheckGLError(func() {
		gl.BindVertexArray(buffer.vaoid)
		gl.BindBuffer(gl.ARRAY_BUFFER, buffer.vboid)
	})
}

// Updates current buffer to GPU
// Caller responsible to bind buffer
func (buffer *VertexBuffer) Update() {
	if len(buffer.buffer) <= 0 {
		panic("empty vertex buffer")
	}
	if buffer.size != len(buffer.buffer) {
		CheckGLError(func() {
			gl.BufferData(gl.ARRAY_BUFFER, len(buffer.buffer)*int(sizeof_Vertex), unsafe.Pointer(&buffer.buffer[0]), gl.DYNAMIC_DRAW)
		})
		buffer.size = len(buffer.buffer)
	} else {
		CheckGLError(func() {
			gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(buffer.buffer)*int(sizeof_Vertex), unsafe.Pointer(&buffer.buffer[0]))
		})
	}
}

// Caller responsible to bind buffer
// Caller responsible to Flush
func (buffer *VertexBuffer) Render() {
	if buffer.size <= 0 {
		panic("buffer size is zero")
	}
	CheckGLError(func() {
		gl.DrawArrays(gl.POINTS, 0, int32(buffer.size))
	})

}

func (buffer *VertexBuffer) Destroy() {
	gl.DeleteVertexArrays(1, &buffer.vaoid)
	gl.DeleteBuffers(1, &buffer.vboid)
	buffer.size = 0
	buffer.buffer = nil
}

// Buffer functions

// Resize should clear the buffer
func (buffer *VertexBuffer) Resize(size int) {
	if size <= 0 {
		panic("vertex buffer size can not be 0")
	}
	if size == len(buffer.buffer) {
		return
	}
	buffer.buffer = make([]Vertex, size)
}

func (buffer *VertexBuffer) SetIndexPos(index int, pos common.Rectangle[float32]) {
	buffer.buffer[index].pos = pos
}

func (buffer *VertexBuffer) SetIndexTex1(index int, tex1 common.Rectangle[float32]) {
	buffer.buffer[index].tex1 = tex1
}

func (buffer *VertexBuffer) SetIndexTex2(index int, tex2 common.Rectangle[float32]) {
	buffer.buffer[index].tex2 = tex2
}

func (buffer *VertexBuffer) SetIndexFg(index int, fg common.F32Color) {
	buffer.buffer[index].fg = fg
}

func (buffer *VertexBuffer) SetIndexBg(index int, bg common.F32Color) {
	buffer.buffer[index].bg = bg
}

func (buffer *VertexBuffer) SetIndexSp(index int, sp common.F32Color) {
	buffer.buffer[index].sp = sp
}

func (buffer *VertexBuffer) CopyButPos(dst, src int) {
	buffer.buffer[dst].tex1 = buffer.buffer[src].tex1
	buffer.buffer[dst].tex2 = buffer.buffer[src].tex2
	buffer.buffer[dst].fg = buffer.buffer[src].fg
	buffer.buffer[dst].bg = buffer.buffer[src].bg
	buffer.buffer[dst].sp = buffer.buffer[src].sp
}

func (buffer *VertexBuffer) VertexAt(index int) Vertex {
	return buffer.buffer[index]
}
