package ant

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/google/gousb"
	"io"
	"log"
)

const (
	MessageRF = 0x01

	MessageTXSync        = 0xA4
	DefaultNetworkNumber = 0x00

	// Configuration Messages
	MessageChannelUnassign      = 0x41
	MessageChannelAssign        = 0x42
	MessageChannelID            = 0x51
	MessageChannelPeriod        = 0x43
	MessageChannelSearchTimeout = 0x44
	MessageChannelFrequency     = 0x45
	MessageChannelTXPower       = 0x60
	MessageNetworkKey           = 0x46
	MessageTXPower              = 0x47
	MessageProximitySearch      = 0x71
	MessageEnableRXExt          = 0x66
	MessageLibConfig            = 0x6E
	MessageChannelOpenRXScan    = 0x5B

	// Notification Messages
	MessageStartup = 0x6F

	// Control Messages
	MessageSystemReset    = 0x4A
	MessageChannelOpen    = 0x4B
	MessageChannelClose   = 0x4C
	MessageChannelRequest = 0x4D

	// Data Messages
	MessageChannelBroadcastData    = 0x4E
	MessageChannelAcknowledgedData = 0x4F
	MessageChannelBurstData        = 0x50

	// Channel Event Messages
	MessageChannelEvent = 0x40

	// Requested response messages
	MessageChannelStatus = 0x52
	MessageVersion       = 0x3E
	MessageCapabilities  = 0x54
	MessageSerialNumber  = 0x61

	// Message Parameters
	ChannelTypeTwoWayReceive             = 0x00
	ChannelTypeTwoWayTransmit            = 0x10
	ChannelTypeSharedReceive             = 0x20
	ChannelTypeSharedTransmit            = 0x30
	ChannelTypeOneWayReceive             = 0x40
	ChannelTypeOneWayTransmit            = 0x50
	RadioTXPowerMinus20DB                = 0x00
	RadioTXPowerMinus10DB                = 0x01
	RadioTXPower0DB                      = 0x02
	RadioTXPowerPlus4DB                  = 0x03
	ResponseNoError                      = 0x00
	EventRXSearchTimeout                 = 0x01
	EventRXFailed                        = 0x02
	EventTX                              = 0x03
	EventTransferRXFailed                = 0x04
	EventTransferTXCompleted             = 0x05
	EventTransferTXFailed                = 0x06
	EventChannelClosed                   = 0x07
	EventRXFailGoToSearch                = 0x08
	EventChannelCollision                = 0x09
	EventTransferTXStart                 = 0x0A
	ChannelInWrongState                  = 0x15
	ChannelNotOpened                     = 0x16
	ChannelIDNotSet                      = 0x18
	CloseAllChannels                     = 0x19
	TransferInProgress                   = 0x1F
	TransferSequenceNumberError          = 0x20
	TransferInError                      = 0x21
	MessageSizeExceedsLimit              = 0x27
	InvalidMessage                       = 0x28
	InvalidNetworkNumber                 = 0x29
	InvalidListID                        = 0x30
	InvalidScanTXChannel                 = 0x31
	InvalidParameterProvided             = 0x33
	EventQueueOverflow                   = 0x35
	USBStringWriteFail                   = 0x70
	ChannelStateUnassigned               = 0x00
	ChannelStateAssigned                 = 0x01
	ChannelStateSearching                = 0x02
	ChannelStateTracking                 = 0x03
	CapabilitiesNoReceiveChannels        = 0x01
	CapabilitiesNoTransmitChannels       = 0x02
	CapabilitiesNoReceiveMessages        = 0x04
	CapabilitiesNoTransmitMessages       = 0x08
	CapabilitiesNoAcknowledgedMessages   = 0x10
	CapabilitiesNoBurstMessages          = 0x20
	CapabilitiesNetworkEnabled           = 0x02
	CapabilitiesSerialNumberEnabled      = 0x08
	CapabilitiesPerChannelTXPowerEnabled = 0x10
	CapabilitiesLowPrioritySearchEnabled = 0x20
	CapabilitiesScriptEnabled            = 0x40
	CapabilitiesSearchListEnabled        = 0x80
	CapabilitiesLEDEnabled               = 0x01
	CapabilitiesExtMessageEnabled        = 0x02
	CapabilitiesScanModeEnabled          = 0x04
	CapabilitiesProxSearchEnabled        = 0x10
	CapabilitiesExtAssignEnabled         = 0x20
	CapabilitiesFSANTFSEnabled           = 0x40
	TimeoutNever                         = 0xFF

	// Message parameters
	BufferIndexMessageLength   = 1
	BufferIndexMessageType     = 2
	BufferIndexChannelNumber   = 3
	BufferIndexMessageData     = 4
	BufferIndexExtMessageBegin = 12
)

func resetSystem() []byte {
	payload := []byte{}
	payload = append(payload, 0x00)
	return buildMessage(payload, MessageSystemReset)
}

func requestMessage(channel uint32, messageID byte) []byte {
	payload := intToLEHexArray(channel, 1)
	payload = append(payload, messageID)
	return buildMessage(payload, MessageChannelRequest)
}

func buildMessage(payload []byte, msgID byte) []byte {
	m := []byte{}
	m = append(m, MessageTXSync)
	m = append(m, byte(len(payload)))
	m = append(m, msgID)
	for _, b := range payload {
		m = append(m, b)
	}
	m = append(m, byte(getChecksum(m)))
	return m
}

func intToLEHexArray(num uint32, numBytes int) []byte {
	var buf bytes.Buffer
	if numBytes <= 0 || numBytes > 4 {
		panic("runtime error: numBytes to get from intToLEHexArray must be [1,4]")
	}
	err := binary.Write(&buf, binary.LittleEndian, num)
	if err != nil {
		log.Fatalf("intToLEHexArray unable to writer to buffer")
	}
	bs := buf.Bytes()
	return bs[:numBytes]
}

func getChecksum(message []byte) int {
	var checksum int
	for _, b := range message {
		checksum = (checksum ^ int(b)) % 0xFF
	}
	return checksum
}

func setNetworkKey() []byte {
	payload := []byte{}
	payload = append(payload, DefaultNetworkNumber)
	payload = append(payload, 0xB9)
	payload = append(payload, 0xA5)
	payload = append(payload, 0x21)
	payload = append(payload, 0xFB)
	payload = append(payload, 0xBD)
	payload = append(payload, 0x72)
	payload = append(payload, 0xC3)
	payload = append(payload, 0x45)
	return buildMessage(payload, MessageNetworkKey)
}

func setDevice(channel, deviceID, deviceType, transmissionType uint32) []byte {
	payload := []byte{}
	bs := intToLEHexArray(channel, 1)
	payload = append(payload, bs...)
	bs = intToLEHexArray(deviceID, 2)
	payload = append(payload, bs...)
	bs = intToLEHexArray(deviceType, 1)
	payload = append(payload, bs...)
	bs = intToLEHexArray(transmissionType, 1)
	payload = append(payload, bs...)
	return buildMessage(payload, MessageChannelID)
}

func searchChannel(channel, timeout uint32) []byte {
	payload := []byte{}
	payload = append(payload, intToLEHexArray(channel, 1)...)
	payload = append(payload, intToLEHexArray(timeout, 1)...)
	return buildMessage(payload, MessageChannelSearchTimeout)
}

func setPeriod(channel, period uint32) []byte {
	payload := []byte{}
	payload = append(payload, intToLEHexArray(channel, 1)...)
	payload = append(payload, intToLEHexArray(period, 1)...)
	return buildMessage(payload, MessageChannelPeriod)
}

func setFrequency(channel, frequency uint32) []byte {
	payload := []byte{}
	payload = append(payload, intToLEHexArray(channel, 1)...)
	payload = append(payload, intToLEHexArray(frequency, 1)...)
	return buildMessage(payload, MessageChannelFrequency)
}

func setRxExt() []byte {
	payload := []byte{}
	payload = append(payload, intToLEHexArray(0, 1)...)
	payload = append(payload, intToLEHexArray(1, 1)...)
	return buildMessage(payload, MessageEnableRXExt)
}

func libConfig(channel, how uint32) []byte {
	payload := []byte{}
	payload = append(payload, intToLEHexArray(channel, 1)...)
	payload = append(payload, intToLEHexArray(how, 1)...)
	return buildMessage(payload, MessageLibConfig)
}

func openRXScan() []byte {
	payload := []byte{}
	payload = append(payload, intToLEHexArray(0, 1)...)
	payload = append(payload, intToLEHexArray(1, 1)...)
	return buildMessage(payload, MessageChannelOpenRXScan)
}

func assignChannel(channel uint32, channelType string) []byte {
	payload := []byte{}
	bs := intToLEHexArray(channel, 1)
	payload = append(payload, bs...)
	switch channelType {
	case "receive":
		payload = append(payload, ChannelTypeTwoWayReceive)
	case "receive_only":
		payload = append(payload, ChannelTypeOneWayReceive)
	case "receive_shared":
		payload = append(payload, ChannelTypeSharedReceive)
	case "transmit":
		payload = append(payload, ChannelTypeTwoWayTransmit)
	case "transmit_only":
		payload = append(payload, ChannelTypeOneWayTransmit)
	case "transmit_shared":
		payload = append(payload, ChannelTypeSharedTransmit)
	default:
		panic(fmt.Sprintf("runtime error: invalid channel type %d in assignChannel", channelType))
	}
	payload = append(payload, DefaultNetworkNumber)
	return buildMessage(payload, MessageChannelAssign)
} 

func unassignChannel(channel uint32) []byte {
	payload := intToLEHexArray(channel, 1)
	return buildMessage(payload, MessageChannelUnassign)
}
 
func openChannel(channel uint32) []byte {
	payload := intToLEHexArray(uint32(channel), 1)
	return buildMessage(payload, MessageChannelOpen)
}

func closeChannel(channel uint32) []byte {
	payload := intToLEHexArray(uint32(channel), 1)
	return buildMessage(payload, MessageChannelClose)
}

func acknowledgedData(channel uint32, data []byte) []byte {
	payload := intToLEHexArray(uint32(channel), 1)
	payload = append(payload, data...)
	return buildMessage(payload, MessageChannelAcknowledgedData)
}

func broadcastData(channel uint32, data []byte) []byte {
	payload := intToLEHexArray(uint32(channel), 1)
	payload = append(payload, data...)
	return buildMessage(payload, MessageChannelBroadcastData)
}

var deviceInUse []*gousb.Device = []*gousb.Device{}

func checkDeviceInUse(desc *gousb.DeviceDesc) bool {
	for _, device := range deviceInUse {
		if device.Desc == desc {
			return true
		}
	}
	return false
}

type Driver interface {
	attach(*BaseSensor, bool) bool
	detach(*BaseSensor) bool
	canScan() bool
	Open(*gousb.Context) error
	Close()
	isScanning() bool
	write([]byte) error
}

type USBDriver struct {
	device               *gousb.Device
	intf                 *gousb.Interface
	intfDone             func()
	detachedKernelDriver bool
	inEp                 *gousb.InEndpoint
	inEpReader			 *gousb.ReadStream
	outEp                *gousb.OutEndpoint
	leftOver             []byte 
	usedChannels         int
	attachedSensors      []*BaseSensor
	vendorID             gousb.ID
	productID            gousb.ID
	startupCallbacks     []func()
	DoneReading			 chan bool

	MaxChannels int
	CanScan     bool
}

func NewUSBDriver(vendorID, productID gousb.ID) *USBDriver {
	return &USBDriver{
		vendorID:  vendorID,
		productID: productID,
		DoneReading: make(chan bool),
	}
}

func (drv *USBDriver) canScan() bool {
	return drv.CanScan
}

func (drv *USBDriver) OnStartup(fn func()) {
	drv.startupCallbacks = append(drv.startupCallbacks, fn)
}

func (drv *USBDriver) Open(ctx *gousb.Context) error {
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == drv.vendorID &&
			desc.Product == drv.productID &&
			!checkDeviceInUse(desc)
	})
	if err != nil {
		return err
	}
	drv.device = devs[0]
	// close any other device that may have been open
	for idx, dev := range devs {
		// dont close the device we are going to use
		// we'll need to close it later when we clean up
		if idx != 0 {
			dev.Close()
		}
	}
	drv.device.SetAutoDetach(true)
	drv.intf, drv.intfDone, err = drv.device.DefaultInterface()
	if err != nil {
		drv.device.Close()
		drv.device = nil
		return err
	}
	deviceInUse = append(deviceInUse, devs[0])
	drv.inEp, err = drv.intf.InEndpoint(1)
	if err != nil {
		log.Println("couldnt get inep 0")
		return err
	}

	maxPacketSize := drv.inEp.Desc.MaxPacketSize
	drv.inEpReader, err = drv.inEp.NewStream(maxPacketSize,3)
	if err != nil {
		log.Fatalf("couldn't get stream reader for in endpoint")
	}

	drv.outEp, err = drv.intf.OutEndpoint(1)
	if err != nil {
		log.Println("couldnt get outep 1")
		return err
	}

	drv.reset()
	
	go func(){	
		defer func(){
			// if we panic in this routine make sure
			// everything gets closed properly
			if a := recover(); a != nil {
				drv.Close()
			}
		}()
		data := make([]byte, drv.inEp.Desc.MaxPacketSize)
		for {
			numBytes, err := drv.inEpReader.Read(data)
			if err != nil {
				if err == io.EOF {
					log.Println("inEpReader closed")
				}
				break
			}


			if numBytes == 0 {
				continue
			}


			if len(drv.leftOver) > 0 {
				data = append(drv.leftOver, data...)
				drv.leftOver = []byte{}
			}

			if data[0] != MessageTXSync {
				log.Fatalf("sync byte missing from stream")
			}

			l := numBytes
			beginBlock := 0
			for beginBlock < l {
				if beginBlock + 1 == l {
					drv.leftOver = data[beginBlock:]
					break
				}
				blockLen := data[beginBlock + 1]
				endBlock := beginBlock + int(blockLen) + 4
				if endBlock > l {
					drv.leftOver = data[beginBlock:]
					break
				}
				readData := data[beginBlock:endBlock]
				drv.read(readData)
				beginBlock = endBlock
			}
		}
		drv.DoneReading <- true
	}()

	return nil
}


func (drv *USBDriver) write(data []byte) error {
	fmt.Printf("Writing: % X\n", data)
	_, err := drv.outEp.Write(data)
	return err
}

func (drv *USBDriver) read(data []byte) {
	messageID := data[2]
	switch {
	case messageID == MessageStartup:
		request := requestMessage(0, MessageCapabilities)
		drv.write(request)
	case messageID == MessageCapabilities:
		drv.MaxChannels = int(data[3])
		drv.CanScan = (data[7] & 0x06) == 0x06
		drv.write(setNetworkKey())
	case messageID == MessageChannelEvent && data[4] == MessageNetworkKey:
		for _, cb := range drv.startupCallbacks {
			cb()
		}
	default:
		for _, sensor := range drv.attachedSensors {
			sensor.handleEventMessages(data)
		}
	}
}

func (drv *USBDriver) attach(sensor *BaseSensor, forScan bool) bool {
	if drv.usedChannels < 0 {
		return false
	}
	if forScan {
		if drv.usedChannels != 0 {
			return false
		}
		drv.usedChannels = -1
	} else {
		if drv.MaxChannels <= drv.usedChannels {
			return false
		}
		drv.usedChannels++
	}
	drv.attachedSensors = append(drv.attachedSensors, sensor)
	return true
}

func (drv *USBDriver) detach(sensor *BaseSensor) bool {
	idx := -1
	for i, s := range drv.attachedSensors {
		if s == sensor {
			idx = i
		}
	}
	if idx < 0 {
		return false
	}
	if drv.usedChannels < 0 {
		drv.usedChannels = 0
	} else {
		drv.usedChannels--
	}
	drv.attachedSensors[idx], drv.attachedSensors[len(drv.attachedSensors)-1] =
		drv.attachedSensors[len(drv.attachedSensors)-1], drv.attachedSensors[idx]
	drv.attachedSensors = drv.attachedSensors[:len(drv.attachedSensors)-1]
	return true
}

func (drv *USBDriver) detachAll() {
	for _, sensor := range drv.attachedSensors {
		sensor.detach()
	}
}

func (drv *USBDriver) Close() {
	drv.detachAll()
	drv.inEpReader.Close()
	drv.inEpReader = nil
	drv.intfDone()
	drv.device.Close()
	drv.device = nil
	drv.intf = nil
	drv.intfDone = nil
}

func (drv *USBDriver) reset() {
	drv.detachAll()
	drv.MaxChannels = 0
	drv.usedChannels = 0
	drv.write(resetSystem())
}

func (drv *USBDriver) isScanning() bool {
	return drv.usedChannels == -1
}

func (drv *USBDriver) getDevices(ctx gousb.Context) []*gousb.Device {
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == drv.vendorID && desc.Product == drv.productID
	})
	if err != nil {
		log.Fatalf("OpenDevices(): %v", err)
	}
	return devs
}

type GarminStick2 struct {
	USBDriver
}

func NewGarminStick2() *GarminStick2 {
	return &GarminStick2{
		USBDriver: USBDriver{
			vendorID:  0x0FCF,
			productID: 0x1008,
		},
	}
}

type GarminStick3 struct {
	USBDriver
}

func NewGarminStick3() *GarminStick3 {
	return &GarminStick3{
		USBDriver: USBDriver{
			vendorID:  0x0FCF,
			productID: 0x1009,
		},
	}
}

type BaseSensor struct {
	channel            *uint32
	deviceID           uint32
	transmissionType   uint32
	driver             Driver
	messageQueue	   []Message
	decodeDataCallback func([]byte)
	statusCallback     func(byte, byte) bool
}

type Sensor interface {
	updateState(uint32, []byte)
}

type SendCallback func(bool)
type Message struct {
	msg		 []byte
	callback SendCallback
}

// need to get a callback to the driver somehow
func NewBaseSensor(driver Driver) *BaseSensor {
	return &BaseSensor{
		driver:  driver,
	}
}

func (sensor *BaseSensor) scan(channelType string, frequency uint32) error {
	if sensor.channel != nil {
		return errors.New("sensor already attached")
	}

	if !sensor.driver.canScan() {
		return errors.New("usb stick cannot scan")
	}

	var channel uint32 = 0

	onStatus := func(msg, code byte) bool {
		switch msg {
		case MessageRF:
			switch code {
			case EventChannelClosed, EventRXFailGoToSearch:
				sensor.write(unassignChannel(channel))
				return true
			case EventTransferTXCompleted, EventTransferTXFailed,
				EventRXFailed, InvalidScanTXChannel:
				if len(sensor.messageQueue) < 0 {
					return true
				}
				message := sensor.messageQueue[0]
				if message.callback != nil {
					message.callback(code == EventTransferTXCompleted)
				}
				if len(sensor.messageQueue) == 1 {
					sensor.messageQueue = []Message{}
					return true
				}
				sensor.messageQueue = sensor.messageQueue[1:]
				sensor.write(sensor.messageQueue[0].msg)
				return true
			}
		case MessageChannelAssign:
			sensor.write(setDevice(channel, 0, 0, 0))
			return true
		case MessageChannelID:
			sensor.write(setFrequency(channel, frequency))
			return true
		case MessageChannelFrequency:
			sensor.write(setRxExt())
			return true
		case MessageEnableRXExt:
			sensor.write(libConfig(channel, 0xE0))
			return true
		case MessageLibConfig:
			sensor.write(openRXScan())
			return true
		case MessageChannelOpenRXScan:
			//TODO emit attached event
			return true
		case MessageChannelClose:
			return true
		case MessageChannelUnassign:
			sensor.statusCallback = nil
			sensor.channel = nil
			//TODO emit detached event
			return true
		case MessageChannelAcknowledgedData:
			return code == TransferInProgress
		}
		return false
	}

	if sensor.driver.isScanning() {
		sensor.channel = &channel
		sensor.deviceID = 0
		sensor.transmissionType = 0
		sensor.statusCallback = onStatus
		//TODO "emit" an attach event
	} else if sensor.driver.attach(sensor, true) {
		sensor.channel = &channel
		sensor.deviceID = 0
		sensor.transmissionType = 0
		sensor.statusCallback = onStatus
		sensor.write(assignChannel(channel, channelType))
	} else {
		return errors.New("cannot attach sensor")
	}
	return nil
}

func (sensor *BaseSensor) attach(channel, deviceID, deviceType, timeout, period,
			frequency, transmissionType uint32, channelType string) error {
	if sensor.channel != nil { 
		return errors.New("sensor already attached")	
	}
	if !sensor.driver.attach(sensor, false) {
		return errors.New("driver can not attach sensor")
	}
	sensor.channel = &channel
	sensor.deviceID = deviceID
	sensor.transmissionType = transmissionType

	onStatus := func(msg, code byte) bool {
		switch msg {
		case MessageRF:
			switch code {
			case EventChannelClosed, EventRXFailGoToSearch:
				sensor.write(unassignChannel(channel))
				return true
			case EventTransferTXCompleted, EventTransferTXFailed,
				EventRXFailed, InvalidScanTXChannel:
				if len(sensor.messageQueue) < 0 {
					return true
				}
				message := sensor.messageQueue[0]
				if message.callback != nil {
					message.callback(code == EventTransferTXCompleted)
				}
				if len(sensor.messageQueue) == 1 {
					sensor.messageQueue = []Message{}
					return true
				}
				sensor.messageQueue = sensor.messageQueue[1:]
				sensor.write(sensor.messageQueue[0].msg)
				return true
			}
		case MessageChannelAssign:
			sensor.write(setDevice(channel, deviceID, deviceType, transmissionType))
			return true
		case MessageChannelID:
			sensor.write(searchChannel(channel, timeout))
			return true
		case MessageChannelSearchTimeout:
			sensor.write(setFrequency(channel, frequency))
			return true
		case MessageChannelFrequency:
			sensor.write(setPeriod(channel, period))
			return true
		case MessageChannelPeriod:
			sensor.write(libConfig(channel, 0xE0))
			return true
		case MessageLibConfig:
			sensor.write(openChannel(channel))
			return true
		case MessageChannelOpen:
			//TODO emit attached event
			return true
		case MessageChannelClose:
			return true
		case MessageChannelUnassign:
			sensor.statusCallback = nil
			sensor.channel = nil
			//TODO emit detached event
			return true
		case MessageChannelAcknowledgedData:
			return code == TransferInProgress
		}
		return false
	}

	sensor.statusCallback = onStatus
	sensor.write(assignChannel(channel, channelType))
	return nil
}

func (sensor *BaseSensor) detach() {
	//TODO do.
	if sensor.channel == nil {
		return
	}
	// write a close message on the channel
	sensor.write(closeChannel(*sensor.channel))
	sensor.driver.detach(sensor)
}

func (sensor *BaseSensor) handleEventMessages(data []byte) {
	messageID := data[BufferIndexMessageType]
	channel := data[BufferIndexChannelNumber]

	if channel == byte(*sensor.channel) {
		if messageID == MessageChannelEvent {
			msg := data[BufferIndexMessageData]
			code := data[BufferIndexMessageData + 1]

			handled := sensor.statusCallback != nil && sensor.statusCallback(msg, code)
			if !handled {
				log.Println("Unhandled event: ", data)
				//TODO emit an eventData event with message and code
			}
		} else if sensor.decodeDataCallback != nil {
			sensor.decodeDataCallback(data)
		}
	}
}

func (sensor *BaseSensor) send(msg Message) {
	sensor.messageQueue = append(sensor.messageQueue, msg)
	if len(sensor.messageQueue) == 1 {
		sensor.write(msg.msg)
	}
}

func (sensor *BaseSensor) write(data []byte) {
	sensor.driver.write(data)
}

type AntPlusBaseSensor struct {
	BaseSensor
	Sensor
}

func (sensor *AntPlusBaseSensor) scan(scanType string) {
	sensor.BaseSensor.scan(scanType, 57)
}

func (sensor *AntPlusBaseSensor) attach(channel, deviceID, deviceType,
	transmissionType, timeout, period uint32, channelType string) {
	sensor.BaseSensor.attach(channel, deviceID, deviceType, transmissionType, 
		timeout, period, 57, channelType)
}

type AntPlusSensor struct {
	AntPlusBaseSensor
}

func NewAntPlusSensor(driver Driver, sensor Sensor) *AntPlusSensor {
	s := AntPlusSensor {
		AntPlusBaseSensor: AntPlusBaseSensor {
			BaseSensor: BaseSensor {
				driver: driver,
			},
			Sensor: sensor,
		},
	}
	s.decodeDataCallback = s.decodeData
	return &s
}

func (sensor *AntPlusSensor) scan() {
	panic("AntPlusSensor does not support scanning")
}

func (sensor *AntPlusSensor) attach(channel, deviceID, deviceType, transmissionType,
	timeout, period uint32, channelType string) {
	sensor.AntPlusBaseSensor.attach(channel, deviceID, deviceType,
		transmissionType, timeout, period, channelType)
}

func (sensor *AntPlusSensor) decodeData(data []byte) {
	switch data[BufferIndexMessageType] {
	case MessageChannelBroadcastData, MessageChannelAcknowledgedData,
		MessageChannelBurstData:
		if sensor.deviceID == 0 {
			sensor.write(requestMessage(*sensor.channel, MessageChannelID))
		}
		sensor.updateState(sensor.deviceID, data)
	case MessageChannelID:
		sensor.deviceID = uint32(data[BufferIndexMessageData])
		sensor.transmissionType = uint32(data[BufferIndexMessageData+3])
	}
}

type AntPlusScanner struct {
	AntPlusBaseSensor
	Scanner
}

type Scanner interface {
	deviceType() uint32
	createStateIfNew(uint32)
	updateRssiAndThreshold(uint32, uint32, uint32)
	updateState(uint32, []byte)
}

func NewAntPlusScanner(driver Driver, scanner Scanner) *AntPlusScanner {
	apScanner := AntPlusScanner {
		AntPlusBaseSensor: AntPlusBaseSensor {
			BaseSensor: BaseSensor {
				driver: driver,
			},
		},
		Scanner: scanner,
	}
	apScanner.decodeDataCallback = apScanner.decodeData
	return &apScanner
}

func (scanner *AntPlusScanner) Scan() {
	scanner.AntPlusBaseSensor.scan("receive")
}

func (scanner *AntPlusScanner) attach() {
	panic("AntPlusScanner: attach not supported")
}

func (scanner *AntPlusScanner) send() {
	panic("AntPlusScanner: send not supported")
}

func (scanner *AntPlusScanner) decodeData(data []byte) {
	if len(data) <= (BufferIndexExtMessageBegin+3) || 
		(data[BufferIndexExtMessageBegin] & 0x80)  == 0 {
			log.Println("wrong message format: ", data)
			return
	}	

	idOffset := BufferIndexExtMessageBegin+1
	deviceID := uint32(binary.LittleEndian.Uint16(data[idOffset:idOffset+2]))
	deviceType := uint32(data[BufferIndexExtMessageBegin+3])

	if deviceType != scanner.deviceType() {
		return
	}

	scanner.createStateIfNew(deviceID)

	if data[BufferIndexExtMessageBegin] & 0x40 != 0 {
		if data[BufferIndexExtMessageBegin+5] == 0x20 {
			scanner.updateRssiAndThreshold(
				deviceID,
				uint32(data[BufferIndexExtMessageBegin+6]),
				uint32(data[BufferIndexExtMessageBegin+7]))
		}
	}

	switch data[BufferIndexMessageType] {
		case MessageChannelBroadcastData, MessageChannelAcknowledgedData,
			MessageChannelBurstData:
			scanner.updateState(deviceID, data)
	}
}
