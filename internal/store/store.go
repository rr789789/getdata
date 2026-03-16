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
	ErrProductNotFound   = errors.New("product not found")
	ErrProductExists     = errors.New("product already exists")
	ErrGroupNotFound     = errors.New("device group not found")
	ErrGroupExists       = errors.New("device group already exists")
	ErrRuleNotFound      = errors.New("rule not found")
	ErrRuleExists        = errors.New("rule already exists")
	ErrConfigNotFound    = errors.New("config profile not found")
	ErrConfigExists      = errors.New("config profile already exists")
	ErrAlertNotFound     = errors.New("alert not found")
	ErrShadowNotFound    = errors.New("device shadow not found")
)

type DeviceStore interface {
	CreateDevice(ctx context.Context, device model.Device) error
	GetDevice(ctx context.Context, deviceID string) (model.Device, error)
	ListDevices(ctx context.Context) ([]model.Device, error)
	SaveDevice(ctx context.Context, device model.Device) error
}

type ProductStore interface {
	CreateProduct(ctx context.Context, product model.Product) error
	GetProduct(ctx context.Context, productID string) (model.Product, error)
	ListProducts(ctx context.Context) ([]model.Product, error)
	SaveProduct(ctx context.Context, product model.Product) error
}

type GroupStore interface {
	CreateGroup(ctx context.Context, group model.DeviceGroup) error
	GetGroup(ctx context.Context, groupID string) (model.DeviceGroup, error)
	ListGroups(ctx context.Context) ([]model.DeviceGroup, error)
	SaveGroup(ctx context.Context, group model.DeviceGroup) error
	AddDeviceToGroup(ctx context.Context, groupID, deviceID string) error
	RemoveDeviceFromGroup(ctx context.Context, groupID, deviceID string) error
	ListGroupIDsByDevice(ctx context.Context, deviceID string) ([]string, error)
	ListDeviceIDsByGroup(ctx context.Context, groupID string) ([]string, error)
}

type RuleStore interface {
	CreateRule(ctx context.Context, rule model.Rule) error
	GetRule(ctx context.Context, ruleID string) (model.Rule, error)
	ListRules(ctx context.Context) ([]model.Rule, error)
	SaveRule(ctx context.Context, rule model.Rule) error
}

type ConfigStore interface {
	CreateConfigProfile(ctx context.Context, profile model.ConfigProfile) error
	GetConfigProfile(ctx context.Context, profileID string) (model.ConfigProfile, error)
	ListConfigProfiles(ctx context.Context) ([]model.ConfigProfile, error)
	SaveConfigProfile(ctx context.Context, profile model.ConfigProfile) error
}

type TelemetryStore interface {
	AppendTelemetry(ctx context.Context, telemetry model.Telemetry) error
	ListTelemetryByDevice(ctx context.Context, deviceID string, limit int) ([]model.Telemetry, error)
}

type ShadowStore interface {
	SaveShadow(ctx context.Context, shadow model.DeviceShadow) error
	GetShadow(ctx context.Context, deviceID string) (model.DeviceShadow, error)
}

type CommandStore interface {
	SaveCommand(ctx context.Context, command model.Command) error
	GetCommand(ctx context.Context, commandID string) (model.Command, error)
	UpdateCommandStatus(ctx context.Context, commandID string, status model.CommandStatus, result string) (model.Command, error)
	ListCommandsByDevice(ctx context.Context, deviceID string, limit int) ([]model.Command, error)
}

type AlertStore interface {
	AppendAlert(ctx context.Context, alert model.AlertEvent) error
	GetAlert(ctx context.Context, alertID string) (model.AlertEvent, error)
	SaveAlert(ctx context.Context, alert model.AlertEvent) error
	ListAlerts(ctx context.Context, limit int) ([]model.AlertEvent, error)
}
