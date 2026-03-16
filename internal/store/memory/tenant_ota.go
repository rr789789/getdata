package memory

import (
	"context"
	"sort"

	"mvp-platform/internal/model"
	"mvp-platform/internal/store"
)

func (s *Store) CreateTenant(_ context.Context, tenant model.Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tenants[tenant.ID]; exists {
		return store.ErrTenantExists
	}
	s.tenants[tenant.ID] = cloneTenant(tenant)
	return nil
}

func (s *Store) GetTenant(_ context.Context, tenantID string) (model.Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tenant, exists := s.tenants[tenantID]
	if !exists {
		return model.Tenant{}, store.ErrTenantNotFound
	}
	return cloneTenant(tenant), nil
}

func (s *Store) ListTenants(_ context.Context) ([]model.Tenant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.Tenant, 0, len(s.tenants))
	for _, tenant := range s.tenants {
		result = append(result, cloneTenant(tenant))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Store) SaveTenant(_ context.Context, tenant model.Tenant) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tenants[tenant.ID]; !exists {
		return store.ErrTenantNotFound
	}
	s.tenants[tenant.ID] = cloneTenant(tenant)
	return nil
}

func (s *Store) CreateFirmwareArtifact(_ context.Context, artifact model.FirmwareArtifact) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.firmwareArtifacts[artifact.ID]; exists {
		return store.ErrFirmwareExists
	}
	s.firmwareArtifacts[artifact.ID] = cloneFirmwareArtifact(artifact)
	return nil
}

func (s *Store) GetFirmwareArtifact(_ context.Context, artifactID string) (model.FirmwareArtifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	artifact, exists := s.firmwareArtifacts[artifactID]
	if !exists {
		return model.FirmwareArtifact{}, store.ErrFirmwareNotFound
	}
	return cloneFirmwareArtifact(artifact), nil
}

func (s *Store) ListFirmwareArtifacts(_ context.Context) ([]model.FirmwareArtifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.FirmwareArtifact, 0, len(s.firmwareArtifacts))
	for _, artifact := range s.firmwareArtifacts {
		result = append(result, cloneFirmwareArtifact(artifact))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Store) SaveFirmwareArtifact(_ context.Context, artifact model.FirmwareArtifact) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.firmwareArtifacts[artifact.ID]; !exists {
		return store.ErrFirmwareNotFound
	}
	s.firmwareArtifacts[artifact.ID] = cloneFirmwareArtifact(artifact)
	return nil
}

func (s *Store) CreateOTACampaign(_ context.Context, campaign model.OTACampaign) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.otaCampaigns[campaign.ID]; exists {
		return store.ErrOTAExists
	}
	s.otaCampaigns[campaign.ID] = cloneOTACampaign(campaign)
	return nil
}

func (s *Store) GetOTACampaign(_ context.Context, campaignID string) (model.OTACampaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	campaign, exists := s.otaCampaigns[campaignID]
	if !exists {
		return model.OTACampaign{}, store.ErrOTANotFound
	}
	return cloneOTACampaign(campaign), nil
}

func (s *Store) ListOTACampaigns(_ context.Context) ([]model.OTACampaign, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]model.OTACampaign, 0, len(s.otaCampaigns))
	for _, campaign := range s.otaCampaigns {
		result = append(result, cloneOTACampaign(campaign))
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *Store) SaveOTACampaign(_ context.Context, campaign model.OTACampaign) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.otaCampaigns[campaign.ID]; !exists {
		return store.ErrOTANotFound
	}
	s.otaCampaigns[campaign.ID] = cloneOTACampaign(campaign)
	return nil
}

func cloneTenant(tenant model.Tenant) model.Tenant {
	return model.Tenant{
		ID:          tenant.ID,
		Slug:        tenant.Slug,
		Name:        tenant.Name,
		Description: tenant.Description,
		Metadata:    cloneStringMap(tenant.Metadata),
		CreatedAt:   tenant.CreatedAt,
		UpdatedAt:   tenant.UpdatedAt,
	}
}

func cloneFirmwareArtifact(artifact model.FirmwareArtifact) model.FirmwareArtifact {
	return model.FirmwareArtifact{
		ID:           artifact.ID,
		TenantID:     artifact.TenantID,
		ProductID:    artifact.ProductID,
		Name:         artifact.Name,
		Version:      artifact.Version,
		FileName:     artifact.FileName,
		URL:          artifact.URL,
		Checksum:     artifact.Checksum,
		ChecksumType: artifact.ChecksumType,
		SizeBytes:    artifact.SizeBytes,
		Notes:        artifact.Notes,
		Metadata:     cloneStringMap(artifact.Metadata),
		CreatedAt:    artifact.CreatedAt,
		UpdatedAt:    artifact.UpdatedAt,
	}
}

func cloneOTACampaign(campaign model.OTACampaign) model.OTACampaign {
	result := model.OTACampaign{
		ID:              campaign.ID,
		TenantID:        campaign.TenantID,
		Name:            campaign.Name,
		FirmwareID:      campaign.FirmwareID,
		ProductID:       campaign.ProductID,
		GroupID:         campaign.GroupID,
		DeviceID:        campaign.DeviceID,
		Status:          campaign.Status,
		TotalDevices:    campaign.TotalDevices,
		DispatchedCount: campaign.DispatchedCount,
		AckedCount:      campaign.AckedCount,
		FailedCount:     campaign.FailedCount,
		CreatedAt:       campaign.CreatedAt,
		UpdatedAt:       campaign.UpdatedAt,
	}
	if campaign.LastDispatchedAt != nil {
		value := *campaign.LastDispatchedAt
		result.LastDispatchedAt = &value
	}
	return result
}
