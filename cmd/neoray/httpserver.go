package main

import (
	"fmt"
	"net/http"
	"sync"
)

const (
	SERVER_DEFAULT_PORT = "17717"

	FORMAT_CHECK_SIGNAL = "this is a signal for checking already open instance"
	FORMAT_OPENFILE     = ":edit %s"
	FORMAT_GOTOLINE     = "call cursor(%d, 0)"
	FORMAT_GOTOCOL      = "call cursor(0, %d)"
)

var (
	ServerCreated  bool
	SignalMutex    sync.Mutex
	SignalReceived AtomicBool
	SignalContent  []string
)

func SendSignal(format string, args ...interface{}) bool {
	defer measure_execution_time("SendSignal")()

	requesturl := "http://localhost:" + SERVER_DEFAULT_PORT + "/" + fmt.Sprintf(format, args...)
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
	if !ServerCreated {
		return
	}
	if SignalReceived.GetBool() {
		SignalMutex.Lock()
		defer SignalMutex.Unlock()
		for _, signal := range SignalContent {
			log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "Signal received:", signal)
			EditorSingleton.nvim.ExecuteVimScript(signal)
		}
		SignalContent = nil
		SignalReceived.SetBool(false)
		EditorSingleton.window.handle.RequestAttention()
	}
}

// Create a server and process incoming signals.
func CreateServer() {
	handler := http.NewServeMux()

	handler.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		content := r.URL.Path[1:]
		if content != "" && content != FORMAT_CHECK_SIGNAL {
			SignalReceived.SetBool(true)
			SignalMutex.Lock()
			defer SignalMutex.Unlock()
			SignalContent = append(SignalContent, content)
		}
	})

	ServerCreated = true
	go func() {
		log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "Listening port:", SERVER_DEFAULT_PORT)
		err := http.ListenAndServe(":"+SERVER_DEFAULT_PORT, handler)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to start local server:", err)
			return
		}
	}()
}
