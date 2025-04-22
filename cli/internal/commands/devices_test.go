package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCreateDeviceCommands(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()
	cfg.DeviceID = "test-device-id"
	cfg.DeviceName = "Test Device"

	// Criar os comandos
	cmds := CreateDeviceCommands(cfg)

	// Verificar se criou pelo menos um comando
	assert.Greater(t, len(cmds), 0)

	// Obter o comando pai (devices)
	cmd := cmds[0]
	assert.Equal(t, "devices", cmd.Use)

	// Verificar se os subcomandos existem
	subCmds := cmd.Commands()

	// Filtramos os comandos de ajuda e completions que são adicionados automaticamente
	var actualCmds []*cobra.Command
	for _, c := range subCmds {
		if c.Use != "help" && !strings.HasPrefix(c.Use, "completion") {
			actualCmds = append(actualCmds, c)
		}
	}

	// Deve haver pelo menos 4 subcomandos principais (list, unlink, rename, info)
	assert.GreaterOrEqual(t, len(actualCmds), 4)

	// Verificar os nomes dos subcomandos
	cmdNames := make(map[string]bool)
	for _, c := range actualCmds {
		cmdNames[c.Use] = true
	}

	assert.True(t, cmdNames["list"])
	assert.True(t, cmdNames["unlink <device-id>"])
	assert.True(t, cmdNames["rename <new-name>"])
	assert.True(t, cmdNames["info [device-id]"])
}

func TestDeviceListCommand(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()
	cfg.DeviceID = "test-device-id"
	cfg.DeviceName = "Test Device"

	// Criar os comandos
	cmds := CreateDeviceCommands(cfg)
	rootCmd := cmds[0]

	// Encontrar o comando list
	var listCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "list" {
			listCmd = c
			break
		}
	}

	assert.NotNil(t, listCmd)

	// Em vez de executar o comando, chamamos a função diretamente
	// O código abaixo é equivalente a executar o comando mas evita problemas com redirecionamento
	if listCmd.RunE != nil {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		listCmd.RunE(listCmd, []string{})

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "Connected Devices")
		assert.Contains(t, output, cfg.DeviceID)
		assert.Contains(t, output, cfg.DeviceName)
	} else {
		t.Skip("Skipping test: RunE function is not defined for list command")
	}
}

func TestDeviceInfoCommand(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()
	cfg.DeviceID = "test-device-id"
	cfg.DeviceName = "Test Device"
	cfg.StorageProvider = "minio"

	// Adicionar uma pasta de sincronização para testes
	cfg.SyncFolders = []config.SyncFolder{
		{
			ID:      "folder-1",
			Path:    "/test/path1",
			Enabled: true,
		},
	}

	// Criar os comandos
	cmds := CreateDeviceCommands(cfg)
	rootCmd := cmds[0]

	// Encontrar o comando info
	var infoCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "info [device-id]" {
			infoCmd = c
			break
		}
	}

	assert.NotNil(t, infoCmd)

	// Em vez de executar o comando, chamamos a função diretamente
	if infoCmd.RunE != nil {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		infoCmd.RunE(infoCmd, []string{})

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "Device Information")
		assert.Contains(t, output, cfg.DeviceID)
		assert.Contains(t, output, cfg.DeviceName)
		assert.Contains(t, output, cfg.StorageProvider)
		assert.Contains(t, output, "Synced Folders")
		assert.Contains(t, output, "folder-1")
	} else {
		t.Skip("Skipping test: RunE function is not defined for info command")
	}
}

func TestDeviceRenameCommand(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()
	cfg.DeviceID = "test-device-id"
	cfg.DeviceName = "Original Name"

	// Criar os comandos
	cmds := CreateDeviceCommands(cfg)
	rootCmd := cmds[0]

	// Encontrar o comando rename
	var renameCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "rename <new-name>" {
			renameCmd = c
			break
		}
	}

	assert.NotNil(t, renameCmd)

	// Em vez de executar o comando, chamamos a função diretamente
	if renameCmd.RunE != nil {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		renameCmd.RunE(renameCmd, []string{"New Device Name"})

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Equal(t, "New Device Name", cfg.DeviceName)
		assert.Contains(t, output, "Device renamed from 'Original Name' to 'New Device Name'")
	} else {
		t.Skip("Skipping test: RunE function is not defined for rename command")
	}
}

func TestDeviceUnlinkCommand(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()
	cfg.DeviceID = "test-device-id"

	// Criar os comandos
	cmds := CreateDeviceCommands(cfg)
	rootCmd := cmds[0]

	// Encontrar o comando unlink
	var unlinkCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "unlink <device-id>" {
			unlinkCmd = c
			break
		}
	}

	assert.NotNil(t, unlinkCmd)

	// Testar tentativa de desconectar o dispositivo atual (deve falhar)
	if unlinkCmd.RunE != nil {
		// Nenhuma redireção de saída necessária para este caso de erro
		err := unlinkCmd.RunE(unlinkCmd, []string{"test-device-id"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unlink the current device")
	} else {
		t.Skip("Skipping test: RunE function is not defined for unlink command")
	}

	// O teste de desconectar outro dispositivo não é possível pois exige entrada do usuário
}
