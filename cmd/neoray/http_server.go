package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	NEORAY_SERVER_PORT = "17717"
)

var (
	SignalMutex        sync.Mutex
	SignalOpenFile     int32
	SignalOpenFileName = ""
)

// Send signals to other instances
func SendOpenFile(fileName string) bool {
	fileAbs, err := filepath.Abs(fileName)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "File error:", err)
		return false
	}
	_, err = os.Stat(fileAbs)
	if err != nil {
		log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "File error:", err)
		return false
	}
	resp, err := http.Get("http://localhost:" + NEORAY_SERVER_PORT + "/openfile/" + fileName)
	if err != nil {
		log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "Connection failed:", err)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		log_message(LOG_LEVEL_DEBUG, LOG_TYPE_NEORAY, "Response is not OK:", resp.StatusCode)
		return false
	}
	return true
}

// Create a server and process incoming signals.
func CreateServer() {
	handler := http.NewServeMux()

	handler.HandleFunc("/openfile/", func(rw http.ResponseWriter, r *http.Request) {
		fileName := strings.Replace(r.URL.Path, "/openfile/", "", 1)
		atomicSetBool(&SignalOpenFile, true)
		SignalMutex.Lock()
		defer SignalMutex.Unlock()
		SignalOpenFileName = fileName
		rw.WriteHeader(http.StatusOK)
	})

	go func() {
		err := http.ListenAndServe(":"+NEORAY_SERVER_PORT, handler)
		if err != nil {
			log_message(LOG_LEVEL_ERROR, LOG_TYPE_NEORAY, "Failed to start local server:", err)
			return
		}
	}()
}
