package profiles

import (
	"nexus-core/internal/config"
)

// Manager gestiona la carga y selección de perfiles de agente.
type Manager struct {
	Profiles map[string]config.ProfileConfig
	ActiveID string
}

// NewManager crea un nuevo gestor de perfiles a partir de la configuración.
func NewManager(cfg config.ProfilesConfig) *Manager {
	m := &Manager{
		Profiles: make(map[string]config.ProfileConfig),
		ActiveID: cfg.ActiveProfile,
	}

	for _, p := range cfg.List {
		if p.Enabled {
			m.Profiles[p.ID] = p
		}
	}

	return m
}

// GetProfile recupera un perfil por su ID.
func (m *Manager) GetProfile(id string) (config.ProfileConfig, bool) {
	p, ok := m.Profiles[id]
	return p, ok
}

// GetActiveProfile retorna el perfil configurado como activo.
func (m *Manager) GetActiveProfile() config.ProfileConfig {
	if p, ok := m.Profiles[m.ActiveID]; ok {
		return p
	}

	// Fallback: si no hay perfil activo o es inválido, buscar el primero habilitado
	for _, p := range m.Profiles {
		return p
	}

	// Último recurso: un perfil genérico básico
	return config.ProfileConfig{
		ID:   "default",
		Name: "Nexus General",
		Role: "general",
	}
}

// HasSkill verifica si un perfil tiene una habilidad específica habilitada.
func HasSkill(p config.ProfileConfig, skillName string) bool {
	for _, s := range p.Skills {
		if s == skillName {
			return true
		}
	}
	return false
}
