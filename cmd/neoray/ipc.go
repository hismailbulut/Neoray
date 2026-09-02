package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"time"

	"github.com/hismailbulut/Neoray/pkg/logger"
)

const (
	DEFAULT_ADDRESS     = "localhost:17717"
	DEFAULT_TIMEOUT     = time.Second / 2
	DEFAULT_BUFFER_SIZE = 1024
)

type IpcMessageType int

type IpcFuncCall struct {
	MsgType    IpcMessageType
	MacAddress uint64
	Args       []any
}

const (
	IPC_MSG_TYPE_OK IpcMessageType = iota
	IPC_MSG_TYPE_CLOSE_CONN
	IPC_MSG_TYPE_OPEN_FILE
	IPC_MSG_TYPE_GOTO_LINE
	IPC_MSG_TYPE_GOTO_COLUMN
)

func (msgType IpcMessageType) String() string {
	switch msgType {
	case IPC_MSG_TYPE_OK:
		return "OK"
	case IPC_MSG_TYPE_CLOSE_CONN:
		return "CLOSE"
	case IPC_MSG_TYPE_OPEN_FILE:
		return "OPEN_FILE"
	case IPC_MSG_TYPE_GOTO_LINE:
		return "GOTO_LINE"
	case IPC_MSG_TYPE_GOTO_COLUMN:
		return "GOTO_COLUMN"
	default:
		panic("Invalid message type.")
	}
}

func getMacAddress() uint64 {
	interfaces, err := net.Interfaces()
	if err != nil {
		return 0
	}
	for _, i := range interfaces {
		if i.Flags&net.FlagUp != 0 && !bytes.Equal(i.HardwareAddr, nil) {
			// Skip locally administered addresses
			if i.HardwareAddr[0]&2 == 2 {
				continue
			}
			var mac uint64
			for j, b := range i.HardwareAddr {
				if j >= 8 {
					break
				}
				mac <<= 8
				mac += uint64(b)
			}
			return mac
		}
	}
	return 0
}

type IpcClient struct {
	conn net.Conn
	mac  uint64
}

func CreateClient() (*IpcClient, error) {
	// NOTE: Timeout parameter may not be enough for tcp connection, but speeds up startup
	conn, err := net.DialTimeout("tcp", DEFAULT_ADDRESS, DEFAULT_TIMEOUT)
	if err != nil {
		return nil, err
	}
	client := IpcClient{
		conn: conn,
		mac:  getMacAddress(),
	}
	return &client, nil
}

func (client *IpcClient) Call(msgType IpcMessageType, args ...any) bool {
	logger.Debug("Sending signal:", msgType)
	// Encode function
	jsonData, err := json.Marshal(IpcFuncCall{
		MsgType:    msgType,
		MacAddress: client.mac,
		Args:       args,
	})
	if err != nil {
		logger.Warn("Failed to encode function call:", err)
		return false
	}
	_, err = client.conn.Write(jsonData)
	if err != nil {
		logger.Warn("Failed to send signal:", err)
		return false
	}
	// Read response from server
	resp := make([]byte, DEFAULT_BUFFER_SIZE)
	n, err := client.conn.Read(resp)
	if err != nil {
		logger.Warn("Failed to read response:", err)
		return false
	}
	resp = resp[:n]
	// Decode response
	var funcCall IpcFuncCall
	err = json.Unmarshal(resp, &funcCall)
	if err != nil {
		logger.Warn("Failed to decode response:", err)
		return false
	}
	// Check mac address
	// NOTE: Actually we don't need to check for mac address in client because
	// client already sent command to execute but anyway, it seems more secure
	if funcCall.MacAddress != client.mac {
		logger.Warn("Signal rejected: Connected server is not running on same machine.")
		return false
	}
	// First client sends close call to server, if server accepts, it resends
	// close call to client and closes its connection. After server closes, client
	// receives a close call and closes itself.
	if funcCall.MsgType == IPC_MSG_TYPE_CLOSE_CONN {
		logger.Trace("Disconnected from server.")
		client.conn.Close()
		return true
	} else if funcCall.MsgType != IPC_MSG_TYPE_OK {
		// Server always has to send OK. if we are not receive any ok this means there is a
		// problem in connection
		logger.Trace("Client sent non OK response:", funcCall.MsgType)
		return false
	}
	return true
}

func (client *IpcClient) Close() {
	client.Call(IPC_MSG_TYPE_CLOSE_CONN)
	logger.Trace("Client closed.")
}

// Server is a listener, not sends messages but processes incoming messages from clients
type IpcServer struct {
	editor    *Editor
	listener  net.Listener
	mac       uint64
	callsChan chan IpcFuncCall
}

// Create a server and process incoming signals.
func CreateServer(editor *Editor) (*IpcServer, error) {
	listener, err := net.Listen("tcp", DEFAULT_ADDRESS)
	if err != nil {
		return nil, err
	}
	server := IpcServer{
		editor:    editor,
		listener:  listener,
		mac:       getMacAddress(),
		callsChan: make(chan IpcFuncCall, 16),
	}
	go server.mainLoop()
	return &server, nil
}

func (server *IpcServer) mainLoop() {
	// Encode ok message because we always use it
	encodedOK, err := json.Marshal(IpcFuncCall{MsgType: IPC_MSG_TYPE_OK, MacAddress: server.mac})
	if err != nil {
		logger.Error("Failed to encode OK:", err)
		return
	}
	// Encode CLOSE message because we always use it
	encodedCLOSE, err := json.Marshal(IpcFuncCall{MsgType: IPC_MSG_TYPE_CLOSE_CONN, MacAddress: server.mac})
	if err != nil {
		logger.Error("Failed to encode CLOSE:", err)
		return
	}
	for {
		conn, err := server.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				logger.Trace("Server closed.")
			} else {
				logger.Error("Server closed because of errors:", err)
			}
			return
		}
		logger.Trace("New client connected:", conn.RemoteAddr())
		// handle connection concurrently
		go func() {
			defer conn.Close()
			for {
				data := make([]byte, DEFAULT_BUFFER_SIZE)
				n, err := conn.Read(data)
				if err != nil {
					logger.Warn("Failed to read client data:", err)
					continue
				}
				data = data[:n]
				// decode data
				var funcCall IpcFuncCall
				err = json.Unmarshal(data, &funcCall)
				if err != nil {
					logger.Warn("Failed to decode client data:", err)
					continue
				}
				// check mac address
				if funcCall.MacAddress != server.mac {
					logger.Warn("Signal Rejected: Connected client is not running on same machine.")
					break
				}
				switch funcCall.MsgType {
				case IPC_MSG_TYPE_CLOSE_CONN:
					logger.Trace("Client", conn.RemoteAddr(), "disconnected.")
					_, err = conn.Write(encodedCLOSE)
					if err != nil {
						logger.Warn("Failed to send response to client.")
					}
					return
				default:
					server.callsChan <- funcCall
					_, err = conn.Write(encodedOK)
					if err != nil {
						logger.Warn("Failed to send response to client.")
					}
				}
			}
		}()
	}
}

func (server *IpcServer) Update() {
	for len(server.callsChan) > 0 {
		call := <-server.callsChan
		// bool, for JSON booleans
		// float64, for JSON numbers
		// string, for JSON strings
		// []any, for JSON arrays
		// map[string]any, for JSON objects
		// nil for JSON null
		switch call.MsgType {
		case IPC_MSG_TYPE_OPEN_FILE:
			path := call.Args[0].(string)
			server.editor.nvim.EditFile(path)
		case IPC_MSG_TYPE_GOTO_LINE:
			line := int(call.Args[0].(float64))
			server.editor.nvim.MoveCursor(line, 0)
		case IPC_MSG_TYPE_GOTO_COLUMN:
			column := int(call.Args[0].(float64))
			server.editor.nvim.MoveCursor(0, column)
		default:
			logger.Warn("Server received invalid signal:", call)
		}
		server.editor.window.Raise()
	}
}

func (server *IpcServer) Close() {
	server.listener.Close()
	logger.Debug("IPC server closed")
}
