# k-rylr896

Go library for RYLR896/RYLR40x LoRa radio modules using AT commands over serial.

## Installation

```bash
go get github.com/1kharvey/k-rylr896
```

## Quick Start

### Creating a Connection

Create a connection by specifying the serial port, baud rate, configuration, and buffer length:

```go
lora, err := krylr896.CreateConnection("/dev/ttyUSB0", krylr896.UartBaudRate_115200, config, 10)
```

For debug output with custom logging, use `CreateConnectionDEBUG`:

```go
lora, err := krylr896.CreateConnectionDEBUG("/dev/ttyUSB0", krylr896.UartBaudRate_115200, config, 10, "MyRadio", func(name, msg string) {
    log.Printf("[DEBUG][%s] %s", name, msg)
})
```

### Configuring the Radio

The `Configuration` struct allows you to set radio parameters. **All fields are pointers and optional** - any field set to `nil` will not be configured on the radio:

```go
type Configuration struct {
    Address       *uint16     // Radio address (0-65535)
    NetworkID     *uint8      // Network ID (0-16), must match for communication
    Band          *uint32     // Frequency band in Hz (e.g., 915000000)
    Parameter     *Parameters // RF transmission parameters
    Mode          *uint8      // Operating mode (0=TRX, 1=SLEEP)
    UartBaudRate  *int        // UART baud rate (300-115200)
    EncryptionKey *[16]byte   // AES128 encryption key (16 bytes)
    RFOutputPower *uint8      // RF output power in dBm (0-15)
}
```

Example configuration setting only address and network:

```go
address := uint16(1)
networkID := uint8(5)
config := krylr896.Configuration{
    Address:   &address,
    NetworkID: &networkID,
    // All other fields are nil and will not be configured
}
```

RF parameters can be configured with the `Parameters` struct:

```go
params := krylr896.Parameters{
    SpreadingFactor:    7,
    Bandwidth:          krylr896.Bandwidth125KHz,
    CodingRate:         1,
    ProgrammedPreamble: 4,
}
```

Complete configuration example:

```go
config := krylr896.Configuration{
    Address:       &address,
    NetworkID:     &networkID,
    Band:          &band,
    Parameter:     &params,
    RFOutputPower: &rfPower,
}
```

Apply configuration during connection or update anytime with `SetConfig`. Only non-nil fields will be sent to the radio:

```go
err := lora.SetConfig(config)
```

### Sending Messages

Send up to 240 bytes to another radio. The first argument is the destination radio's address. **Both radios must have the same NetworkID to communicate**:

```go
err := lora.SendMessage(uint16(2), []byte("Hello!"))
```

This sends "Hello!" to the radio with address `2`. The radios must share the same NetworkID (configured in `Configuration.NetworkID`).

### Receiving Messages

Messages are delivered via the `RecievedData` channel. Set up a goroutine to listen:

```go
go func() {
    for msg := range lora.RecievedData {
        data := string(msg.Data[:msg.Length])
        fmt.Printf("From %d: %s (RSSI: %d, SNR: %d)\n", msg.Address, data, msg.ReceivedSignalStrengthIndicator, msg.SignalToNoiseRatio)
    }
}()
```

### Sending Raw AT Commands

For direct AT command access, send to the `Commands` channel:

```go
responseChan := make(chan krylr896.CommandResponse, 1)
lora.Commands <- krylr896.Command{
    Text:         "AT+ADDRESS?",
    ResponseChan: responseChan,
}
```

Read the response:

```go
response := <-responseChan
if response.Error != nil {
    // handle error
}
fmt.Println(response.Response)
```

### Error Handling

The library distinguishes between two types of errors:

**Radio Errors** - Error codes from the LoRa module (e.g., `+ERR=2`):
- Stored in `ErrorEvent.Code` as `*int`
- `nil` if not a radio error

**Go Errors** - Runtime errors from the library:
- Stored in `ErrorEvent.Err` as `error`
- `nil` if not a Go error

Check both fields:

```go
if errEvent.Code != nil {
    fmt.Printf("Radio error code: %d\n", *errEvent.Code)
}
if errEvent.Err != nil {
    fmt.Printf("Go error: %v\n", errEvent.Err)
}
```

### Uncategorized Errors Channel

The `Errors` channel receives:
- Unsolicited error codes from the radio (e.g., spontaneous `+ERR=10`)
- Serial port errors
- Unknown/unrecognizable output that doesn't match expected patterns

Monitor errors in a goroutine:

```go
go func() {
    for errEvent := range lora.Errors {
        if errEvent.Code != nil {
            log.Printf("Unsolicited radio error: %d", *errEvent.Code)
        }
        if errEvent.Err != nil {
            log.Printf("System error: %v", errEvent.Err)
        }
    }
}()
```

Any data received from the radio that doesn't match known patterns (`+OK`, `+ERR=`, `+RCV=`) is sent to this channel as a Go error with the message "unknown unsolicited data".

### Closing the Connection

Always close when finished:

```go
defer lora.CloseConnection()
```

This closes the serial port and command channels gracefully.

## Constants

### Bandwidth
- `Bandwidth125KHz`, `Bandwidth250KHz`, `Bandwidth500KHz` (recommended)
- `Bandwidth7_8KHz`, `Bandwidth10_4KHz` (not recommended)

### Frequency Bands
- `BandUSA` (915 MHz)
- `BandEUROPE1` (868 MHz)
- `BandEUROPE2` (433 MHz)
- `BandCHINA`, `BandASIA`, `BandAUSTRALIA`, `BandINDIA`, `BandKOREA`, `BandBRAZIL`, `BandJAPAN`

### Operating Mode
- `MODE_TRX` - Transmit and receive
- `MODE_SLEEP` - Sleep mode

### UART Baud Rates
- `UartBaudRate_9600`, `UartBaudRate_115200` (default), etc.

### Error Codes
- `OK` (0) - Success
- `NO_ENTER` (1) - Missing `\r\n` after command
- `NO_AT` (2) - Command doesn't start with AT
- `NO_EQ` (3) - Missing `=` in command
- `UNK_CMD` (4) - Unknown command
- `TX_OT` (10) - Transmit timeout
- `RX_OT` (11) - Receive timeout
- `CRC_ERR` (12) - CRC error
- `TX_OR` (13) - Transmit overrun (>240 bytes)
- `UNK_ERR` (15) - Unknown error

## Example

See `lora_test.go` for complete examples including:
- Full radio configuration
- Bidirectional communication
- Error handling
- Raw AT commands

## License

MIT License - See LICENSE.txt
