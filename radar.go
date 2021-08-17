package ant

import (
	"encoding/binary"
)

type Target struct {
	ThreatLevel byte
	ThreatSide byte
	Range float32
	Speed float32
}

type BikeRadarSensorState struct {
	DeviceID          uint32
	OperatingTime     uint32
	ManID             uint16
	SerialNumber      uint32
	HWVersion         byte
	SWVersion         float32
	ModelNumber       uint16
	BatteryLevel      byte
	BatteryVoltage    float32
	BatteryStatus     string
	DeviceStatus      string
	ErrorLevel        string
	ErrorDesc         string
	ErrorComponent    byte
	ManufacturerError uint32
	Targets           [8]*Target
}

func (s *BikeRadarSensorState) update(page *Page, data []byte) {
	pageNumber := data[BufferIndexMessageData]
	if page.pageState == InitPage {
		page.pageState = StdPage
	} else if pageNumber != page.oldPage || page.pageState == ExtPage {
		page.pageState = ExtPage
		switch pageNumber & ^ToggleMask {
		case 0x01: // Main Data Page 1 - Device Status
			masked := data[BufferIndexMessageData+1] & 0x03
			if masked == 1 {
				s.DeviceStatus = "Shutdown"
			} else {
				s.DeviceStatus = "Aborting Shutdown"
			}
		case 0x30, 0x31: // Data Page 48 - Radar Targets A
			rangeData := binary.BigEndian.Uint32(data[BufferIndexMessageData+2:BufferIndexMessageData+6]) & 0x00FFFFFF
			for i := 0; i < 4; i++ {
				threatLevel := data[BufferIndexMessageData+1] >> byte(2*i) & 0x03
				if threatLevel == 0 {
					s.Targets[i] = nil
					continue
				}
				threatSide := data[BufferIndexMessageData+2] >> byte(2*i) & 0x03
				target := Target{
					ThreatLevel: threatLevel,
					ThreatSide: threatSide,
					Range: float32(rangeData >> uint32(6*i) & 0x3F) * 3.125,
					Speed: float32(data[BufferIndexMessageData+6+i/2] >> byte(4*(i%2)) & 0x0F) * 3.04,
				}
				index := i;
				if pageNumber & ^ToggleMask == 0x31 {
					index += 4
				}
				s.Targets[index] = &target
			}
		case 0x50: // Common page 80 - Manufacturer's Identification
			s.HWVersion = data[BufferIndexMessageData+3]
			s.ManID = binary.LittleEndian.Uint16(data[BufferIndexMessageData+4 : BufferIndexMessageData+6])
			s.ModelNumber = binary.LittleEndian.Uint16(data[BufferIndexMessageData+6 : BufferIndexMessageData+8])
		case 0x51: // Common page 81 - Product Information
			supplementalVersion := data[BufferIndexMessageData+2]
			mainVersion := data[BufferIndexMessageData+3]
			if supplementalVersion != 0xFF {
				s.SWVersion = (float32(mainVersion)*100.0 + float32(supplementalVersion)) / 1000.0
			} else {
				s.SWVersion = float32(mainVersion) / 10.0
			}
			s.SerialNumber = binary.LittleEndian.Uint32(data[BufferIndexMessageData+4 : BufferIndexMessageData+8])
		case 0x52: // Common page 82 - Battery Status
			batteryIdentifier := data[BufferIndexMessageData+2]
			if batteryIdentifier != 0xFF {
				//TODO to support multi battery devices
			}
			batteryFrac := float32(data[BufferIndexMessageData+6])
			batteryStatus := data[BufferIndexMessageData+7]
			s.BatteryVoltage = float32(batteryStatus&0x0F) + (batteryFrac / 256.0)
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
			var otResolution uint32 = 16
			if batteryStatus&0x01 == 1 {
				otResolution = 2
			}
			s.OperatingTime = uint32(data[BufferIndexMessageData+3])
			s.OperatingTime |= uint32(data[BufferIndexMessageData+4]) << 8
			s.OperatingTime |= uint32(data[BufferIndexMessageData+5]) << 16
			s.OperatingTime *= otResolution
		case 0x57: // Common page 87 - Error Description
			s.ManufacturerError = binary.LittleEndian.Uint32(data[BufferIndexMessageData+4 : BufferIndexMessageData+8])
			infoField := data[BufferIndexMessageData+2]
			if infoField>>6 == 2 {
				s.ErrorLevel = "Critical"
			} else {
				s.ErrorLevel = "Warning"
			}
			s.ErrorComponent = infoField & 0x0F
			switch data[BufferIndexMessageData+3] {
			case 0:
				s.ErrorDesc = "Radar Saturated"
			case 1:
				s.ErrorDesc = "Unit Skew"
			}
		}
	}
	page.oldPage = pageNumber
}

type BikeRadarScannerState struct {
	*BikeRadarSensorState
	RSSI      uint32
	Threshold uint32
}

type BikeRadarSensor struct {
	//TODO do
	*AntPlusSensor
	state     *BikeRadarSensorState
	page      *Page
	listeners []func(*BikeRadarSensorState)
}

func NewBikeRadarSensor(driver Driver) *BikeRadarSensor {
	hrs := BikeRadarSensor{
		state: &BikeRadarSensorState{},
		page:  &Page{oldPage: 1<<8 - 1, pageState: InitPage},
	}
	hrs.AntPlusSensor = NewAntPlusSensor(driver, &hrs)
	return &hrs
}

func (sensor *BikeRadarSensor) updateState(deviceID uint32, data []byte) {
	sensor.state.update(sensor.page, data)
	for _, cb := range sensor.listeners {
		cb(sensor.state)
	}
}

func (sensor *BikeRadarSensor) ListenForData(cb func(*BikeRadarSensorState)) {
	sensor.listeners = append(sensor.listeners, cb)
}

type BikeRadarScanner struct {
	*AntPlusScanner
	states    map[uint32]*BikeRadarScannerState
	pages     map[uint32]*Page
	listeners []func(*BikeRadarScannerState)
}

func NewBikeRadarScannerState(deviceID uint32) *BikeRadarScannerState {
	return &BikeRadarScannerState{
		BikeRadarSensorState: &BikeRadarSensorState{
			DeviceID: deviceID,
		},
	}
}

func NewBikeRadarScanner(driver Driver) *BikeRadarScanner {
	hrs := BikeRadarScanner{
		states: make(map[uint32]*BikeRadarScannerState),
		pages:  make(map[uint32]*Page),
	}
	hrs.AntPlusScanner = NewAntPlusScanner(driver, &hrs)
	return &hrs
}

func (s *BikeRadarScanner) deviceType() uint32 {
	return 0x28
}

func (s *BikeRadarScanner) createStateIfNew(deviceID uint32) {
	if _, ok := s.states[deviceID]; !ok {
		s.states[deviceID] = NewBikeRadarScannerState(deviceID)
	}
	if _, ok := s.pages[deviceID]; !ok {
		s.pages[deviceID] = &Page{oldPage: 1<<8 - 1, pageState: InitPage}
	}
}

func (s *BikeRadarScanner) ListenForData(cb func(*BikeRadarScannerState)) {
	s.listeners = append(s.listeners, cb)
}

func (s *BikeRadarScanner) updateRssiAndThreshold(deviceID, rssi, threshold uint32) {
	s.states[deviceID].RSSI = rssi
	s.states[deviceID].Threshold = threshold
}

func (s *BikeRadarScanner) updateState(deviceID uint32, data []byte) {
	s.states[deviceID].update(s.pages[deviceID], data)
	for _, cb := range s.listeners {
		cb(s.states[deviceID])
	}
}
