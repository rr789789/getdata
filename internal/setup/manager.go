package setup

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type State struct {
	InstallLock       bool      `json:"install_lock"`
	AppName           string    `json:"app_name"`
	SiteURL           string    `json:"site_url,omitempty"`
	AdminUsername     string    `json:"admin_username,omitempty"`
	AdminEmail        string    `json:"admin_email,omitempty"`
	DefaultTenantName string    `json:"default_tenant_name,omitempty"`
	DefaultTenantSlug string    `json:"default_tenant_slug,omitempty"`
	InstalledAt       time.Time `json:"installed_at,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

type BootstrapRequest struct {
	AppName           string `json:"app_name"`
	SiteURL           string `json:"site_url"`
	AdminUsername     string `json:"admin_username"`
	AdminEmail        string `json:"admin_email"`
	DefaultTenantName string `json:"default_tenant_name"`
	DefaultTenantSlug string `json:"default_tenant_slug"`
}

type Manager struct {
	path  string
	mu    sync.RWMutex
	state State

	afterPersist func(context.Context, []byte)
}

func NewManager(path string) (*Manager, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("setup state path is required")
	}

	manager := &Manager{path: path}
	if err := manager.load(); err != nil {
		return nil, err
	}
	return manager, nil
}

func (m *Manager) Path() string {
	return m.path
}

func (m *Manager) Status() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *Manager) Installed() bool {
	return m.Status().InstallLock
}

func (m *Manager) SetAfterPersistHook(hook func(context.Context, []byte)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.afterPersist = hook
}

func (m *Manager) Bootstrap(request BootstrapRequest) (State, error) {
	request.AppName = strings.TrimSpace(request.AppName)
	request.SiteURL = normalizeURL(request.SiteURL)
	request.AdminUsername = strings.TrimSpace(request.AdminUsername)
	request.AdminEmail = strings.TrimSpace(request.AdminEmail)
	request.DefaultTenantName = strings.TrimSpace(request.DefaultTenantName)
	request.DefaultTenantSlug = strings.TrimSpace(request.DefaultTenantSlug)

	if request.AppName == "" {
		return State{}, errors.New("app_name is required")
	}
	if request.AdminUsername == "" {
		return State{}, errors.New("admin_username is required")
	}

	now := time.Now().UTC()
	state := State{
		InstallLock:       true,
		AppName:           request.AppName,
		SiteURL:           request.SiteURL,
		AdminUsername:     request.AdminUsername,
		AdminEmail:        request.AdminEmail,
		DefaultTenantName: request.DefaultTenantName,
		DefaultTenantSlug: request.DefaultTenantSlug,
		InstalledAt:       now,
		UpdatedAt:         now,
	}

	m.mu.Lock()
	if m.state.InstallLock {
		current := m.state
		m.mu.Unlock()
		return current, nil
	}
	data, err := marshalState(state)
	if err != nil {
		m.mu.Unlock()
		return State{}, err
	}
	if err := m.writeLocked(data); err != nil {
		m.mu.Unlock()
		return State{}, err
	}
	m.state = state
	hook := m.afterPersist
	m.mu.Unlock()

	if hook != nil {
		snapshot := append([]byte(nil), data...)
		go hook(context.Background(), snapshot)
	}
	return state, nil
}

func (m *Manager) ApplyReplicaState(data []byte) error {
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.writeLocked(data); err != nil {
		return err
	}
	m.state = state
	return nil
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	m.mu.Lock()
	m.state = state
	m.mu.Unlock()
	return nil
}

func (m *Manager) writeLocked(data []byte) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(m.path), "mvp-setup-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	_ = os.Remove(m.path)
	if err := os.Rename(tmpPath, m.path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func normalizeURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.TrimRight(value, "/")
}

func marshalState(state State) ([]byte, error) {
	return json.MarshalIndent(state, "", "  ")
}
