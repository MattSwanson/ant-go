package ant

import (
	"fmt"
)


type StrideSpeedDistanceSensorState struct {
	DeviceID          uint32
	OperatingTime	  uint32
	ManID             byte
	SerialNumber      uint32
	HWVersion         byte
	SWVersion		  byte
	ModelNumber       byte

	TimeFractional	  byte
	TimeInteger byte
	DistanceFractional byte
	DistanceInteger byte
	SpeedFractional byte
	SpeedInteger byte
	StrideCount byte
	UpdateLatency byte
	CadenceFractional byte
	CadenceInteger byte
	Status byte
	Calories byte
}

func (s *StrideSpeedDistanceSensorState) update(page *Page, data []byte) {
	fmt.Println("flskdjf")
	pageNumber := data[BufferIndexMessageData]
	if page.pageState == InitPage {
		page.pageState = StdPage
	} else if pageNumber != page.oldPage || page.pageState == ExtPage {
		page.pageState = ExtPage
		switch pageNumber & ^ToggleMask {
		case 0x01:
			s.TimeFractional = data[BufferIndexMessageData+1]
			s.TimeInteger = data[BufferIndexMessageData+2]
			s.DistanceInteger = data[BufferIndexMessageData+3]
			s.DistanceFractional = data[BufferIndexMessageData+4] >> 4
			s.SpeedInteger = data[BufferIndexMessageData+4] & 0x0F
			s.SpeedFractional = data[BufferIndexMessageData+5]
			s.StrideCount = data[BufferIndexMessageData+6]
			s.UpdateLatency = data[BufferIndexMessageData+7]
		case 0x02:
			s.CadenceInteger = data[BufferIndexMessageData+3]
			s.CadenceFractional = data[BufferIndexMessageData+4] >> 4
			s.SpeedInteger = data[BufferIndexMessageData+4] & 0x0F
			s.SpeedFractional = data[BufferIndexMessageData+5]
			s.Status = data[BufferIndexMessageData+7]
		case 0x03:
			s.CadenceInteger = data[BufferIndexMessageData+3]
			s.CadenceFractional = data[BufferIndexMessageData+4] >> 4
			s.SpeedInteger = data[BufferIndexMessageData+4] & 0x0F
			s.SpeedFractional = data[BufferIndexMessageData+5]
			s.Calories = data[BufferIndexMessageData+6]
			s.Status = data[BufferIndexMessageData+7]
			
			/*case 1:
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
		*/
		}
	}
	page.oldPage = pageNumber
}

type StrideSpeedDistanceScannerState struct {
	*StrideSpeedDistanceSensorState
	RSSI      uint32
	Threshold uint32
}

type StrideSpeedDistanceSensor struct {
//TODO do
	*AntPlusSensor
	state *StrideSpeedDistanceSensorState
	page *Page
	listeners []func(*StrideSpeedDistanceSensorState)
}

func NewStrideSpeedDistanceSensor(driver Driver) *StrideSpeedDistanceSensor {
	hrs := StrideSpeedDistanceSensor{
		state: &StrideSpeedDistanceSensorState{},
		page: &Page{oldPage: 1 << 8 - 1, pageState: InitPage},
	}
	hrs.AntPlusSensor = NewAntPlusSensor(driver, &hrs)
	return &hrs
}

func (sensor *StrideSpeedDistanceSensor) updateState(deviceID uint32, data []byte) {
	sensor.state.update(sensor.page, data)
	for _, cb := range sensor.listeners {
		cb(sensor.state)
	}
}

func (sensor *StrideSpeedDistanceSensor) ListenForData(cb func(*StrideSpeedDistanceSensorState)) {
	sensor.listeners = append(sensor.listeners, cb)
}


type StrideSpeedDistanceScanner struct {
	*AntPlusScanner
	states map[uint32]*StrideSpeedDistanceScannerState
	pages map[uint32]*Page
	listeners []func(*StrideSpeedDistanceScannerState)
}

func NewStrideSpeedDistanceScannerState(deviceID uint32) *StrideSpeedDistanceScannerState {
	fmt.Println("ceating ssds with id: ", deviceID)
	return &StrideSpeedDistanceScannerState{
		StrideSpeedDistanceSensorState: &StrideSpeedDistanceSensorState{
			DeviceID: deviceID,
		},
	}
}

func NewStrideSpeedDistanceScanner(driver Driver) *StrideSpeedDistanceScanner {
	hrs := StrideSpeedDistanceScanner{
		states: make(map[uint32]*StrideSpeedDistanceScannerState),
		pages: make(map[uint32]*Page),
	}
	hrs.AntPlusScanner = NewAntPlusScanner(driver, &hrs)
	fmt.Println("created pod")
	return &hrs
}

func (s *StrideSpeedDistanceScanner) deviceType() uint32 {
	return 124
}

func (s *StrideSpeedDistanceScanner) createStateIfNew(deviceID uint32) {
	fmt.Println("create state if new :", deviceID)
	if _, ok := s.states[deviceID]; !ok {
		s.states[deviceID] = NewStrideSpeedDistanceScannerState(deviceID)
	}
	if _, ok := s.pages[deviceID]; !ok {
		s.pages[deviceID] = &Page{oldPage: 1 << 8 -1, pageState: InitPage}
	}
}

func (s *StrideSpeedDistanceScanner) ListenForData(cb func(*StrideSpeedDistanceScannerState)) {
	s.listeners = append(s.listeners, cb)
}

func (s *StrideSpeedDistanceScanner) updateRssiAndThreshold(deviceID, rssi, threshold uint32) {
	s.states[deviceID].RSSI = rssi
	s.states[deviceID].Threshold = threshold
}

func (s *StrideSpeedDistanceScanner) updateState(deviceID uint32, data []byte) {
	s.states[deviceID].update(s.pages[deviceID], data)
	for _, cb := range s.listeners {
		cb(s.states[deviceID])
	}
}
