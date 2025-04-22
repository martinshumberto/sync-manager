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

func TestCreateFolderCommands(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()

	// Mock da função de salvamento
	saveCount := 0
	saveFn := func() error {
		saveCount++
		return nil
	}

	// Criar os comandos
	cmds := CreateFolderCommands(cfg, saveFn)

	// Verificar se criou pelo menos os 5 comandos esperados
	assert.Equal(t, 5, len(cmds))

	// Verificar os nomes dos comandos
	cmdNames := make(map[string]bool)
	for _, c := range cmds {
		cmdNames[c.Use] = true
	}

	// Verificar se todos os comandos esperados existem
	assert.True(t, cmdNames["add-folder [path]"])
	assert.True(t, cmdNames["list-folders"])
	assert.True(t, cmdNames["remove-folder [folder-id]"])
	assert.True(t, cmdNames["enable-folder [folder-id]"])
	assert.True(t, cmdNames["disable-folder [folder-id]"])
}

func TestFolderListCommand(t *testing.T) {
	// Preparar uma configuração de teste com pastas
	cfg := config.DefaultConfig()
	cfg.SyncFolders = []config.SyncFolder{
		{
			ID:         "folder-1",
			Path:       "/test/path1",
			Enabled:    true,
			Exclude:    []string{".git"},
			TwoWaySync: true,
		},
		{
			ID:         "folder-2",
			Path:       "/test/path2",
			Enabled:    false,
			Exclude:    []string{".temp"},
			TwoWaySync: false,
		},
	}

	saveFn := func() error { return nil }

	// Criar os comandos
	cmds := CreateFolderCommands(cfg, saveFn)

	// Encontrar o comando list-folders
	var listCmd *cobra.Command
	for _, c := range cmds {
		if c.Use == "list-folders" {
			listCmd = c
			break
		}
	}

	assert.NotNil(t, listCmd)

	// Redirecionando saída para captura
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Executar função RunE diretamente
	err := listCmd.RunE(listCmd, []string{})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verificar se a saída contém informações sobre as pastas
	assert.Contains(t, output, "folder-1")
	assert.Contains(t, output, "/test/path1")
	assert.Contains(t, output, "folder-2")
	assert.Contains(t, output, "/test/path2")
}

func TestFolderAddCommand(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()

	// Mock da função de salvamento
	saveCount := 0
	saveFn := func() error {
		saveCount++
		return nil
	}

	// Criar os comandos
	cmds := CreateFolderCommands(cfg, saveFn)

	// Encontrar o comando add-folder
	var addCmd *cobra.Command
	for _, c := range cmds {
		if c.Use == "add-folder [path]" {
			addCmd = c
			break
		}
	}

	assert.NotNil(t, addCmd)

	// Não podemos testar a execução completa pois ela depende de interação do sistema de arquivos,
	// mas podemos verificar se o comando está configurado corretamente
	assert.NotNil(t, addCmd.RunE)

	// Verificar se os flags estão configurados
	excludeFlag := addCmd.Flag("exclude")
	assert.NotNil(t, excludeFlag)

	disableFlag := addCmd.Flag("disable")
	assert.Nil(t, disableFlag) // Este flag não existe na implementação atual

	// Verificar flags que realmente existem
	twoWayFlag := addCmd.Flag("two-way")
	assert.NotNil(t, twoWayFlag)

	nameFlag := addCmd.Flag("name")
	assert.NotNil(t, nameFlag)
}

func TestFolderRemoveCommand(t *testing.T) {
	// Preparar uma configuração de teste com uma pasta
	cfg := config.DefaultConfig()
	cfg.SyncFolders = []config.SyncFolder{
		{
			ID:      "folder-to-remove",
			Path:    "/test/path-to-remove",
			Enabled: true,
		},
	}

	// Mock da função de salvamento
	saveCount := 0
	saveFn := func() error {
		saveCount++
		return nil
	}

	// Criar os comandos
	cmds := CreateFolderCommands(cfg, saveFn)

	// Encontrar o comando remove-folder
	var removeCmd *cobra.Command
	for _, c := range cmds {
		if c.Use == "remove-folder [folder-id]" {
			removeCmd = c
			break
		}
	}

	assert.NotNil(t, removeCmd)

	// Sem confirmação de usuário no código atual, podemos testar a execução direta
	err := removeCmd.RunE(removeCmd, []string{"folder-to-remove"})
	assert.NoError(t, err)

	// Verificar se a pasta foi removida
	assert.Equal(t, 0, len(cfg.SyncFolders))

	// Verificar se a função de salvamento foi chamada
	assert.Equal(t, 1, saveCount)
}

// Testa o comando para habilitar uma pasta
func TestFolderEnableCommand(t *testing.T) {
	// Preparar uma configuração de teste com uma pasta desabilitada
	cfg := config.DefaultConfig()
	cfg.SyncFolders = []config.SyncFolder{
		{
			ID:      "folder-to-enable",
			Path:    "/test/path-to-enable",
			Enabled: false,
		},
	}

	// Mock da função de salvamento
	saveCount := 0
	saveFn := func() error {
		saveCount++
		return nil
	}

	// Criar os comandos
	cmds := CreateFolderCommands(cfg, saveFn)

	// Encontrar o comando enable-folder
	var enableCmd *cobra.Command
	for _, c := range cmds {
		if c.Use == "enable-folder [folder-id]" {
			enableCmd = c
			break
		}
	}

	assert.NotNil(t, enableCmd)

	// Executar o comando
	err := enableCmd.RunE(enableCmd, []string{"folder-to-enable"})
	assert.NoError(t, err)

	// Verificar se a pasta foi habilitada
	assert.True(t, cfg.SyncFolders[0].Enabled)

	// Verificar se a função de salvamento foi chamada
	assert.Equal(t, 1, saveCount)
}

// Testa o comando para desabilitar uma pasta
func TestFolderDisableCommand(t *testing.T) {
	// Preparar uma configuração de teste com uma pasta habilitada
	cfg := config.DefaultConfig()
	cfg.SyncFolders = []config.SyncFolder{
		{
			ID:      "folder-to-disable",
			Path:    "/test/path-to-disable",
			Enabled: true,
		},
	}

	// Mock da função de salvamento
	saveCount := 0
	saveFn := func() error {
		saveCount++
		return nil
	}

	// Criar os comandos
	cmds := CreateFolderCommands(cfg, saveFn)

	// Encontrar o comando disable-folder
	var disableCmd *cobra.Command
	for _, c := range cmds {
		if c.Use == "disable-folder [folder-id]" {
			disableCmd = c
			break
		}
	}

	assert.NotNil(t, disableCmd)

	// Executar o comando
	err := disableCmd.RunE(disableCmd, []string{"folder-to-disable"})
	assert.NoError(t, err)

	// Verificar se a pasta foi desabilitada
	assert.False(t, cfg.SyncFolders[0].Enabled)

	// Verificar se a função de salvamento foi chamada
	assert.Equal(t, 1, saveCount)
}
