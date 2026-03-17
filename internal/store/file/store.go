package file

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mvp-platform/internal/model"
	"mvp-platform/internal/store/memory"
)

type Store struct {
	path  string
	inner *memory.Store

	mutateMu        sync.Mutex
	persistMu       sync.Mutex
	lastPersistedAt time.Time
	persistErrors   int64
	afterPersist    func(context.Context, []byte)
}

type persistedSnapshot struct {
	Version   int              `json:"version"`
	SavedAt   time.Time        `json:"saved_at"`
	Snapshot  memory.Snapshot  `json:"snapshot"`
}

func New(path string, telemetryRetention int) (*Store, error) {
	if path == "" {
		return nil, errors.New("persistence path is required")
	}

	inner := memory.New(telemetryRetention)
	store := &Store{
		path:  path,
		inner: inner,
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) BackendName() string {
	return "file"
}

func (s *Store) PersistencePath() string {
	return s.path
}

func (s *Store) StorageStats(ctx context.Context) (model.StorageStats, error) {
	stats, err := s.inner.StorageStats(ctx)
	if err != nil {
		return model.StorageStats{}, err
	}
	s.persistMu.Lock()
	lastPersistedAt := s.lastPersistedAt
	persistErrors := s.persistErrors
	s.persistMu.Unlock()
	stats.Backend = s.BackendName()
	stats.PersistencePath = s.path
	if !lastPersistedAt.IsZero() {
		value := lastPersistedAt
		stats.LastPersistedAt = &value
	}
	stats.PersistErrors = persistErrors
	return stats, nil
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	snapshot, err := decodeSnapshot(data)
	if err != nil {
		return err
	}

	s.inner.Restore(snapshot.Snapshot)
	s.persistMu.Lock()
	s.lastPersistedAt = snapshot.SavedAt.UTC()
	s.persistMu.Unlock()
	return nil
}

func (s *Store) SetAfterPersistHook(hook func(context.Context, []byte)) {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.afterPersist = hook
}

func (s *Store) ApplyReplicaSnapshot(data []byte) error {
	snapshot, err := decodeSnapshot(data)
	if err != nil {
		return err
	}

	s.mutateMu.Lock()
	defer s.mutateMu.Unlock()

	s.persistMu.Lock()
	err = s.writeSnapshotFile(data, snapshot.SavedAt)
	s.persistMu.Unlock()
	if err != nil {
		return err
	}
	s.inner.Restore(snapshot.Snapshot)
	return nil
}

func (s *Store) persist() ([]byte, error) {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()

	snapshot := persistedSnapshot{
		Version:  1,
		SavedAt:  time.Now().UTC(),
		Snapshot: s.inner.Snapshot(),
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		s.persistErrors++
		return nil, err
	}

	if err := s.writeSnapshotFile(data, snapshot.SavedAt); err != nil {
		return nil, err
	}

	return data, nil
}

func (s *Store) writeSnapshotFile(data []byte, savedAt time.Time) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		s.persistErrors++
		return err
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(s.path), "mvp-store-*.tmp")
	if err != nil {
		s.persistErrors++
		return err
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		s.persistErrors++
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		s.persistErrors++
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		s.persistErrors++
		return err
	}

	_ = os.Remove(s.path)
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		s.persistErrors++
		return err
	}

	s.lastPersistedAt = savedAt.UTC()
	return nil
}

func (s *Store) mutate(apply func() error) error {
	s.mutateMu.Lock()
	defer s.mutateMu.Unlock()

	rollback := s.inner.Snapshot()
	if err := apply(); err != nil {
		return err
	}
	data, err := s.persist()
	if err != nil {
		s.inner.Restore(rollback)
		return err
	}
	s.persistMu.Lock()
	hook := s.afterPersist
	s.persistMu.Unlock()
	if hook != nil {
		snapshot := append([]byte(nil), data...)
		go hook(context.Background(), snapshot)
	}
	return nil
}

func decodeSnapshot(data []byte) (persistedSnapshot, error) {
	var snapshot persistedSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return persistedSnapshot{}, err
	}
	return snapshot, nil
}

func (s *Store) CreateTenant(ctx context.Context, tenant model.Tenant) error {
	return s.mutate(func() error {
		return s.inner.CreateTenant(ctx, tenant)
	})
}

func (s *Store) GetTenant(ctx context.Context, tenantID string) (model.Tenant, error) {
	return s.inner.GetTenant(ctx, tenantID)
}

func (s *Store) ListTenants(ctx context.Context) ([]model.Tenant, error) {
	return s.inner.ListTenants(ctx)
}

func (s *Store) SaveTenant(ctx context.Context, tenant model.Tenant) error {
	return s.mutate(func() error {
		return s.inner.SaveTenant(ctx, tenant)
	})
}

func (s *Store) CreateProduct(ctx context.Context, product model.Product) error {
	return s.mutate(func() error {
		return s.inner.CreateProduct(ctx, product)
	})
}

func (s *Store) GetProduct(ctx context.Context, productID string) (model.Product, error) {
	return s.inner.GetProduct(ctx, productID)
}

func (s *Store) ListProducts(ctx context.Context) ([]model.Product, error) {
	return s.inner.ListProducts(ctx)
}

func (s *Store) SaveProduct(ctx context.Context, product model.Product) error {
	return s.mutate(func() error {
		return s.inner.SaveProduct(ctx, product)
	})
}

func (s *Store) CreateDevice(ctx context.Context, device model.Device) error {
	return s.mutate(func() error {
		return s.inner.CreateDevice(ctx, device)
	})
}

func (s *Store) GetDevice(ctx context.Context, deviceID string) (model.Device, error) {
	return s.inner.GetDevice(ctx, deviceID)
}

func (s *Store) ListDevices(ctx context.Context) ([]model.Device, error) {
	return s.inner.ListDevices(ctx)
}

func (s *Store) SaveDevice(ctx context.Context, device model.Device) error {
	return s.mutate(func() error {
		return s.inner.SaveDevice(ctx, device)
	})
}

func (s *Store) CreateGroup(ctx context.Context, group model.DeviceGroup) error {
	return s.mutate(func() error {
		return s.inner.CreateGroup(ctx, group)
	})
}

func (s *Store) GetGroup(ctx context.Context, groupID string) (model.DeviceGroup, error) {
	return s.inner.GetGroup(ctx, groupID)
}

func (s *Store) ListGroups(ctx context.Context) ([]model.DeviceGroup, error) {
	return s.inner.ListGroups(ctx)
}

func (s *Store) SaveGroup(ctx context.Context, group model.DeviceGroup) error {
	return s.mutate(func() error {
		return s.inner.SaveGroup(ctx, group)
	})
}

func (s *Store) AddDeviceToGroup(ctx context.Context, groupID, deviceID string) error {
	return s.mutate(func() error {
		return s.inner.AddDeviceToGroup(ctx, groupID, deviceID)
	})
}

func (s *Store) RemoveDeviceFromGroup(ctx context.Context, groupID, deviceID string) error {
	return s.mutate(func() error {
		return s.inner.RemoveDeviceFromGroup(ctx, groupID, deviceID)
	})
}

func (s *Store) ListGroupIDsByDevice(ctx context.Context, deviceID string) ([]string, error) {
	return s.inner.ListGroupIDsByDevice(ctx, deviceID)
}

func (s *Store) ListDeviceIDsByGroup(ctx context.Context, groupID string) ([]string, error) {
	return s.inner.ListDeviceIDsByGroup(ctx, groupID)
}

func (s *Store) CreateRule(ctx context.Context, rule model.Rule) error {
	return s.mutate(func() error {
		return s.inner.CreateRule(ctx, rule)
	})
}

func (s *Store) GetRule(ctx context.Context, ruleID string) (model.Rule, error) {
	return s.inner.GetRule(ctx, ruleID)
}

func (s *Store) ListRules(ctx context.Context) ([]model.Rule, error) {
	return s.inner.ListRules(ctx)
}

func (s *Store) SaveRule(ctx context.Context, rule model.Rule) error {
	return s.mutate(func() error {
		return s.inner.SaveRule(ctx, rule)
	})
}

func (s *Store) CreateConfigProfile(ctx context.Context, profile model.ConfigProfile) error {
	return s.mutate(func() error {
		return s.inner.CreateConfigProfile(ctx, profile)
	})
}

func (s *Store) GetConfigProfile(ctx context.Context, profileID string) (model.ConfigProfile, error) {
	return s.inner.GetConfigProfile(ctx, profileID)
}

func (s *Store) ListConfigProfiles(ctx context.Context) ([]model.ConfigProfile, error) {
	return s.inner.ListConfigProfiles(ctx)
}

func (s *Store) SaveConfigProfile(ctx context.Context, profile model.ConfigProfile) error {
	return s.mutate(func() error {
		return s.inner.SaveConfigProfile(ctx, profile)
	})
}

func (s *Store) CreateFirmwareArtifact(ctx context.Context, artifact model.FirmwareArtifact) error {
	return s.mutate(func() error {
		return s.inner.CreateFirmwareArtifact(ctx, artifact)
	})
}

func (s *Store) GetFirmwareArtifact(ctx context.Context, artifactID string) (model.FirmwareArtifact, error) {
	return s.inner.GetFirmwareArtifact(ctx, artifactID)
}

func (s *Store) ListFirmwareArtifacts(ctx context.Context) ([]model.FirmwareArtifact, error) {
	return s.inner.ListFirmwareArtifacts(ctx)
}

func (s *Store) SaveFirmwareArtifact(ctx context.Context, artifact model.FirmwareArtifact) error {
	return s.mutate(func() error {
		return s.inner.SaveFirmwareArtifact(ctx, artifact)
	})
}

func (s *Store) CreateOTACampaign(ctx context.Context, campaign model.OTACampaign) error {
	return s.mutate(func() error {
		return s.inner.CreateOTACampaign(ctx, campaign)
	})
}

func (s *Store) GetOTACampaign(ctx context.Context, campaignID string) (model.OTACampaign, error) {
	return s.inner.GetOTACampaign(ctx, campaignID)
}

func (s *Store) ListOTACampaigns(ctx context.Context) ([]model.OTACampaign, error) {
	return s.inner.ListOTACampaigns(ctx)
}

func (s *Store) SaveOTACampaign(ctx context.Context, campaign model.OTACampaign) error {
	return s.mutate(func() error {
		return s.inner.SaveOTACampaign(ctx, campaign)
	})
}

func (s *Store) AppendTelemetry(ctx context.Context, telemetry model.Telemetry) error {
	return s.mutate(func() error {
		return s.inner.AppendTelemetry(ctx, telemetry)
	})
}

func (s *Store) ListTelemetryByDevice(ctx context.Context, deviceID string, limit int) ([]model.Telemetry, error) {
	return s.inner.ListTelemetryByDevice(ctx, deviceID, limit)
}

func (s *Store) SaveShadow(ctx context.Context, shadow model.DeviceShadow) error {
	return s.mutate(func() error {
		return s.inner.SaveShadow(ctx, shadow)
	})
}

func (s *Store) GetShadow(ctx context.Context, deviceID string) (model.DeviceShadow, error) {
	return s.inner.GetShadow(ctx, deviceID)
}

func (s *Store) SaveCommand(ctx context.Context, command model.Command) error {
	return s.mutate(func() error {
		return s.inner.SaveCommand(ctx, command)
	})
}

func (s *Store) GetCommand(ctx context.Context, commandID string) (model.Command, error) {
	return s.inner.GetCommand(ctx, commandID)
}

func (s *Store) UpdateCommandStatus(ctx context.Context, commandID string, status model.CommandStatus, result string) (model.Command, error) {
	var command model.Command
	err := s.mutate(func() error {
		var updateErr error
		command, updateErr = s.inner.UpdateCommandStatus(ctx, commandID, status, result)
		return updateErr
	})
	if err != nil {
		return model.Command{}, err
	}
	return command, nil
}

func (s *Store) ListCommandsByDevice(ctx context.Context, deviceID string, limit int) ([]model.Command, error) {
	return s.inner.ListCommandsByDevice(ctx, deviceID, limit)
}

func (s *Store) AppendAlert(ctx context.Context, alert model.AlertEvent) error {
	return s.mutate(func() error {
		return s.inner.AppendAlert(ctx, alert)
	})
}

func (s *Store) GetAlert(ctx context.Context, alertID string) (model.AlertEvent, error) {
	return s.inner.GetAlert(ctx, alertID)
}

func (s *Store) SaveAlert(ctx context.Context, alert model.AlertEvent) error {
	return s.mutate(func() error {
		return s.inner.SaveAlert(ctx, alert)
	})
}

func (s *Store) ListAlerts(ctx context.Context, limit int) ([]model.AlertEvent, error) {
	return s.inner.ListAlerts(ctx, limit)
}
