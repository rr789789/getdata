package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"mvp-platform/internal/model"
	"mvp-platform/internal/store"
	"mvp-platform/internal/util"
)

func (s *Service) sendCommandWithCampaign(ctx context.Context, deviceID, campaignID, name string, params map[string]any) (model.Command, error) {
	device, err := s.devices.GetDevice(ctx, deviceID)
	if err != nil {
		return model.Command{}, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return model.Command{}, errors.New("command name is required")
	}

	product, hasProduct := s.loadProduct(ctx, device.ProductID)
	if hasProduct {
		if err := validateCommandName(product, name); err != nil {
			return model.Command{}, err
		}
	}

	now := time.Now().UTC()
	command := model.Command{
		ID:         util.NewID("cmd"),
		DeviceID:   deviceID,
		CampaignID: strings.TrimSpace(campaignID),
		Name:       name,
		Params:     cloneAnyMap(params),
		Status:     model.CommandStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.commands.SaveCommand(ctx, command); err != nil {
		return model.Command{}, err
	}

	s.mu.RLock()
	state := s.states[deviceID]
	session := state.session
	s.mu.RUnlock()

	if session == nil {
		updated, updateErr := s.commands.UpdateCommandStatus(ctx, command.ID, model.CommandStatusFailed, ErrDeviceOffline.Error())
		if updateErr == nil {
			command = updated
		}
		return command, ErrDeviceOffline
	}

	message := model.ServerMessage{
		Type:       "command",
		ServerTime: now.UnixMilli(),
		CommandID:  command.ID,
		Name:       command.Name,
		Params:     cloneAnyMap(command.Params),
	}
	if err := session.Send(message); err != nil {
		updated, updateErr := s.commands.UpdateCommandStatus(ctx, command.ID, model.CommandStatusFailed, err.Error())
		if updateErr == nil {
			command = updated
		}
		return command, err
	}

	updated, err := s.commands.UpdateCommandStatus(ctx, command.ID, model.CommandStatusSent, "sent")
	if err != nil {
		return model.Command{}, err
	}

	s.commandsSent.Add(1)
	s.recordCommandPublished(sessionTransportName(session))
	s.TouchDevice(deviceID, now)
	return updated, nil
}

func (s *Service) updateCampaignFromCommandAck(ctx context.Context, campaignID string, status model.CommandStatus) error {
	campaign, err := s.campaigns.GetOTACampaign(ctx, campaignID)
	if err != nil {
		return err
	}
	switch status {
	case model.CommandStatusFailed:
		campaign.FailedCount++
	default:
		campaign.AckedCount++
	}
	if campaign.AckedCount+campaign.FailedCount >= campaign.TotalDevices && campaign.TotalDevices > 0 {
		if campaign.FailedCount > 0 {
			campaign.Status = model.OTACampaignStatusPartial
		} else {
			campaign.Status = model.OTACampaignStatusCompleted
		}
	} else if campaign.DispatchedCount > 0 {
		campaign.Status = model.OTACampaignStatusRunning
	}
	campaign.UpdatedAt = time.Now().UTC()
	return s.campaigns.SaveOTACampaign(ctx, campaign)
}

func (s *Service) executeRuleActions(ctx context.Context, rule model.Rule, device model.Device, telemetry model.Telemetry, value any) {
	actions := rule.Actions
	if len(actions) == 0 {
		actions = []model.RuleAction{{Type: model.RuleActionAlert, Severity: rule.Severity}}
	}

	for _, action := range actions {
		switch action.Type {
		case model.RuleActionSendCommand:
			if _, err := s.sendCommandWithCampaign(ctx, device.ID, "", action.Name, action.Params); err != nil && !errors.Is(err, ErrDeviceOffline) {
				s.logger.Warn("rule send_command action failed", "rule_id", rule.ID, "device_id", device.ID, "error", err)
			}
		case model.RuleActionApplyConfig:
			if _, err := s.ApplyConfigProfile(ctx, action.ConfigProfileID, device.ID); err != nil {
				s.logger.Warn("rule apply_config_profile action failed", "rule_id", rule.ID, "device_id", device.ID, "config_profile_id", action.ConfigProfileID, "error", err)
			}
		default:
			alert := buildRuleAlert(rule, device, telemetry.Timestamp, value, action)
			if err := s.alerts.AppendAlert(ctx, alert); err != nil {
				s.logger.Warn("rule alert action failed", "rule_id", rule.ID, "device_id", device.ID, "error", err)
			}
		}
	}
}

func buildRuleAlert(rule model.Rule, device model.Device, at time.Time, value any, action model.RuleAction) model.AlertEvent {
	severity := normalizeSeverity(rule.Severity)
	if action.Severity != "" {
		severity = normalizeSeverity(action.Severity)
	}

	message := strings.TrimSpace(action.Message)
	if message == "" {
		message = fmt.Sprintf("%s: %s %s %v, got %v", rule.Name, rule.Condition.Property, rule.Condition.Operator, rule.Condition.Value, value)
	}

	return model.AlertEvent{
		ID:          util.NewID("alt"),
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		TenantID:    device.TenantID,
		ProductID:   device.ProductID,
		GroupID:     rule.GroupID,
		DeviceID:    device.ID,
		DeviceName:  device.Name,
		Property:    rule.Condition.Property,
		Operator:    rule.Condition.Operator,
		Threshold:   rule.Condition.Value,
		Value:       value,
		Severity:    severity,
		Status:      model.AlertStatusNew,
		Message:     message,
		TriggeredAt: at,
	}
}
