package sync

import (
	"github.com/martinshumberto/sync-manager/agent/internal/config"
	"github.com/martinshumberto/sync-manager/agent/internal/storage"
	"github.com/martinshumberto/sync-manager/agent/internal/syncmanager"
	"github.com/martinshumberto/sync-manager/agent/internal/uploader"
	commonconfig "github.com/martinshumberto/sync-manager/common/config"
)

// Manager é uma interface que simplifica o acesso ao SyncManager
type Manager interface {
	Start() error
	Stop()
}

// ManagerWrapper é um wrapper em torno do SyncManager
type ManagerWrapper struct {
	sm *syncmanager.SyncManager
}

// NewManager cria uma nova instância do gerenciador de sincronização
func NewManager(cfg interface{}, store storage.Storage, uploader *uploader.Uploader) (Manager, error) {
	// Adaptação da configuração para o formato esperado pelo SyncManager
	var internalCfg *config.Config

	// Se for configuração comum, adaptar para configuração interna
	if commonCfg, ok := cfg.(*commonconfig.Config); ok {
		internalCfg = &config.Config{
			Sync: config.SyncConfig{
				IntervalMinutes: int(commonCfg.SyncInterval.Minutes()),
				AutoSync:        true,
			},
			Folders: make(map[string]config.SyncFolder),
		}

		// Converter pastas sincronizadas
		for _, folder := range commonCfg.SyncFolders {
			internalCfg.Folders[folder.ID] = config.SyncFolder{
				LocalPath:       folder.Path,
				RemotePath:      folder.ID, // Usar ID como caminho remoto por padrão
				ExcludePatterns: folder.Exclude,
				Enabled:         folder.Enabled,
			}
		}
	} else if agentCfg, ok := cfg.(*config.Config); ok {
		// Usar a configuração interna diretamente
		internalCfg = agentCfg
	}

	// Criar o SyncManager usando a configuração interna
	sm, err := syncmanager.NewSyncManager(internalCfg)
	if err != nil {
		return nil, err
	}

	return &ManagerWrapper{
		sm: sm,
	}, nil
}

// Start inicia o gerenciador de sincronização
func (m *ManagerWrapper) Start() error {
	return m.sm.Start()
}

// Stop para o gerenciador de sincronização
func (m *ManagerWrapper) Stop() {
	m.sm.Stop()
}
