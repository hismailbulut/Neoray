package window

import (
	"errors"
	"fmt"
	"image"

	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/hismailbulut/Neoray/pkg/common"
	"github.com/hismailbulut/Neoray/pkg/opengl"
)

type Window struct {
	handle  *glfw.Window
	context *opengl.Context
	// info and cache
	dims         common.Rectangle[int]   // window dimensions used for restoring window from fullscreen
	events       WindowEventStack        // Cached event stack
	eventHandler func(event WindowEvent) // Event handler function will be called for every event at PollEvents call
	windowScale  int
}

// New creates a window and initializes an opengl context for it
// Im order to use context just call GL function of window
// You must call the Show function to show the window
func New(title string, width, height int, debugContext bool, windowScale int) (*Window, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New("Window dimensions must bigger than zero")
	}

	window := new(Window)

	window.windowScale = windowScale

	// Set opengl library version
	// TODO: make it 2.1 (needs some research)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	// We need to create forward compatible context for macos support.
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	if debugContext {
		glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	}

	// We are initializing window as hidden, and then we show it when mainloop begins
	glfw.WindowHint(glfw.Visible, glfw.False)
	// Framebuffer transparency not working on fullscreen when doublebuffer is on.
	glfw.WindowHint(glfw.DoubleBuffer, glfw.False)
	glfw.WindowHint(glfw.TransparentFramebuffer, glfw.True)
	// Scales window width and height to monitor
	glfw.WindowHint(glfw.ScaleToMonitor, glfw.True)
	// Focus to window after shown
	glfw.WindowHint(glfw.FocusOnShow, glfw.True)

	var err error
	window.handle, err = glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create glfw window: %s", err)
	}

	// This is important
	window.handle.MakeContextCurrent()

	// Disable v-sync, already disabled by default but make sure.
	glfw.SwapInterval(0)

	// Load opengl context
	window.context, err = opengl.New(glfw.GetProcAddress)
	if err != nil {
		return nil, err
	}

	// CALLBACKS

	window.handle.SetFramebufferSizeCallback(func(w *glfw.Window, width, height int) {
		window.events.Push(WindowEventResize, width, height)
		window.events.Push(WindowEventResize, width*windowScale, height*windowScale)
	})

	window.handle.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		window.events.Push(WindowEventKeyInput, key, scancode, action, mods)
	})

	window.handle.SetCharCallback(func(w *glfw.Window, char rune) {
		window.events.Push(WindowEventCharInput, char)
	})

	window.handle.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		window.events.Push(WindowEventMouseInput, button, action, mods)
	})

	window.handle.SetCursorPosCallback(func(w *glfw.Window, xpos, ypos float64) {
		window.events.Push(WindowEventMouseMove, xpos, ypos)
	})

	window.handle.SetScrollCallback(func(w *glfw.Window, xoff, yoff float64) {
		window.events.Push(WindowEventScroll, xoff, yoff)
	})

	window.handle.SetDropCallback(func(w *glfw.Window, names []string) {
		window.events.Push(WindowEventDrop, names)
	})

	window.handle.SetContentScaleCallback(func(w *glfw.Window, x, y float32) {
		window.events.Push(WindowEventScaleChanged)
	})

	return window, nil
}

func (window *Window) GL() *opengl.Context {
	return window.context
}

// You can use this if window closed by user but you don't want to close
// Immediately call after WindowEventClose received
func (window *Window) KeepAlive() {
	window.handle.SetShouldClose(false)
}

func (window *Window) Show() {
	window.handle.Show()
}

func (window *Window) IsVisible() bool {
	return window.handle.GetAttrib(glfw.Visible) == glfw.True
}

func (window *Window) SetTitle(title string) {
	window.handle.SetTitle(title)
}

func (window *Window) Move(pos common.Vector2[int]) {
	window.handle.SetPos(pos.X, pos.Y)
}

func (window *Window) Resize(size common.Vector2[int]) {
	if size.X <= 0 {
		size.X = window.Size().Width()
	}
	if size.Y <= 0 {
		size.Y = window.Size().Height()
	}
	window.handle.SetSize(size.X*window.windowScale, size.Y*window.windowScale)
}

func (window *Window) SetMinSize(minSize common.Vector2[int]) {
	window.handle.SetSizeLimits(minSize.X, minSize.Y, glfw.DontCare, glfw.DontCare)
}

func (window *Window) Dimensions() common.Rectangle[int] {
	X, Y := window.handle.GetPos()
	W, H := window.handle.GetSize()
	return common.Rectangle[int]{X: X, Y: Y, W: W, H: H}
}

func (window *Window) Size() common.Vector2[int] {
	W, H := window.handle.GetSize()
	return common.Vector2[int]{X: W * window.windowScale, Y: H * window.windowScale}
}

func (window *Window) Viewport() common.Rectangle[int] {
	return common.Rectangle[int]{X: 0, Y: 0, W: window.Size().Width(), H: window.Size().Height()}
}

func (window *Window) SetIcon(icons [3]image.Image) {
	// Set icons, images must png
	window.handle.SetIcon(icons[:])
}

// Not working on win11
func (window *Window) Raise() {
	if window.IsMinimized() {
		window.handle.Restore()
	}
	window.handle.Focus()
}

func (window *Window) Minimize() {
	window.handle.Iconify()
}

func (window *Window) IsMinimized() bool {
	return window.handle.GetAttrib(glfw.Iconified) == glfw.True
}

func (window *Window) Maximize() {
	window.handle.Maximize()
}

func (window *Window) IsMaximized() bool {
	return window.handle.GetAttrib(glfw.Maximized) == glfw.True
}

func (window *Window) ToggleFullscreen() {
	if window.handle.GetMonitor() == nil {
		// Store dimension for restoring
		window.dims = window.Dimensions()
		// Fulscreen to current monitor
		monitor := window.getCurrentMonitor(window.dims)
		videoMode := monitor.GetVideoMode()
		window.handle.SetMonitor(monitor, 0, 0, videoMode.Width, videoMode.Height, videoMode.RefreshRate)
	} else {
		// restore
		window.handle.SetMonitor(nil, window.dims.X, window.dims.Y, window.dims.W, window.dims.H, 0)
	}
}

func (window *Window) IsFullscreen() bool {
	return window.handle.GetMonitor() != nil
}

func (window *Window) Center() {
	windowRect := window.Dimensions()
	monitor := window.getCurrentMonitor(windowRect)
	mx, my := monitor.GetPos()
	videoMode := monitor.GetVideoMode()
	posX := mx + ((videoMode.Width / 2) - (windowRect.W / 2))
	posY := my + ((videoMode.Height / 2) - (windowRect.H / 2))
	window.handle.SetPos(posX, posY)
}

func (window *Window) ShowMouseCursor() {
	window.handle.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
}

func (window *Window) HideMouseCursor() {
	window.handle.SetInputMode(glfw.CursorMode, glfw.CursorHidden)
}

func (window *Window) DPI() float64 {
	_, y := window.handle.GetContentScale()
	return float64(96 * y)
}

func (window *Window) Destroy() {
	window.context.Destroy()
	window.handle.Destroy()
}

// Returns the monitor where the window currently is.
func (window *Window) getCurrentMonitor(windowRect common.Rectangle[int]) *glfw.Monitor {
	// Reference:
	// https://stackoverflow.com/a/31526753
	var bestMonitor *glfw.Monitor
	var bestOverlap int
	for _, monitor := range glfw.GetMonitors() {
		mx, my := monitor.GetPos()
		videoMode := monitor.GetVideoMode()
		mw, mh := videoMode.Width, videoMode.Height
		overlapX := common.Max(0, common.Min(windowRect.X+windowRect.W, mx+mw)-common.Max(windowRect.X, mx))
		overlapY := common.Max(0, common.Min(windowRect.Y+windowRect.H, my+mh)-common.Max(windowRect.Y, my))
		overlap := overlapX * overlapY
		if overlap > bestOverlap {
			bestOverlap = overlap
			bestMonitor = monitor
		}
	}
	return bestMonitor
}
