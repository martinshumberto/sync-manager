package commands

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCreateSyncCommands(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()

	// Criar os comandos
	cmds := CreateSyncCommands(cfg)

	// Verificar se criou pelo menos 4 comandos
	assert.Equal(t, 4, len(cmds))

	// Verificar os nomes dos comandos
	cmdNames := make(map[string]bool)
	for _, c := range cmds {
		cmdNames[c.Use] = true
	}

	assert.True(t, cmdNames["sync"])
	assert.True(t, cmdNames["sync-folder <path>"])
	assert.True(t, cmdNames["pause"])
	assert.True(t, cmdNames["resume"])
}

func TestSyncCommand(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()

	// Adicionar uma pasta de sincronização para o teste
	cfg.SyncFolders = []config.SyncFolder{
		{
			ID:      "folder-1",
			Path:    "/test/path",
			Enabled: true,
		},
	}

	// Criar os comandos
	cmds := CreateSyncCommands(cfg)

	// Encontrar o comando sync
	var syncCmd *cobra.Command
	for _, c := range cmds {
		if c.Use == "sync" {
			syncCmd = c
			break
		}
	}

	assert.NotNil(t, syncCmd)

	// Redirecionando saída para captura
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Executar comando
	err := syncCmd.RunE(syncCmd, []string{})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verificar mensagens do comando
	assert.Contains(t, output, "Initiating synchronization")
	assert.Contains(t, output, "/test/path")
	assert.Contains(t, output, "Synchronization complete")
}

func TestSyncFolderCommand(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()

	// Adicionar uma pasta de sincronização para o teste
	cfg.SyncFolders = []config.SyncFolder{
		{
			ID:      "folder-1",
			Path:    "/test/path",
			Enabled: true,
		},
	}

	// Criar os comandos
	cmds := CreateSyncCommands(cfg)

	// Encontrar o comando sync-folder
	var syncFolderCmd *cobra.Command
	for _, c := range cmds {
		if c.Use == "sync-folder <path>" {
			syncFolderCmd = c
			break
		}
	}

	assert.NotNil(t, syncFolderCmd)

	// Redirecionando saída para captura
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Executar comando
	err := syncFolderCmd.RunE(syncFolderCmd, []string{"/test/path"})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verificar mensagens do comando
	assert.Contains(t, output, "Synchronizing folder")
	assert.Contains(t, output, "/test/path")
	assert.Contains(t, output, "Folder synchronization complete")
}

func TestPauseCommand(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()

	// Criar os comandos
	cmds := CreateSyncCommands(cfg)

	// Encontrar o comando pause
	var pauseCmd *cobra.Command
	for _, c := range cmds {
		if c.Use == "pause" {
			pauseCmd = c
			break
		}
	}

	assert.NotNil(t, pauseCmd)

	// Redirecionando saída para captura
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Executar comando
	err := pauseCmd.RunE(pauseCmd, []string{})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verificar mensagens do comando
	assert.Contains(t, output, "Synchronization paused")
}

func TestResumeCommand(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()

	// Criar os comandos
	cmds := CreateSyncCommands(cfg)

	// Encontrar o comando resume
	var resumeCmd *cobra.Command
	for _, c := range cmds {
		if c.Use == "resume" {
			resumeCmd = c
			break
		}
	}

	assert.NotNil(t, resumeCmd)

	// Redirecionando saída para captura
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Executar comando
	err := resumeCmd.RunE(resumeCmd, []string{})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verificar mensagens do comando
	assert.Contains(t, output, "Synchronization resumed")
}
