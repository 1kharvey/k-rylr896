package krylr896

import (
	"encoding/hex"
	"fmt"

	"go.bug.st/serial"
)

type lora struct {
	Errors       chan ErrorEvent   // read uncategorized errors
	RecievedData chan RecievedData // read recieved messages
	Commands     chan Command      // commands are written to here by the user or internally
	port         serial.Port
	IS_DEBUG     bool                         // enable debug logging
	debugName    string                       // debug name prefix for logging
	debugFunc    func(name string, msg string) // debug callback function
}

// debugLog logs a debug message using the debug callback
func (Lora *lora) debugLog(format string, args ...interface{}) {
	if Lora.IS_DEBUG && Lora.debugFunc != nil {
		msg := fmt.Sprintf(format, args...)
		Lora.debugFunc(Lora.debugName, msg)
	}
}

// createConnectionInternal is the internal connection creation function
func createConnectionInternal(serialInterfaceName string, config Configuration, buffLen int, debug bool, debugName string, debugFunc func(string, string)) (Lora *lora, errEvent *ErrorEvent) {
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(serialInterfaceName, mode)
	if err != nil {
		return nil, &ErrorEvent{Code: nil, Err: err}
	}

	Lora = &lora{
		Commands:     make(chan Command, buffLen),
		Errors:       make(chan ErrorEvent, buffLen),
		RecievedData: make(chan RecievedData, buffLen),
		port:         port,
		IS_DEBUG:     debug,
		debugName:    debugName,
		debugFunc:    debugFunc,
	}

	// start run in background
	go run(Lora)

	// set configuration
	if errEvent := Lora.SetConfig(config); errEvent != nil {
		Lora.CloseConnection()
		return nil, &ErrorEvent{Code: errEvent.Code, Err: fmt.Errorf("failed to set configuration: %w", errEvent.Err)}
	}

	return Lora, nil
}

// CreateConnection attaches to a uart serial port, and a desired buffer length and returns a lora object
func CreateConnection(serialInterfaceName string, config Configuration, buffLen int) (Lora *lora, errEvent *ErrorEvent) {
	return createConnectionInternal(serialInterfaceName, config, buffLen, false, "", nil)
}

// CreateConnectionDEBUG creates a connection with debug logging enabled
func CreateConnectionDEBUG(serialInterfaceName string, config Configuration, buffLen int, debugName string, debugFunc func(string, string)) (Lora *lora, errEvent *ErrorEvent) {
	return createConnectionInternal(serialInterfaceName, config, buffLen, true, debugName, debugFunc)
}

// CloseConnection closes the connection to the LoRa module
func (Lora *lora) CloseConnection() (err error) {
	close(Lora.Commands)
	return Lora.port.Close()
}

// SetConfig applies a configuration to the radio, nil fields are ignored
func (Lora *lora) SetConfig(config Configuration) *ErrorEvent {
	// channel to receive command results
	resultChan := make(chan CommandResponse, 1)

	// helper function to send a command and wait for response
	sendCommand := func(cmd string) *ErrorEvent {
		Lora.Commands <- Command{
			Text:         cmd,
			ResponseChan: resultChan,
		}

		// wait for response
		resp := <-resultChan
		return resp.Error
	}

	// set ADDRESS if not nil
	if config.Address != nil {
		if err := sendCommand(fmt.Sprintf("AT+ADDRESS=%d", *config.Address)); err != nil {
			if err.Err != nil {
				return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set address: %w", err.Err)}
			}
			return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set address")}
		}
	}

	// set NETWORKID if not nil
	if config.NetworkID != nil {
		if err := sendCommand(fmt.Sprintf("AT+NETWORKID=%d", *config.NetworkID)); err != nil {
			if err.Err != nil {
				return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set network ID: %w", err.Err)}
			}
			return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set network ID")}
		}
	}

	// set BAND if not nil
	if config.Band != nil {
		if err := sendCommand(fmt.Sprintf("AT+BAND=%d", *config.Band)); err != nil {
			if err.Err != nil {
				return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set band: %w", err.Err)}
			}
			return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set band")}
		}
	}

	// set PARAMETER if not nil
	if config.Parameter != nil {
		cmd := fmt.Sprintf("AT+PARAMETER=%d,%d,%d,%d",
			config.Parameter.SpreadingFactor,
			config.Parameter.Bandwidth,
			config.Parameter.CodingRate,
			config.Parameter.ProgrammedPreamble)
		if err := sendCommand(cmd); err != nil {
			if err.Err != nil {
				return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set parameter: %w", err.Err)}
			}
			return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set parameter")}
		}
	}

	// set MODE if not nil
	if config.Mode != nil {
		if err := sendCommand(fmt.Sprintf("AT+MODE=%d", *config.Mode)); err != nil {
			if err.Err != nil {
				return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set mode: %w", err.Err)}
			}
			return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set mode")}
		}
	}

	// set IPR (UART baud rate) if not nil
	if config.UartBaudRate != nil {
		if err := sendCommand(fmt.Sprintf("AT+IPR=%d", *config.UartBaudRate)); err != nil {
			if err.Err != nil {
				return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set UART baud rate: %w", err.Err)}
			}
			return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set UART baud rate")}
		}
	}

	// set CPIN (encryption key) if not nil
	if config.EncryptionKey != nil {
		hexKey := hex.EncodeToString(config.EncryptionKey[:])
		if err := sendCommand(fmt.Sprintf("AT+CPIN=%s", hexKey)); err != nil {
			if err.Err != nil {
				return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set encryption key: %w", err.Err)}
			}
			return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set encryption key")}
		}
	}

	// set CRFOP (RF output power) if not nil
	if config.RFOutputPower != nil {
		if err := sendCommand(fmt.Sprintf("AT+CRFOP=%d", *config.RFOutputPower)); err != nil {
			if err.Err != nil {
				return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set RF output power: %w", err.Err)}
			}
			return &ErrorEvent{Code: err.Code, Err: fmt.Errorf("failed to set RF output power")}
		}
	}

	return nil
}

// SendMessage sends bytes to specified address
func (Lora *lora) SendMessage(address uint16, data []byte) *ErrorEvent {
	if len(data) > 240 {
		return &ErrorEvent{Code: nil, Err: fmt.Errorf("data length %d exceeds maximum of 240 bytes", len(data))}
	}

	// channel to receive command result
	resultChan := make(chan CommandResponse, 1)

	cmd := fmt.Sprintf("AT+SEND=%d,%d,%s", address, len(data), string(data))
	Lora.Commands <- Command{
		Text:         cmd,
		ResponseChan: resultChan,
	}

	// wait for response
	resp := <-resultChan
	if resp.Error != nil {
		return &ErrorEvent{Code: resp.Error.Code, Err: fmt.Errorf("send failed: %w", resp.Error.Err)}
	}

	return nil
}
