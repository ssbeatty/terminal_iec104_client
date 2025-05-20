package iec_client

import (
	"time"
)

// DataType represents the type of IEC104 data
type DataType int

const (
	// Telemetry represents measured values (analog values)
	Telemetry DataType = iota
	// Teleindication represents status information (digital values)
	Teleindication
	// Telecontrol represents commands (digital control)
	Telecontrol
	// Teleregulation represents setpoints (analog control)
	Teleregulation
)

func (d DataType) String() string {
	switch d {
	case Telemetry:
		return "Telemetry"
	case Teleindication:
		return "Teleindication"
	case Telecontrol:
		return "Telecontrol"
	case Teleregulation:
		return "Teleregulation"
	default:
		return "Unknown"
	}
}

// DataPoint represents a generic IEC104 data point
type DataPoint struct {
	Address     int
	Description string
	Timestamp   time.Time
}

// TelemetryPoint represents a measured value (analog)
type TelemetryPoint struct {
	DataPoint
	Value float64
}

// TeleindPoint represents status information (digital)
type TeleindPoint struct {
	DataPoint
	Value bool
}

// TelecontrolPoint represents a command (digital control)
type TelecontrolPoint struct {
	DataPoint
	Value bool
}

// TeleregulationPoint represents a setpoint (analog control)
type TeleregulationPoint struct {
	DataPoint
	Value float64
}
