package main

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DEFAULT_PORT = "17717"

	SIGNAL_OK = "OK\n"

	SIGNAL_CHECK_CONNECTION = "CHECK\n"
	SIGNAL_CLOSE_CONNECTION = "CLOSE\n"

	SIGNAL_OPEN_FILE   = "OPENFILE"
	SIGNAL_GOTO_LINE   = "GOTOLINE"
	SIGNAL_GOTO_COLUMN = "GOTOCOLUMN"
)

// This is a tcp server/client implementation of neoray.
// We are using tcp for communicating between neoray instances.
// This implementation may has security issues. If you are a
// pro socket programmer please review this source code.

// Signals must end with newline
// Signal and args must be separated with null character

type TCPClient struct {
	connection net.Conn
	data       chan string
	resp       chan bool
}

func CreateClient() (*TCPClient, error) {
	client := TCPClient{
		data: make(chan string),
		resp: make(chan bool),
	}
	c, err := net.Dial("tcp", ":"+DEFAULT_PORT)
	if err != nil {
		return nil, err
	}
	client.connection = c
	log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Connected to", c.RemoteAddr())
	go func() {
		for {
			data := <-client.data
			_, err := c.Write([]byte(data))
			if err != nil {
				log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Failed to send signal:", err)
				client.resp <- false
				continue
			}
			resp, err := bufio.NewReader(c).ReadString('\n')
			if err != nil {
				log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Failed to get response:", err)
				client.resp <- false
				continue
			}
			if resp == SIGNAL_CLOSE_CONNECTION {
				client.connection.Close()
				client.resp <- true
				log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Disconnected from server.")
				return
			}
			client.resp <- true
		}
	}()
	return &client, nil
}

func (client *TCPClient) SendSignal(signal string, args ...string) bool {
	log_debug("Sending signal:", signal, args)
	for _, arg := range args {
		signal += "\x00" + arg
	}
	if signal[len(signal)-1] != '\n' {
		signal += "\n"
	}
	client.data <- signal
	select {
	case result := <-client.resp:
		if !result {
			return false
		}
		break
	case <-time.Tick(time.Second):
		log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Signal timeout.")
		return false
	}
	return true
}

func (client *TCPClient) Close() {
	client.SendSignal(SIGNAL_CLOSE_CONNECTION)
	close(client.data)
	close(client.resp)
}

type TCPServer struct {
	listener     net.Listener
	dataReceived AtomicBool
	dataMutex    sync.Mutex
	data         []string
}

// Create a server and process incoming signals.
func CreateServer() (*TCPServer, error) {
	server := TCPServer{}
	l, err := net.Listen("tcp", ":"+DEFAULT_PORT)
	if err != nil {
		return nil, err
	}
	server.listener = l
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Server closed.")
				return
			}
			log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "New client connected:", c.RemoteAddr())
			// handle connection concurrently
			go func() {
				defer c.Close()
				for {
					var resp string
					data, err := bufio.NewReader(c).ReadString('\n')
					if err != nil {
						log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Failed to read client data:", err)
						break
					}
					switch data {
					case SIGNAL_CHECK_CONNECTION:
						resp = SIGNAL_OK
						break
					case SIGNAL_CLOSE_CONNECTION:
						resp = SIGNAL_CLOSE_CONNECTION
						break
					default:
						server.dataReceived.Set(true)
						server.dataMutex.Lock()
						server.data = append(server.data, data)
						server.dataMutex.Unlock()
						resp = SIGNAL_OK
						break
					}
					_, err = c.Write([]byte(resp))
					if err != nil {
						log_message(LOG_LEVEL_WARN, LOG_TYPE_NEORAY, "Failed to send response to client.")
					}
					if resp == SIGNAL_CLOSE_CONNECTION {
						log_message(LOG_LEVEL_TRACE, LOG_TYPE_NEORAY, "Client disconnected.")
						return
					}
				}
			}()
		}
	}()
	return &server, nil
}

func (server *TCPServer) Process() {
	if server.dataReceived.Get() {
		server.dataMutex.Lock()
		defer server.dataMutex.Unlock()
		for _, sig := range server.data {
			args := strings.Split(strings.Split(sig, "\n")[0], "\x00")
			switch args[0] {
			case SIGNAL_OPEN_FILE:
				EditorSingleton.nvim.openFile(args[1])
				break
			case SIGNAL_GOTO_LINE:
				ln, err := strconv.Atoi(args[1])
				if err == nil {
					EditorSingleton.nvim.gotoLine(ln)
				}
				break
			case SIGNAL_GOTO_COLUMN:
				cl, err := strconv.Atoi(args[1])
				if err == nil {
					EditorSingleton.nvim.gotoColumn(cl)
				}
				break
			default:
				log_debug("Received invalid signal:", sig)
				break
			}
		}
		EditorSingleton.window.raise()
		server.data = nil
		server.dataReceived.Set(false)
	}
}

func (server *TCPServer) Close() {
	server.listener.Close()
}
