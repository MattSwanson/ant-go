package ant

import (
	"encoding/binary"
)

const (
	SpeedSensorDeviceType = 0x7B
	DefaultWheelCircumference = 2.199
)

// --------------------------------------------------------------
// SpeedSensorState
// -------------------------------------------------------------
type SpeedSensorState struct {
	DeviceID                       uint32
	SpeedEventTime                 uint32
	CumulativeSpeedRevolutionCount uint32
	CalculatedDistance             float32
	CalculatedSpeed                float32
	WheelCircumference float32

	OperatingTime  uint32
	ManID byte
	SerialNumber   uint32
	HWVersion      byte
	SWVersion      byte
	ModelNumber    byte
	BatteryVoltage float32
	BatteryStatus  string
	BatteryLevel   byte
	Motion         bool
}

func (s *SpeedSensorState) update(data []byte) {
	pageNumber := data[BufferIndexMessageData]
	switch pageNumber & ^ToggleMask {
		case 1:
			s.OperatingTime = uint32(data[BufferIndexMessageData + 1])
			s.OperatingTime = uint32(data[BufferIndexMessageData + 2]) << 8
			s.OperatingTime = uint32(data[BufferIndexMessageData + 3]) << 16
			s.OperatingTime *= 2
		case 2:
			s.ManID = data[BufferIndexMessageData+1]
			s.SerialNumber = uint32(s.DeviceID)
			s.SerialNumber |= uint32(binary.LittleEndian.Uint16(data[BufferIndexMessageData + 2:BufferIndexMessageData+4])) << 16
			s.SerialNumber ^= 0x80000000
		case 3:
			s.HWVersion = data[BufferIndexMessageData+1]
			s.SWVersion = data[BufferIndexMessageData+2]
			s.ModelNumber = data[BufferIndexMessageData+3]
		case 4:
			batteryLevel := data[BufferIndexMessageData+1]
			batteryFrac := float32(data[BufferIndexMessageData+2])
			batteryStatus := data[BufferIndexMessageData+3]
			if batteryLevel != 0xFF {
				s.BatteryLevel = batteryLevel
			}
			s.BatteryVoltage = float32(batteryStatus & 0x0F) + (batteryFrac / 256)
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
		case 5:
			s.Motion = (data[BufferIndexMessageData+1] & 0x01) == 0x01

	}
	oldSpeedTime := s.SpeedEventTime
	oldSpeedCount := s.CumulativeSpeedRevolutionCount;

	speedEventTime := uint32(binary.LittleEndian.Uint16(data[BufferIndexMessageData+4:BufferIndexMessageData+6]))
	speedRevolutionCount := uint32(binary.LittleEndian.Uint16(data[BufferIndexMessageData+6:BufferIndexMessageData+8]))

	if speedEventTime != oldSpeedTime {
		s.SpeedEventTime = speedEventTime
		s.CumulativeSpeedRevolutionCount = speedRevolutionCount

		if oldSpeedTime > speedEventTime {
			speedEventTime += (1024 * 64)
		}
		
		if oldSpeedCount > speedRevolutionCount {
			speedRevolutionCount += (1024 * 64)
		}

		distance := s.WheelCircumference * float32(speedRevolutionCount - oldSpeedCount)
		s.CalculatedDistance = distance

		deno := speedEventTime - oldSpeedTime
		if deno == 0 {
			return
		}
		s.CalculatedSpeed = (distance * 1024) / float32(deno)
	}

}

// -------------------------------------------------------------
// SpeedScannerState
// -------------------------------------------------------------
type SpeedScannerState struct {
	*SpeedSensorState
	RSSI uint32
	Threshold uint32
}

func NewSpeedScannerState(deviceID uint32) *SpeedScannerState {
	return &SpeedScannerState{
		SpeedSensorState: &SpeedSensorState{
			DeviceID: deviceID,
			WheelCircumference: DefaultWheelCircumference,
		},
	}
}

// -------------------------------------------------------------
// SpeedSensor
// -------------------------------------------------------------
type SpeedSensor struct {
	*AntPlusSensor
	state *SpeedSensorState
	listeners []func(*SpeedSensorState)
}

func NewSpeedSensor(driver Driver) *SpeedSensor {
	ss := SpeedSensor{
		state: &SpeedSensorState{
			WheelCircumference: DefaultWheelCircumference,
		},
	}
	ss.AntPlusSensor = NewAntPlusSensor(driver, &ss)
	return &ss
}

func (sensor *SpeedSensor) updateState(deviceID uint32, data []byte) {
	sensor.state.update(data)
	for _, cb := range sensor.listeners {
		cb(sensor.state)
	}
}

func (sensor *SpeedSensor) ListenForData(cb func(*SpeedSensorState)) {
	sensor.listeners = append(sensor.listeners, cb)
}

func (sensor *SpeedSensor) SetWheelCircumference(wheelCirc float32) {
	sensor.state.WheelCircumference = wheelCirc
}

// -------------------------------------------------------------
// SpeedScanner
// -------------------------------------------------------------
type SpeedScanner struct {
	*AntPlusScanner
	states map[uint32]*SpeedScannerState
	wheelCircumference float32
	listeners []func(*SpeedScannerState)
}

func NewSpeedScanner(driver Driver) *SpeedScanner {
	ss := SpeedScanner{
		states: make(map[uint32]*SpeedScannerState),
		wheelCircumference: DefaultWheelCircumference,
	}
	ss.AntPlusScanner = NewAntPlusScanner(driver, &ss)
	return &ss
}

func (s *SpeedScanner) deviceType() uint32 {
	return SpeedSensorDeviceType
}

func (s *SpeedScanner) SetWheelCircumference(deviceID uint32, wheelCirc float32) {
	s.states[deviceID].WheelCircumference = wheelCirc
}

func (s *SpeedScanner) createStateIfNew(deviceID uint32) {
	if _, ok := s.states[deviceID]; !ok {
		s.states[deviceID] = NewSpeedScannerState(deviceID)
	}
}

func (s *SpeedScanner) updateRssiAndThreshold(deviceID,  rssi, threshold uint32) {
	s.states[deviceID].RSSI = rssi
	s.states[deviceID].Threshold = threshold
}

func (s *SpeedScanner) updateState(deviceID uint32, data []byte) {
	s.states[deviceID].update(data)
	for _, cb := range s.listeners {
		cb(s.states[deviceID])
	}
}

func (s *SpeedScanner) ListenForData(cb func(*SpeedScannerState)) {
	s.listeners = append(s.listeners, cb)
}
