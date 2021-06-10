package main

import (
	"fmt"
	"net/http"
	"sync"
)

const (
	SERVER_DEFAULT_PORT = "17717"

	FORMAT_OPENFILE = iota
	FORMAT_GOTOLINE
	FORMAT_GOTOCOL
)

var (
	SignalMutex    sync.Mutex
	SignalReceived AtomicValue
	SignalContent  []string

	SignalScriptFormats = map[int]string{
		FORMAT_OPENFILE: ":edit %s",
		FORMAT_GOTOLINE: "call cursor(%d, 0)",
		FORMAT_GOTOCOL:  "call cursor(0, %d)",
	}
)

func SendSignal(format int, args ...interface{}) bool {
	defer measure_execution_time("SendSignal")()
	formatStr, ok := SignalScriptFormats[format]
	if !ok {
		return false
	}
	formatted := fmt.Sprintf(formatStr, args...)
	requesturl := "http://localhost:" + SERVER_DEFAULT_PORT + "/" + formatted
	resp, err := http.Get(requesturl)
	if err != nil {
		log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "Connection failed:", err)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "Response is not OK:", resp.StatusCode)
		return false
	}
	log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "Signal Sended:", requesturl)
	return true
}

func ProcessSignals() {
	if SignalReceived.GetBool() {
		SignalMutex.Lock()
		defer SignalMutex.Unlock()
		for _, signal := range SignalContent {
			log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "Signal received:", signal)
			EditorSingleton.nvim.ExecuteVimScript(signal)
		}
		SignalContent = nil
		SignalReceived.SetBool(false)
		EditorSingleton.window.handle.Raise()
	}
}

// Create a server and process incoming signals.
func CreateServer() {
	handler := http.NewServeMux()

	handler.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		SignalMutex.Lock()
		defer SignalMutex.Unlock()
		content := r.URL.Path[1:]
		if content != "" {
			SignalContent = append(SignalContent, content)
			SignalReceived.SetBool(true)
		}
	})

	go func() {
		log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "Listening port:", SERVER_DEFAULT_PORT)
		err := http.ListenAndServe(":"+SERVER_DEFAULT_PORT, handler)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to start local server:", err)
			return
		}
	}()
}
