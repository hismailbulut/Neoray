package window

import "github.com/go-gl/glfw/v3.3/glfw"

type WindowEventType uint32

const (
	WindowEventRefresh WindowEventType = iota
	WindowEventResize
	WindowEventKeyInput
	WindowEventCharInput
	WindowEventMouseInput
	WindowEventMouseMove
	WindowEventScroll
	WindowEventDrop
	WindowEventScaleChanged
	WindowEventClose
)

func (Type WindowEventType) String() string {
	switch Type {
	case WindowEventRefresh:
		return "WindowEventRefresh"
	case WindowEventResize:
		return "WindowEventResize"
	case WindowEventKeyInput:
		return "WindowEventKeyInput"
	case WindowEventCharInput:
		return "WindowEventCharInput"
	case WindowEventMouseInput:
		return "WindowEventMouseInput"
	case WindowEventMouseMove:
		return "WindowEventMouseMove"
	case WindowEventScroll:
		return "WindowEventScroll"
	case WindowEventDrop:
		return "WindowEventDrop"
	case WindowEventClose:
		return "WindowEventClose"
	default:
		panic("unknown window event type")
	}
}

type WindowEvent struct {
	Type   WindowEventType
	Params []any
}

type WindowEventStack []WindowEvent

func (stack *WindowEventStack) Push(eventType WindowEventType, params ...any) {
	*stack = append(*stack, WindowEvent{
		Type:   eventType,
		Params: params,
	})
}

func (window *Window) SetEventHandler(eventHandler func(event WindowEvent)) {
	window.handle.SetRefreshCallback(func(w *glfw.Window) {
		eventHandler(WindowEvent{Type: WindowEventRefresh})
	})
	window.eventHandler = eventHandler
}

func (window *Window) PollEvents() {
	glfw.PollEvents()
	resizeIndex := -1
	for i := 0; i < len(window.events); i++ {
		event := window.events[i]
		if event.Type == WindowEventResize {
			resizeIndex = i
		} else {
			window.eventHandler(event)
		}
	}
	// NOTE: Only send last resize event because when refreshing happens glfw
	// pollevents call blocked but glfw stores all resize events and sends them
	// at same time, only the last one is the actual size of the window
	if resizeIndex != -1 {
		window.eventHandler(window.events[resizeIndex])
	}
	// Check close event
	if window.handle.ShouldClose() {
		window.eventHandler(WindowEvent{Type: WindowEventClose})
	}
	// Clear event stack
	window.events = window.events[0:0]
}
