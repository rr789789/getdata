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

func (s *Service) CreateTenant(ctx context.Context, name, slug, description string, metadata map[string]string) (model.Tenant, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Tenant{}, fmt.Errorf("%w: tenant name is required", ErrInvalidProduct)
	}

	slug = normalizeTenantSlug(slug, name)
	existing, err := s.tenants.ListTenants(ctx)
	if err != nil {
		return model.Tenant{}, err
	}
	for _, tenant := range existing {
		if tenant.Slug == slug {
			return model.Tenant{}, store.ErrTenantExists
		}
	}

	now := time.Now().UTC()
	tenant := model.Tenant{
		ID:          util.NewID("ten"),
		Slug:        slug,
		Name:        name,
		Description: strings.TrimSpace(description),
		Metadata:    cloneStringMap(metadata),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.tenants.CreateTenant(ctx, tenant); err != nil {
		return model.Tenant{}, err
	}
	return tenant, nil
}

func (s *Service) ListTenants(ctx context.Context) ([]model.TenantView, error) {
	tenants, err := s.tenants.ListTenants(ctx)
	if err != nil {
		return nil, err
	}

	products, err := s.products.ListProducts(ctx)
	if err != nil {
		return nil, err
	}
	devices, err := s.devices.ListDevices(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := s.groups.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := s.rules.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	configProfiles, err := s.configs.ListConfigProfiles(ctx)
	if err != nil {
		return nil, err
	}
	firmwareArtifacts, err := s.firmware.ListFirmwareArtifacts(ctx)
	if err != nil {
		return nil, err
	}
	campaigns, err := s.campaigns.ListOTACampaigns(ctx)
	if err != nil {
		return nil, err
	}

	views := make([]model.TenantView, 0, len(tenants))
	for _, tenant := range tenants {
		view := model.TenantView{Tenant: tenant}
		for _, product := range products {
			if product.TenantID == tenant.ID {
				view.ProductCount++
			}
		}
		for _, device := range devices {
			if device.TenantID == tenant.ID {
				view.DeviceCount++
			}
		}
		for _, group := range groups {
			if group.TenantID == tenant.ID {
				view.GroupCount++
			}
		}
		for _, rule := range rules {
			if rule.TenantID == tenant.ID {
				view.RuleCount++
			}
		}
		for _, profile := range configProfiles {
			if profile.TenantID == tenant.ID {
				view.ConfigProfileCount++
			}
		}
		for _, artifact := range firmwareArtifacts {
			if artifact.TenantID == tenant.ID {
				view.FirmwareCount++
			}
		}
		for _, campaign := range campaigns {
			if campaign.TenantID == tenant.ID {
				view.OTACampaignCount++
			}
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) CreateProductWithTenant(ctx context.Context, tenantID, name, description string, metadata map[string]string, accessProfile model.ProductAccessProfile, thingModel model.ThingModel) (model.Product, error) {
	tenantID = strings.TrimSpace(tenantID)
	if err := s.assertTenantExists(ctx, tenantID); err != nil {
		return model.Product{}, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return model.Product{}, fmt.Errorf("%w: product name is required", ErrInvalidProduct)
	}

	now := time.Now().UTC()
	normalizedThingModel, err := normalizeThingModel(thingModel, 1, now)
	if err != nil {
		return model.Product{}, err
	}
	normalizedAccessProfile, err := normalizeAccessProfile(accessProfile)
	if err != nil {
		return model.Product{}, err
	}
	if err := validateAccessMappings(normalizedThingModel, normalizedAccessProfile); err != nil {
		return model.Product{}, err
	}

	product := model.Product{
		ID:            util.NewID("prd"),
		TenantID:      tenantID,
		Key:           util.NewID("pk"),
		Name:          name,
		Description:   strings.TrimSpace(description),
		Metadata:      cloneStringMap(metadata),
		AccessProfile: normalizedAccessProfile,
		ThingModel:    normalizedThingModel,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.products.CreateProduct(ctx, product); err != nil {
		return model.Product{}, err
	}
	return product, nil
}

func (s *Service) ListProductsByTenant(ctx context.Context, tenantID string) ([]model.ProductView, error) {
	products, err := s.products.ListProducts(ctx)
	if err != nil {
		return nil, err
	}

	tenantID = strings.TrimSpace(tenantID)
	views := make([]model.ProductView, 0, len(products))
	for _, product := range products {
		if tenantID != "" && product.TenantID != tenantID {
			continue
		}
		view, buildErr := s.buildProductView(ctx, product)
		if buildErr != nil {
			return nil, buildErr
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) CreateGroupWithTenant(ctx context.Context, tenantID, name, description, productID string, tags map[string]string) (model.DeviceGroup, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.DeviceGroup{}, fmt.Errorf("%w: group name is required", ErrInvalidGroup)
	}

	tenantID = strings.TrimSpace(tenantID)
	productID = strings.TrimSpace(productID)
	if err := s.assertTenantExists(ctx, tenantID); err != nil {
		return model.DeviceGroup{}, err
	}

	if productID != "" {
		product, err := s.products.GetProduct(ctx, productID)
		if err != nil {
			return model.DeviceGroup{}, err
		}
		if tenantID == "" {
			tenantID = product.TenantID
		} else if product.TenantID != tenantID {
			return model.DeviceGroup{}, fmt.Errorf("%w: group tenant does not match product tenant", ErrInvalidGroup)
		}
	}

	now := time.Now().UTC()
	group := model.DeviceGroup{
		ID:          util.NewID("grp"),
		TenantID:    tenantID,
		Name:        name,
		Description: strings.TrimSpace(description),
		ProductID:   productID,
		Tags:        cloneStringMap(tags),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.groups.CreateGroup(ctx, group); err != nil {
		return model.DeviceGroup{}, err
	}
	return group, nil
}

func (s *Service) ListGroupsByTenant(ctx context.Context, tenantID string) ([]model.GroupView, error) {
	groups, err := s.groups.ListGroups(ctx)
	if err != nil {
		return nil, err
	}

	tenantID = strings.TrimSpace(tenantID)
	views := make([]model.GroupView, 0, len(groups))
	for _, group := range groups {
		if tenantID != "" && group.TenantID != tenantID {
			continue
		}
		view, buildErr := s.buildGroupView(ctx, group)
		if buildErr != nil {
			return nil, buildErr
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) CreateDeviceWithTenant(ctx context.Context, tenantID, name, productID string, tags, metadata map[string]string) (model.Device, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "device"
	}

	tenantID = strings.TrimSpace(tenantID)
	productID = strings.TrimSpace(productID)
	if err := s.assertTenantExists(ctx, tenantID); err != nil {
		return model.Device{}, err
	}

	var product model.Product
	var hasProduct bool
	if productID != "" {
		var err error
		product, err = s.products.GetProduct(ctx, productID)
		if err != nil {
			return model.Device{}, err
		}
		hasProduct = true
		if tenantID == "" {
			tenantID = product.TenantID
		} else if product.TenantID != tenantID {
			return model.Device{}, fmt.Errorf("%w: device tenant does not match product tenant", ErrInvalidProduct)
		}
	}

	now := time.Now().UTC()
	device := model.Device{
		ID:         util.NewID("dev"),
		TenantID:   tenantID,
		Name:       name,
		ProductID:  productID,
		ProductKey: product.Key,
		Token:      util.NewToken(),
		Tags:       cloneStringMap(tags),
		Metadata:   cloneStringMap(metadata),
		CreatedAt:  now,
	}

	if err := s.devices.CreateDevice(ctx, device); err != nil {
		return model.Device{}, err
	}

	shadow := model.DeviceShadow{
		DeviceID:  device.ID,
		ProductID: productID,
		Reported:  map[string]any{},
		Desired:   map[string]any{},
		Version:   1,
		UpdatedAt: now,
	}
	if err := s.shadows.SaveShadow(ctx, shadow); err != nil {
		return model.Device{}, err
	}

	if hasProduct {
		s.TouchDevice(device.ID, now)
	}
	s.registeredDevices.Add(1)
	return device, nil
}

func (s *Service) ListDevicesByTenant(ctx context.Context, tenantID, productID string) ([]model.DeviceView, error) {
	devices, err := s.devices.ListDevices(ctx)
	if err != nil {
		return nil, err
	}

	tenantID = strings.TrimSpace(tenantID)
	productID = strings.TrimSpace(productID)
	result := make([]model.DeviceView, 0, len(devices))
	for _, device := range devices {
		if tenantID != "" && device.TenantID != tenantID {
			continue
		}
		if productID != "" && device.ProductID != productID {
			continue
		}
		view, buildErr := s.buildDeviceView(ctx, device)
		if buildErr != nil {
			return nil, buildErr
		}
		result = append(result, view)
	}
	return result, nil
}

func (s *Service) CreateConfigProfileWithTenant(ctx context.Context, tenantID, name, description, productID string, values map[string]any) (model.ConfigProfile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.ConfigProfile{}, fmt.Errorf("%w: config profile name is required", ErrInvalidConfig)
	}

	tenantID = strings.TrimSpace(tenantID)
	productID = strings.TrimSpace(productID)
	if err := s.assertTenantExists(ctx, tenantID); err != nil {
		return model.ConfigProfile{}, err
	}

	product, hasProduct := s.loadProduct(ctx, productID)
	if productID != "" && !hasProduct {
		return model.ConfigProfile{}, store.ErrProductNotFound
	}
	if hasProduct {
		if tenantID == "" {
			tenantID = product.TenantID
		} else if product.TenantID != tenantID {
			return model.ConfigProfile{}, fmt.Errorf("%w: config tenant does not match product tenant", ErrInvalidConfig)
		}
		if err := validateThingValues(product, values); err != nil {
			return model.ConfigProfile{}, err
		}
	}

	now := time.Now().UTC()
	profile := model.ConfigProfile{
		ID:          util.NewID("cfg"),
		TenantID:    tenantID,
		Name:        name,
		Description: strings.TrimSpace(description),
		ProductID:   productID,
		Values:      cloneAnyMap(values),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if profile.Values == nil {
		profile.Values = map[string]any{}
	}

	if err := s.configs.CreateConfigProfile(ctx, profile); err != nil {
		return model.ConfigProfile{}, err
	}
	return profile, nil
}

func (s *Service) ListConfigProfilesByTenant(ctx context.Context, tenantID string) ([]model.ConfigProfileView, error) {
	profiles, err := s.configs.ListConfigProfiles(ctx)
	if err != nil {
		return nil, err
	}

	tenantID = strings.TrimSpace(tenantID)
	views := make([]model.ConfigProfileView, 0, len(profiles))
	for _, profile := range profiles {
		if tenantID != "" && profile.TenantID != tenantID {
			continue
		}
		view, buildErr := s.buildConfigProfileView(ctx, profile)
		if buildErr != nil {
			return nil, buildErr
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) CreateRuleWithTenant(ctx context.Context, tenantID, name, description, productID, groupID, deviceID string, enabled bool, severity model.AlertSeverity, cooldownSeconds int, condition model.RuleCondition, actions []model.RuleAction) (model.Rule, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.Rule{}, fmt.Errorf("%w: rule name is required", ErrInvalidRule)
	}

	tenantID = strings.TrimSpace(tenantID)
	groupID = strings.TrimSpace(groupID)
	deviceID = strings.TrimSpace(deviceID)
	productID = strings.TrimSpace(productID)
	if productID == "" && groupID == "" && deviceID == "" {
		return model.Rule{}, fmt.Errorf("%w: at least one scope product_id, group_id or device_id is required", ErrInvalidRule)
	}
	if cooldownSeconds < 0 {
		return model.Rule{}, fmt.Errorf("%w: cooldown_seconds must be >= 0", ErrInvalidRule)
	}
	if err := s.assertTenantExists(ctx, tenantID); err != nil {
		return model.Rule{}, err
	}

	if groupID != "" {
		group, err := s.groups.GetGroup(ctx, groupID)
		if err != nil {
			return model.Rule{}, err
		}
		if tenantID == "" {
			tenantID = group.TenantID
		} else if group.TenantID != tenantID {
			return model.Rule{}, fmt.Errorf("%w: group tenant scope mismatch", ErrInvalidRule)
		}
		if group.ProductID != "" {
			if productID == "" {
				productID = group.ProductID
			} else if productID != group.ProductID {
				return model.Rule{}, fmt.Errorf("%w: group product scope mismatch", ErrInvalidRule)
			}
		}
	}

	if deviceID != "" {
		device, err := s.devices.GetDevice(ctx, deviceID)
		if err != nil {
			return model.Rule{}, err
		}
		if tenantID == "" {
			tenantID = device.TenantID
		} else if device.TenantID != tenantID {
			return model.Rule{}, fmt.Errorf("%w: device tenant scope mismatch", ErrInvalidRule)
		}
		if device.ProductID != "" {
			if productID == "" {
				productID = device.ProductID
			} else if productID != device.ProductID {
				return model.Rule{}, fmt.Errorf("%w: device product scope mismatch", ErrInvalidRule)
			}
		}
	}

	product, hasProduct := s.loadProduct(ctx, productID)
	if hasProduct {
		if tenantID == "" {
			tenantID = product.TenantID
		} else if product.TenantID != tenantID {
			return model.Rule{}, fmt.Errorf("%w: product tenant scope mismatch", ErrInvalidRule)
		}
	}

	normalizedCondition, err := normalizeRuleCondition(product, hasProduct, condition)
	if err != nil {
		return model.Rule{}, err
	}
	normalizedActions, err := s.normalizeRuleActions(ctx, tenantID, product, hasProduct, severity, actions)
	if err != nil {
		return model.Rule{}, err
	}

	now := time.Now().UTC()
	rule := model.Rule{
		ID:              util.NewID("rul"),
		TenantID:        tenantID,
		Name:            name,
		Description:     strings.TrimSpace(description),
		ProductID:       productID,
		GroupID:         groupID,
		DeviceID:        deviceID,
		Enabled:         enabled,
		Severity:        normalizeSeverity(severity),
		CooldownSeconds: cooldownSeconds,
		Condition:       normalizedCondition,
		Actions:         normalizedActions,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.rules.CreateRule(ctx, rule); err != nil {
		return model.Rule{}, err
	}
	return rule, nil
}

func (s *Service) ListRulesByTenant(ctx context.Context, tenantID string) ([]model.RuleView, error) {
	rules, err := s.rules.ListRules(ctx)
	if err != nil {
		return nil, err
	}

	tenantID = strings.TrimSpace(tenantID)
	views := make([]model.RuleView, 0, len(rules))
	for _, rule := range rules {
		if tenantID != "" && rule.TenantID != tenantID {
			continue
		}
		view, buildErr := s.buildRuleView(ctx, rule)
		if buildErr != nil {
			return nil, buildErr
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) CreateFirmwareArtifact(ctx context.Context, tenantID, productID, name, version, fileName, url, checksum, checksumType string, sizeBytes int64, metadata map[string]string, notes string) (model.FirmwareArtifact, error) {
	tenantID = strings.TrimSpace(tenantID)
	productID = strings.TrimSpace(productID)
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	url = strings.TrimSpace(url)

	if name == "" || version == "" || url == "" {
		return model.FirmwareArtifact{}, fmt.Errorf("%w: firmware name, version and url are required", ErrInvalidConfig)
	}
	if err := s.assertTenantExists(ctx, tenantID); err != nil {
		return model.FirmwareArtifact{}, err
	}
	if productID != "" {
		product, err := s.products.GetProduct(ctx, productID)
		if err != nil {
			return model.FirmwareArtifact{}, err
		}
		if tenantID == "" {
			tenantID = product.TenantID
		} else if product.TenantID != tenantID {
			return model.FirmwareArtifact{}, fmt.Errorf("%w: firmware tenant does not match product tenant", ErrInvalidConfig)
		}
	}

	now := time.Now().UTC()
	artifact := model.FirmwareArtifact{
		ID:           util.NewID("fw"),
		TenantID:     tenantID,
		ProductID:    productID,
		Name:         name,
		Version:      version,
		FileName:     strings.TrimSpace(fileName),
		URL:          url,
		Checksum:     strings.TrimSpace(checksum),
		ChecksumType: strings.TrimSpace(checksumType),
		SizeBytes:    sizeBytes,
		Notes:        strings.TrimSpace(notes),
		Metadata:     cloneStringMap(metadata),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.firmware.CreateFirmwareArtifact(ctx, artifact); err != nil {
		return model.FirmwareArtifact{}, err
	}
	return artifact, nil
}

func (s *Service) ListFirmwareArtifacts(ctx context.Context, tenantID, productID string) ([]model.FirmwareArtifactView, error) {
	artifacts, err := s.firmware.ListFirmwareArtifacts(ctx)
	if err != nil {
		return nil, err
	}

	tenantID = strings.TrimSpace(tenantID)
	productID = strings.TrimSpace(productID)
	views := make([]model.FirmwareArtifactView, 0, len(artifacts))
	for _, artifact := range artifacts {
		if tenantID != "" && artifact.TenantID != tenantID {
			continue
		}
		if productID != "" && artifact.ProductID != productID {
			continue
		}
		view, buildErr := s.buildFirmwareArtifactView(ctx, artifact)
		if buildErr != nil {
			return nil, buildErr
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) CreateOTACampaign(ctx context.Context, tenantID, name, firmwareID, productID, groupID, deviceID string) (model.OTACampaign, error) {
	tenantID = strings.TrimSpace(tenantID)
	name = strings.TrimSpace(name)
	firmwareID = strings.TrimSpace(firmwareID)
	productID = strings.TrimSpace(productID)
	groupID = strings.TrimSpace(groupID)
	deviceID = strings.TrimSpace(deviceID)

	if name == "" || firmwareID == "" {
		return model.OTACampaign{}, fmt.Errorf("%w: ota name and firmware_id are required", ErrInvalidConfig)
	}

	artifact, err := s.firmware.GetFirmwareArtifact(ctx, firmwareID)
	if err != nil {
		return model.OTACampaign{}, err
	}
	if tenantID == "" {
		tenantID = artifact.TenantID
	} else if artifact.TenantID != tenantID {
		return model.OTACampaign{}, fmt.Errorf("%w: ota tenant does not match firmware tenant", ErrInvalidConfig)
	}

	resolvedProductID, err := s.resolveOTATargetProduct(ctx, tenantID, artifact, productID, groupID, deviceID)
	if err != nil {
		return model.OTACampaign{}, err
	}

	now := time.Now().UTC()
	campaign := model.OTACampaign{
		ID:         util.NewID("ota"),
		TenantID:   tenantID,
		Name:       name,
		FirmwareID: artifact.ID,
		ProductID:  resolvedProductID,
		GroupID:    groupID,
		DeviceID:   deviceID,
		Status:     model.OTACampaignStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.campaigns.CreateOTACampaign(ctx, campaign); err != nil {
		return model.OTACampaign{}, err
	}

	updated, err := s.dispatchOTACampaign(ctx, campaign, artifact)
	if err != nil {
		return model.OTACampaign{}, err
	}
	return updated, nil
}

func (s *Service) ListOTACampaigns(ctx context.Context, tenantID string) ([]model.OTACampaignView, error) {
	campaigns, err := s.campaigns.ListOTACampaigns(ctx)
	if err != nil {
		return nil, err
	}

	tenantID = strings.TrimSpace(tenantID)
	views := make([]model.OTACampaignView, 0, len(campaigns))
	for _, campaign := range campaigns {
		if tenantID != "" && campaign.TenantID != tenantID {
			continue
		}
		view, buildErr := s.buildOTACampaignView(ctx, campaign)
		if buildErr != nil {
			return nil, buildErr
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) normalizeRuleActions(ctx context.Context, tenantID string, product model.Product, hasProduct bool, defaultSeverity model.AlertSeverity, actions []model.RuleAction) ([]model.RuleAction, error) {
	if len(actions) == 0 {
		return []model.RuleAction{{Type: model.RuleActionAlert, Severity: normalizeSeverity(defaultSeverity)}}, nil
	}

	normalized := make([]model.RuleAction, 0, len(actions))
	for _, action := range actions {
		actionType := normalizeRuleActionType(action.Type)
		if actionType == "" {
			return nil, fmt.Errorf("%w: unsupported action type", ErrInvalidRule)
		}
		item := model.RuleAction{
			Type:            actionType,
			Name:            strings.TrimSpace(action.Name),
			Params:          cloneAnyMap(action.Params),
			ConfigProfileID: strings.TrimSpace(action.ConfigProfileID),
			Severity:        normalizeSeverity(action.Severity),
			Message:         strings.TrimSpace(action.Message),
		}
		switch item.Type {
		case model.RuleActionAlert:
			if item.Severity == "" {
				item.Severity = normalizeSeverity(defaultSeverity)
			}
		case model.RuleActionSendCommand:
			if item.Name == "" {
				return nil, fmt.Errorf("%w: send_command action requires name", ErrInvalidRule)
			}
			if hasProduct {
				if err := validateCommandName(product, item.Name); err != nil {
					return nil, err
				}
			}
		case model.RuleActionApplyConfig:
			if item.ConfigProfileID == "" {
				return nil, fmt.Errorf("%w: apply_config_profile action requires config_profile_id", ErrInvalidRule)
			}
			profile, err := s.configs.GetConfigProfile(ctx, item.ConfigProfileID)
			if err != nil {
				return nil, err
			}
			if tenantID != "" && profile.TenantID != tenantID {
				return nil, fmt.Errorf("%w: config profile tenant mismatch", ErrInvalidRule)
			}
			if hasProduct && profile.ProductID != "" && profile.ProductID != product.ID {
				return nil, fmt.Errorf("%w: config profile product mismatch", ErrInvalidRule)
			}
		}
		normalized = append(normalized, item)
	}
	return normalized, nil
}

func normalizeRuleActionType(value model.RuleActionType) model.RuleActionType {
	switch strings.ToLower(strings.TrimSpace(string(value))) {
	case "alert":
		return model.RuleActionAlert
	case "send_command", "command":
		return model.RuleActionSendCommand
	case "apply_config_profile", "config", "config_profile":
		return model.RuleActionApplyConfig
	default:
		return ""
	}
}

func (s *Service) assertTenantExists(ctx context.Context, tenantID string) error {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil
	}
	if _, err := s.tenants.GetTenant(ctx, tenantID); err != nil {
		return err
	}
	return nil
}

func (s *Service) loadTenant(ctx context.Context, tenantID string) (model.Tenant, bool) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return model.Tenant{}, false
	}
	tenant, err := s.tenants.GetTenant(ctx, tenantID)
	if err != nil {
		return model.Tenant{}, false
	}
	return tenant, true
}

func (s *Service) tenantSummary(ctx context.Context, tenantID string) (*model.TenantSummary, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, nil
	}
	tenant, err := s.tenants.GetTenant(ctx, tenantID)
	if err != nil {
		if errors.Is(err, store.ErrTenantNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &model.TenantSummary{ID: tenant.ID, Slug: tenant.Slug, Name: tenant.Name}, nil
}

func normalizeTenantSlug(slug, name string) string {
	raw := strings.TrimSpace(strings.ToLower(slug))
	if raw == "" {
		raw = strings.TrimSpace(strings.ToLower(name))
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "tenant"
	}
	return result
}

func (s *Service) resolveOTATargetProduct(ctx context.Context, tenantID string, artifact model.FirmwareArtifact, productID, groupID, deviceID string) (string, error) {
	if deviceID != "" {
		device, err := s.devices.GetDevice(ctx, deviceID)
		if err != nil {
			return "", err
		}
		if tenantID != "" && device.TenantID != tenantID {
			return "", fmt.Errorf("%w: ota device tenant mismatch", ErrInvalidConfig)
		}
		if artifact.ProductID != "" && device.ProductID != artifact.ProductID {
			return "", fmt.Errorf("%w: ota device product mismatch", ErrInvalidConfig)
		}
		return device.ProductID, nil
	}
	if groupID != "" {
		group, err := s.groups.GetGroup(ctx, groupID)
		if err != nil {
			return "", err
		}
		if tenantID != "" && group.TenantID != tenantID {
			return "", fmt.Errorf("%w: ota group tenant mismatch", ErrInvalidConfig)
		}
		if artifact.ProductID != "" && group.ProductID != "" && group.ProductID != artifact.ProductID {
			return "", fmt.Errorf("%w: ota group product mismatch", ErrInvalidConfig)
		}
		if productID == "" {
			productID = group.ProductID
		}
	}
	if productID != "" {
		product, err := s.products.GetProduct(ctx, productID)
		if err != nil {
			return "", err
		}
		if tenantID != "" && product.TenantID != tenantID {
			return "", fmt.Errorf("%w: ota product tenant mismatch", ErrInvalidConfig)
		}
		if artifact.ProductID != "" && artifact.ProductID != product.ID {
			return "", fmt.Errorf("%w: ota firmware product mismatch", ErrInvalidConfig)
		}
		return product.ID, nil
	}
	if artifact.ProductID != "" {
		return artifact.ProductID, nil
	}
	return "", fmt.Errorf("%w: ota target scope is required", ErrInvalidConfig)
}

func (s *Service) dispatchOTACampaign(ctx context.Context, campaign model.OTACampaign, artifact model.FirmwareArtifact) (model.OTACampaign, error) {
	deviceIDs, err := s.resolveOTATargetDevices(ctx, campaign)
	if err != nil {
		return model.OTACampaign{}, err
	}

	params := map[string]any{
		"campaign_id":    campaign.ID,
		"firmware_id":    artifact.ID,
		"name":           artifact.Name,
		"version":        artifact.Version,
		"url":            artifact.URL,
		"checksum":       artifact.Checksum,
		"checksum_type":  artifact.ChecksumType,
		"file_name":      artifact.FileName,
		"size_bytes":     artifact.SizeBytes,
	}

	campaign.TotalDevices = len(deviceIDs)
	campaign.DispatchedCount = 0
	campaign.FailedCount = 0
	now := time.Now().UTC()
	for _, deviceID := range deviceIDs {
		_, sendErr := s.sendCommandWithCampaign(ctx, deviceID, campaign.ID, "ota_upgrade", params)
		if sendErr != nil {
			campaign.FailedCount++
			continue
		}
		campaign.DispatchedCount++
	}
	campaign.LastDispatchedAt = &now
	campaign.UpdatedAt = now
	switch {
	case campaign.DispatchedCount > 0 && campaign.FailedCount == 0:
		campaign.Status = model.OTACampaignStatusRunning
	case campaign.DispatchedCount > 0:
		campaign.Status = model.OTACampaignStatusPartial
	default:
		campaign.Status = model.OTACampaignStatusPending
	}

	if err := s.campaigns.SaveOTACampaign(ctx, campaign); err != nil {
		return model.OTACampaign{}, err
	}
	return campaign, nil
}

func (s *Service) resolveOTATargetDevices(ctx context.Context, campaign model.OTACampaign) ([]string, error) {
	if campaign.DeviceID != "" {
		return []string{campaign.DeviceID}, nil
	}
	if campaign.GroupID != "" {
		return s.groups.ListDeviceIDsByGroup(ctx, campaign.GroupID)
	}

	devices, err := s.devices.ListDevices(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(devices))
	for _, device := range devices {
		if campaign.TenantID != "" && device.TenantID != campaign.TenantID {
			continue
		}
		if campaign.ProductID != "" && device.ProductID != campaign.ProductID {
			continue
		}
		result = append(result, device.ID)
	}
	return result, nil
}

func (s *Service) buildFirmwareArtifactView(ctx context.Context, artifact model.FirmwareArtifact) (model.FirmwareArtifactView, error) {
	view := model.FirmwareArtifactView{Artifact: artifact}
	tenant, err := s.tenantSummary(ctx, artifact.TenantID)
	if err != nil {
		return model.FirmwareArtifactView{}, err
	}
	view.Tenant = tenant
	if artifact.ProductID != "" {
		product, err := s.products.GetProduct(ctx, artifact.ProductID)
		if err != nil && !errors.Is(err, store.ErrProductNotFound) {
			return model.FirmwareArtifactView{}, err
		}
		if err == nil {
			view.Product = &model.ProductSummary{ID: product.ID, Key: product.Key, Name: product.Name}
		}
	}
	return view, nil
}

func (s *Service) buildOTACampaignView(ctx context.Context, campaign model.OTACampaign) (model.OTACampaignView, error) {
	view := model.OTACampaignView{Campaign: campaign}
	tenant, err := s.tenantSummary(ctx, campaign.TenantID)
	if err != nil {
		return model.OTACampaignView{}, err
	}
	view.Tenant = tenant

	if campaign.ProductID != "" {
		product, err := s.products.GetProduct(ctx, campaign.ProductID)
		if err != nil && !errors.Is(err, store.ErrProductNotFound) {
			return model.OTACampaignView{}, err
		}
		if err == nil {
			view.Product = &model.ProductSummary{ID: product.ID, Key: product.Key, Name: product.Name}
		}
	}
	if campaign.GroupID != "" {
		group, err := s.groups.GetGroup(ctx, campaign.GroupID)
		if err != nil && !errors.Is(err, store.ErrGroupNotFound) {
			return model.OTACampaignView{}, err
		}
		if err == nil {
			view.Group = &model.GroupSummary{ID: group.ID, Name: group.Name}
		}
	}
	if campaign.DeviceID != "" {
		device, err := s.devices.GetDevice(ctx, campaign.DeviceID)
		if err != nil && !errors.Is(err, store.ErrDeviceNotFound) {
			return model.OTACampaignView{}, err
		}
		if err == nil {
			view.Device = &model.DeviceSummary{ID: device.ID, Name: device.Name}
		}
	}
	if campaign.FirmwareID != "" {
		artifact, err := s.firmware.GetFirmwareArtifact(ctx, campaign.FirmwareID)
		if err != nil && !errors.Is(err, store.ErrFirmwareNotFound) {
			return model.OTACampaignView{}, err
		}
		if err == nil {
			view.Firmware = &model.FirmwareArtifactSummary{ID: artifact.ID, Name: artifact.Name, Version: artifact.Version}
		}
	}
	return view, nil
}
