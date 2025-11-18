package krylr896

// data schema and constant definitions for the radio

//
// Error and Command structures
//

type ErrorEvent struct {
	Code *int  // LoRa error code (nil if not applicable)
	Err  error // Go error (nil if not applicable)
}

type Command struct {
	Text         string
	ResponseChan chan CommandResponse // channel to receive response on
}

type CommandResponse struct {
	Response string
	Error    *ErrorEvent // nil if no error
}

//
// radio setup information
//

// complete configuration
type Configuration struct {
	Address       *uint16     // ADDRESS, 		0-65535,															ident of the transciever
	NetworkID     *uint8      // NETWORKID, 	0-16,																must be the same for radios to communicate
	Band          *uint32     // BAND(Hz), 		433000000-915000000,												center freq of wireless band
	Parameter     *Parameters // PARAMETER, 																		rf params
	Mode          *uint8      // MODE, 			0-1,																work mode
	UartBaudRate  *int        // IPR, 			300-115200, 														uart baud rate
	EncryptionKey *[16]byte   // CPIN, 			00000000000000000000000000000000-FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF,	AES128 network password
	RFOutputPower *uint8      // CRFOP(dBm),	0-15,																RF output power

}

// rf transmission params,
type Parameters struct {
	SpreadingFactor    uint8 // SF, 7-12
	Bandwidth          uint8 // BW, 0-9
	CodingRate         uint8 // CR, 1-4
	ProgrammedPreamble uint8 // PP, 4-7
}

//
// Response Structures
//

type RecievedData struct {
	Address                         uint16    // transmitter address
	Length                          uint8     // data length
	Data                            [240]byte // data
	ReceivedSignalStrengthIndicator int8      // RSSI(dBm)
	SignalToNoiseRatio              int8      // SNR
}
