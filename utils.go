package krylr896

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// run this in a goroutine, it will quit when we close
func run(Lora *lora) {
	reader := bufio.NewReader(Lora.port)
	commandInProgress := false
	var currentResponseChan chan CommandResponse
	var commandTimeout <-chan time.Time

	// channel to receive lines from the port
	portLines := make(chan string, 10)
	portErrors := make(chan error, 1)

	// goroutine to continuously read from port
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				portErrors <- err
				return
			}
			Lora.debugLog("RX: %q", line)
			portLines <- line
		}
	}()

	for {
		select {
		case cmd, ok := <-Lora.Commands:
			if !ok {
				return
			}
			if cmd.Text == "" {
				return
			}

			// send command to port
			cmdString := cmd.Text + "\r\n"
			Lora.debugLog("TX: %q", cmdString)
			_, err := Lora.port.Write([]byte(cmdString))
			if err != nil {
				Lora.debugLog("TX Error: %v", err)
				if cmd.ResponseChan != nil {
					code := UNK_ERR
					cmd.ResponseChan <- CommandResponse{Error: &ErrorEvent{Code: &code, Err: err}}
				}
				continue
			}

			commandInProgress = true
			currentResponseChan = cmd.ResponseChan
			commandTimeout = time.After(10 * time.Second)

		case line := <-portLines:
			if commandInProgress {
				// this is a response to our command
				response := parseCommandResponse(line, Lora)
				if currentResponseChan != nil {
					currentResponseChan <- response
				}
				commandInProgress = false
				currentResponseChan = nil
				commandTimeout = nil

				// sleep 4ms between commands, I've reached out to the manufacturer to ask why this is necessary
				time.Sleep(4 * time.Millisecond)
			} else {
				// this is unsolicited data - classify it
				classifyOutput(line, Lora)
			}

		case <-commandTimeout:
			// command timeout occurred
			if commandInProgress && currentResponseChan != nil {
				currentResponseChan <- CommandResponse{
					Response: "",
					Error:    &ErrorEvent{Code: nil, Err: fmt.Errorf("command timeout after 10 seconds")},
				}
			}
			commandInProgress = false
			currentResponseChan = nil
			commandTimeout = nil

		case err := <-portErrors:
			// handle port read error - send to Errors channel
			select {
			case Lora.Errors <- ErrorEvent{Code: nil, Err: err}:
			default:
				// channel is full, drop error
			}
			return
		}
	}
}

// parseCommandResponse parses a command response line and returns a CommandResponse
func parseCommandResponse(line string, Lora *lora) CommandResponse {
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")

	Lora.debugLog("Parsing command response: %q", line)

	// check if it's an OK response
	if strings.HasPrefix(line, "+OK") {
		Lora.debugLog("Parsed as OK response")
		return CommandResponse{Response: line, Error: nil}
	}

	// check if it's an error response (e.g., "+ERR=1")
	if errCodeStr, found := strings.CutPrefix(line, "+ERR="); found {
		if errCode, err := strconv.Atoi(errCodeStr); err == nil {
			Lora.debugLog("Parsed as error response: code=%d", errCode)
			return CommandResponse{Response: line, Error: &ErrorEvent{Code: &errCode, Err: nil}}
		}
	}

	// anything else is a data response (e.g., +ADDRESS=1, +BAND=915000000, plain integers, etc.)
	Lora.debugLog("Parsed as data response: %q", line)
	return CommandResponse{Response: line, Error: nil}
}

// classifyOutput classifies unsolicited output as either a received message or an error
func classifyOutput(line string, Lora *lora) {
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")

	Lora.debugLog("Classifying unsolicited output: %q", line)

	// check if it's a received message (format: +RCV=<Address>,<Length>,<Data>,<RSSI>,<SNR>)
	if payload, found := strings.CutPrefix(line, "+RCV="); found {
		Lora.debugLog("Detected received message: %q", payload)
		// parse received message and send to RecievedData channel
		if msg, ok := parseReceivedMessage(payload, Lora); ok {
			Lora.debugLog("Parsed received message: addr=%d, len=%d, rssi=%d, snr=%d",
				msg.Address, msg.Length, msg.ReceivedSignalStrengthIndicator, msg.SignalToNoiseRatio)
			select {
			case Lora.RecievedData <- msg:
			default:
				// channel is full, drop message
				Lora.debugLog("RecievedData channel full, dropping message")
			}
		}
		return
	}

	// check if it's an error code (format: +ERR=<code>)
	if errCodeStr, found := strings.CutPrefix(line, "+ERR="); found {
		if errCode, err := strconv.Atoi(errCodeStr); err == nil {
			Lora.debugLog("Detected unsolicited error code: %d", errCode)
			select {
			case Lora.Errors <- ErrorEvent{Code: &errCode, Err: nil}:
			default:
				// channel is full, drop error
				Lora.debugLog("Errors channel full, dropping error")
			}
		}
		return
	}

	// unknown unsolicited data - send to Errors channel as Go error
	Lora.debugLog("Unknown unsolicited data: %q", line)
	select {
	case Lora.Errors <- ErrorEvent{Code: nil, Err: fmt.Errorf("unknown unsolicited data: %s", line)}:
	default:
		// channel is full, drop error
		Lora.debugLog("Errors channel full, dropping error")
	}
}

// parseReceivedMessage parses a received message payload into a RecievedData struct
// Format: <Address>,<Length>,<Data>,<RSSI>,<SNR>
// Example: 50,5,HELLO,-99,40
func parseReceivedMessage(payload string, Lora *lora) (RecievedData, bool) {
	var msg RecievedData

	Lora.debugLog("Parsing received message payload: %q", payload)

	// find first comma to get address
	idx1 := strings.Index(payload, ",")
	if idx1 == -1 {
		return msg, false
	}

	// parse address
	addr, err := strconv.ParseUint(payload[:idx1], 10, 16)
	if err != nil {
		return msg, false
	}
	msg.Address = uint16(addr)

	// find second comma to get length
	remaining := payload[idx1+1:]
	idx2 := strings.Index(remaining, ",")
	if idx2 == -1 {
		return msg, false
	}

	// parse length
	length, err := strconv.ParseUint(remaining[:idx2], 10, 8)
	if err != nil {
		return msg, false
	}
	msg.Length = uint8(length)

	// extract data (next <length> bytes)
	dataStart := idx1 + 1 + idx2 + 1
	if dataStart+int(length) > len(payload) {
		return msg, false
	}
	dataEnd := dataStart + int(length)
	copy(msg.Data[:], payload[dataStart:dataEnd])

	// find comma after data to get RSSI and SNR
	afterData := payload[dataEnd:]
	if !strings.HasPrefix(afterData, ",") {
		return msg, false
	}
	afterData = afterData[1:] // skip the comma

	// find last comma to separate RSSI and SNR
	idx3 := strings.Index(afterData, ",")
	if idx3 == -1 {
		return msg, false
	}

	// parse RSSI
	rssi, err := strconv.ParseInt(afterData[:idx3], 10, 8)
	if err != nil {
		return msg, false
	}
	msg.ReceivedSignalStrengthIndicator = int8(rssi)

	// parse SNR
	snr, err := strconv.ParseInt(afterData[idx3+1:], 10, 8)
	if err != nil {
		return msg, false
	}
	msg.SignalToNoiseRatio = int8(snr)

	return msg, true
}
