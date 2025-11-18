package krylr896

// bandwidth constants
const (
	Bandwidth7_8KHz   uint8 = 0 // NOT RECCOMENDED
	Bandwidth10_4KHz  uint8 = 1 // NOT RECCOMENDED
	Bandwidth15_6KHz  uint8 = 2
	Bandwidth20_8KHz  uint8 = 3
	Bandwidth31_25KHz uint8 = 4
	Bandwidth41_7KHz  uint8 = 5
	Bandwidth62_5KHz  uint8 = 6
	Bandwidth125KHz   uint8 = 7
	Bandwidth250KHz   uint8 = 8
	Bandwidth500KHz   uint8 = 9
)

// band constants, HZ
const (
	BandUSA       uint32 = 915000000
	BandEUROPE1   uint32 = 868000000
	BandEUROPE2   uint32 = 433000000
	BandCHINA     uint32 = 470000000
	BandASIA      uint32 = 470000000
	BandAUSTRALIA uint32 = 923000000
	BandINDIA     uint32 = 865000000
	BandKOREA     uint32 = 920000000
	BandBRAZIL    uint32 = 915000000
	BandJAPAN     uint32 = 920000000
)

// mode constants
const (
	MODE_TRX   uint8 = 0 // transmit and recieve
	MODE_SLEEP uint8 = 1 // sleep
)

// UART baud rate constants
const (
	UartBaudRate_300    int = 300
	UartBaudRate_1200   int = 1200
	UartBaudRate_4800   int = 4800
	UartBaudRate_9600   int = 9600
	UartBaudRate_28800  int = 19200
	UartBaudRate_38400  int = 38400
	UartBaudRate_57600  int = 57600
	UartBaudRate_115200 int = 115200 // default
)

// result code constants
const (
	OK       = 0  // OK
	NO_ENTER = 1  // missing "\r\n" after command
	NO_AT    = 2  // head of command is not AT
	NO_EQ    = 3  // missing "=" in AT command
	UNK_CMD  = 4  // unknown command
	TX_OT    = 10 // transmit over time
	RX_OT    = 11 // receive over time
	CRC_ERR  = 12 // CRC error
	TX_OR    = 13 // transmit over run(over 240 bytes)
	UNK_ERR  = 15 // unknown error
)
