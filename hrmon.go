package ant

import (
	"encoding/binary"
)


const (
	InitPage PageState = 0
	StdPage PageState = 1
	ExtPage PageState = 2

	ToggleMask byte = 0x80
)

type HeartRateSensorState struct {
	DeviceID          uint32
	BeatTime          uint16
	BeatCount         byte
	ComputedHeartRate byte
	OperatingTime	  uint32
	ManID             byte
	SerialNumber      uint32
	HWVersion         byte
	SWVersion		  byte
	ModelNumber       byte
	PreviousBeat      uint16
	IntervalAverage   byte
	IntervalMax       byte
	SessionAverage    byte
	SupportedFeatures byte
	EnabledFeatures   byte
	BatteryLevel      byte
	BatteryVoltage    float32
	BatteryStatus     string
}

func (s *HeartRateSensorState) update(page *Page, data []byte) {
	pageNumber := data[BufferIndexMessageData]
	if page.pageState == InitPage {
		page.pageState = StdPage
	} else if pageNumber != page.oldPage || page.pageState == ExtPage {
		page.pageState = ExtPage
		switch pageNumber & ^ToggleMask {
			case 1:
				s.OperatingTime = uint32(data[BufferIndexMessageData + 1])
				s.OperatingTime |= uint32(data[BufferIndexMessageData + 2]) << 8
				s.OperatingTime |= uint32(data[BufferIndexMessageData + 3]) << 16
				s.OperatingTime *= 2
			case 2:
				s.ManID = data[BufferIndexMessageData+1]
				s.SerialNumber = uint32(s.DeviceID)
				s.SerialNumber |= uint32(binary.LittleEndian.Uint16(data[BufferIndexMessageData+2:BufferIndexMessageData+4])) << 16
				s.SerialNumber ^= 0x80000000
			case 3:
				s.HWVersion = data[BufferIndexMessageData+1]
				s.SWVersion = data[BufferIndexMessageData+2]
				s.ModelNumber = data[BufferIndexMessageData+3]
			case 4:
				s.PreviousBeat = binary.LittleEndian.Uint16(data[BufferIndexMessageData+2:BufferIndexMessageData+4])
			case 5:
				s.IntervalAverage = data[BufferIndexMessageData+1]
				s.IntervalMax = data[BufferIndexMessageData+2]
				s.SessionAverage = data[BufferIndexMessageData+3]
			case 6:
				s.SupportedFeatures = data[BufferIndexMessageData+2]
				s.EnabledFeatures = data[BufferIndexMessageData+3]
			case 7:
				batteryLevel := data[BufferIndexMessageData+1]
				batteryFrac := float32(data[BufferIndexMessageData+2])
				batteryStatus := data[BufferIndexMessageData+3]
				if batteryLevel != 0xFF {
					s.BatteryLevel = batteryLevel
				}
				s.BatteryVoltage = float32(batteryStatus & 0x0F) + (batteryFrac / 256.0)
				batteryFlags := (batteryStatus & 0x70) >> 4
				batteryFlags ^= 0x80
				switch batteryFlags {
				case 1:
					s.BatteryStatus = "New"
				case 2:
					s.BatteryStatus = "Good"
				case 3:
					s.BatteryStatus = "Ok"
				case 4:
					s.BatteryStatus = "Low"
				case 5:
					s.BatteryStatus = "Critical"
				default:
					s.BatteryVoltage = 0
					s.BatteryStatus = "Invalid"
				}
		}
	}
	hrOffset := BufferIndexMessageData+4
	s.BeatTime = binary.LittleEndian.Uint16(data[hrOffset:hrOffset+2])
	s.BeatCount = data[hrOffset+2]
	s.ComputedHeartRate = data[hrOffset+3]
	page.oldPage = pageNumber
}

type HeartRateScannerState struct {
	*HeartRateSensorState
	RSSI      uint32
	Threshold uint32
}

type HeartRateSensor struct {
//TODO do
	*AntPlusSensor
	state *HeartRateSensorState
	page *Page
	listeners []func(*HeartRateSensorState)
}

func NewHeartRateSensor(driver Driver) *HeartRateSensor {
	hrs := HeartRateSensor{
		state: &HeartRateSensorState{},
		page: &Page{oldPage: 1 << 8 - 1, pageState: InitPage},
	}
	hrs.AntPlusSensor = NewAntPlusSensor(driver, &hrs)
	return &hrs
}

func (sensor *HeartRateSensor) updateState(deviceID uint32, data []byte) {
	sensor.state.update(sensor.page, data)
	for _, cb := range sensor.listeners {
		cb(sensor.state)
	}
}

func (sensor *HeartRateSensor) ListenForData(cb func(*HeartRateSensorState)) {
	sensor.listeners = append(sensor.listeners, cb)
}


type HeartRateScanner struct {
	*AntPlusScanner
	states map[uint32]*HeartRateScannerState
	pages map[uint32]*Page
	listeners []func(*HeartRateScannerState)
}

func NewHeartRateScannerState(deviceID uint32) *HeartRateScannerState {
	return &HeartRateScannerState{
		HeartRateSensorState: &HeartRateSensorState{
			DeviceID: deviceID,
		},
	}
}

type PageState uint32

type Page struct {
	oldPage byte
	pageState PageState
}

func NewHeartRateScanner(driver Driver) *HeartRateScanner {
	hrs := HeartRateScanner{
		states: make(map[uint32]*HeartRateScannerState),
		pages: make(map[uint32]*Page),
	}
	hrs.AntPlusScanner = NewAntPlusScanner(driver, &hrs)
	return &hrs
}

func (s *HeartRateScanner) deviceType() uint32 {
	return 120
}

func (s *HeartRateScanner) createStateIfNew(deviceID uint32) {
	if _, ok := s.states[deviceID]; !ok {
		s.states[deviceID] = NewHeartRateScannerState(deviceID)
	}
	if _, ok := s.pages[deviceID]; !ok {
		s.pages[deviceID] = &Page{oldPage: 1 << 8 -1, pageState: InitPage}
	}
}

func (s *HeartRateScanner) ListenForData(cb func(*HeartRateScannerState)) {
	s.listeners = append(s.listeners, cb)
}

func (s *HeartRateScanner) updateRssiAndThreshold(deviceID, rssi, threshold uint32) {
	s.states[deviceID].RSSI = rssi
	s.states[deviceID].Threshold = threshold
}

func (s *HeartRateScanner) updateState(deviceID uint32, data []byte) {
	s.states[deviceID].update(s.pages[deviceID], data)
	for _, cb := range s.listeners {
		cb(s.states[deviceID])
	}
}
