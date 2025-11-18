package krylr896

import (
	"fmt"
	"log"
	"testing"
	"time"
)

// test configuration
const testSerialPort = "/dev/ttyUSB0"

// TestLoRaConnection tests basic connection and configuration
func TestLoRaConnection(t *testing.T) {
	address := uint16(1)
	networkID := uint8(5)
	band := BandUSA
	mode := MODE_TRX
	baudRate := UartBaudRate_115200
	rfPower := uint8(15)

	// RF parameters
	params := Parameters{
		SpreadingFactor:    7,
		Bandwidth:          Bandwidth125KHz,
		CodingRate:         1,
		ProgrammedPreamble: 4,
	}

	// complete configuration
	config := Configuration{
		Address:       &address,
		NetworkID:     &networkID,
		Band:          &band,
		Parameter:     &params,
		Mode:          &mode,
		UartBaudRate:  &baudRate,
		RFOutputPower: &rfPower,
	}

	// create connection
	// the callback allows users to set their own logging mechanism
	lora, err := CreateConnectionDEBUG(testSerialPort, config, 10, "r1", func(name, msg string) {
		log.Printf("[DEBUG][%s] %s", name, msg)
	})
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err.Err)
	}
	defer lora.CloseConnection()

	t.Log("Successfully connected to /dev/ttyUSB0")
}

// TestLoRaSendMessage tests sending a message
func TestLoRaSendMessage(t *testing.T) {
	// minimal configuration
	config := Configuration{}

	lora, err := CreateConnectionDEBUG(testSerialPort, config, 10, "TestLoRaSendMessage", func(name, msg string) {
		log.Printf("[DEBUG][%s] %s", name, msg)
	})
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err.Err)
	}
	defer lora.CloseConnection()

	// send a test message to address 2
	if sendErr := lora.SendMessage(uint16(2), []byte("Hello, LoRa!")); sendErr != nil {
		t.Fatalf("Failed to send message: %v", sendErr.Err)
	}

	t.Logf("Successfully sent message")
}

// TestLoRaReceiveMessage tests receiving messages
func TestLoRaReceiveMessage(t *testing.T) {
	// minimal configuration
	config := Configuration{}

	// create connection
	lora, err := CreateConnectionDEBUG(testSerialPort, config, 10, "r1", func(name, msg string) {
		log.Printf("[DEBUG][%s] %s", name, msg)
	})
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err.Err)
	}
	defer lora.CloseConnection()

	t.Log("Waiting for incoming messages for 30 seconds...")

	// listen for messages with timeout
	timeout := time.After(30 * time.Second)
	messageReceived := false

	for !messageReceived {
		select {
		case msg := <-lora.RecievedData:
			t.Logf("Received message from address %d: %s (RSSI: %d, SNR: %d)",
				msg.Address,
				string(msg.Data[:msg.Length]),
				msg.ReceivedSignalStrengthIndicator,
				msg.SignalToNoiseRatio)
			messageReceived = true

		case errEvent := <-lora.Errors:
			if errEvent.Code != nil {
				t.Logf("Received error code: %d", *errEvent.Code)
			}
			if errEvent.Err != nil {
				t.Logf("Received error: %v", errEvent.Err)
			}

		case <-timeout:
			t.Log("Timeout - no messages received in 30 seconds")
			return
		}
	}
}

// TestLoRaRawCommand tests sending raw AT commands
func TestLoRaRawCommand(t *testing.T) {
	// minimal configuration
	config := Configuration{}

	// create connection
	lora, err := CreateConnectionDEBUG(testSerialPort, config, 10, "r1", func(name, msg string) {
		log.Printf("[DEBUG][%s] %s", name, msg)
	})
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err.Err)
	}
	defer lora.CloseConnection()

	// send a raw AT command to get the address
	responseChan := make(chan CommandResponse, 1)
	lora.Commands <- Command{
		Text:         "AT+ADDRESS?",
		ResponseChan: responseChan,
	}

	// wait for response
	select {
	case response := <-responseChan:
		if response.Error != nil {
			t.Fatalf("Command failed: %v", response.Error.Err)
		}
		t.Logf("Command response: %s", response.Response)

	case <-time.After(5 * time.Second):
		t.Fatal("Command timeout")
	}
}

// Example_basicUsage demonstrates basic library usage
func Example_basicUsage() {
	// configure the radio
	address := uint16(1)
	networkID := uint8(5)
	band := BandUSA

	config := Configuration{
		Address:   &address,
		NetworkID: &networkID,
		Band:      &band,
	}

	// create connection
	lora, err := CreateConnectionDEBUG(testSerialPort, config, 10, "r1", func(name, msg string) {
		log.Printf("[DEBUG][%s] %s", name, msg)
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err.Err)
		return
	}
	defer lora.CloseConnection()

	// send a message
	targetAddress := uint16(2)
	message := []byte("Hello!")
	if sendErr := lora.SendMessage(targetAddress, message); sendErr != nil {
		fmt.Printf("Send error: %v\n", sendErr.Err)
		return
	}

	// listen for incoming messages
	go func() {
		for {
			select {
			case msg := <-lora.RecievedData:
				fmt.Printf("Received: %s from %d\n",
					string(msg.Data[:msg.Length]),
					msg.Address)

			case errEvent := <-lora.Errors:
				if errEvent.Err != nil {
					fmt.Printf("Error: %v\n", errEvent.Err)
				}
			}
		}
	}()

	// keep running
	time.Sleep(60 * time.Second)
}
