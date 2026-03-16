package store

import (
	"context"
	"errors"

	"mvp-platform/internal/model"
)

var (
	ErrDeviceNotFound    = errors.New("device not found")
	ErrDeviceExists      = errors.New("device already exists")
	ErrInvalidCredential = errors.New("invalid device credential")
	ErrCommandNotFound   = errors.New("command not found")
)

type DeviceStore interface {
	CreateDevice(ctx context.Context, device model.Device) error
	GetDevice(ctx context.Context, deviceID string) (model.Device, error)
}

type TelemetryStore interface {
	AppendTelemetry(ctx context.Context, telemetry model.Telemetry) error
	ListTelemetryByDevice(ctx context.Context, deviceID string, limit int) ([]model.Telemetry, error)
}

type CommandStore interface {
	SaveCommand(ctx context.Context, command model.Command) error
	GetCommand(ctx context.Context, commandID string) (model.Command, error)
	UpdateCommandStatus(ctx context.Context, commandID string, status model.CommandStatus, result string) (model.Command, error)
	ListCommandsByDevice(ctx context.Context, deviceID string, limit int) ([]model.Command, error)
}
