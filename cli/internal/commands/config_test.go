package commands

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestDisplayConfig(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := &config.Config{
		DeviceID:        "test-device-id",
		DeviceName:      "test-device",
		LogLevel:        "info",
		SyncInterval:    5 * time.Minute,
		MaxConcurrency:  4,
		ThrottleBytes:   0,
		StorageProvider: "minio",
		MinioConfig: config.MinioConfig{
			Endpoint:  "localhost:9000",
			Region:    "us-east-1",
			Bucket:    "test-bucket",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			UseSSL:    false,
		},
	}

	// Capturar a saída
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DisplayConfig(cfg)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verificar se a saída contém informações importantes
	assert.Contains(t, output, "test-device-id")
	assert.Contains(t, output, "test-device")
	assert.Contains(t, output, "minio")
	assert.Contains(t, output, "localhost:9000")
	assert.Contains(t, output, "test-bucket")
}

func TestCreateConfigCommands(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()

	// Mock da função de salvamento
	saveCount := 0
	saveFn := func() error {
		saveCount++
		return nil
	}

	// Criar os comandos
	cmds := CreateConfigCommands(cfg, saveFn)

	// Verificar se criou pelo menos um comando
	assert.Greater(t, len(cmds), 0)

	// Obter o comando pai (config)
	cmd := cmds[0]
	assert.Equal(t, "config", cmd.Use)

	// Verificar se os subcomandos existem
	subCmds := cmd.Commands()

	// Deve haver pelo menos 3 subcomandos (get, set, reset)
	assert.GreaterOrEqual(t, len(subCmds), 3)

	// Verificar os nomes dos subcomandos
	cmdNames := make(map[string]bool)
	for _, c := range subCmds {
		cmdNames[c.Use] = true
	}

	assert.True(t, cmdNames["get [key]"])
	assert.True(t, cmdNames["set <key> <value>"])
	assert.True(t, cmdNames["reset"])
}

func TestConfigGetCommand(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := &config.Config{
		DeviceID:        "test-device-id",
		DeviceName:      "test-device",
		StorageProvider: "minio",
		MinioConfig: config.MinioConfig{
			Bucket: "test-bucket",
		},
	}

	saveFn := func() error { return nil }

	// Criar os comandos
	cmds := CreateConfigCommands(cfg, saveFn)
	rootCmd := cmds[0]

	// Encontrar o comando get
	var getCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "get [key]" {
			getCmd = c
			break
		}
	}

	assert.NotNil(t, getCmd)

	// Redirecionando saída para captura
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Testar o comando get com um parâmetro específico
	err := getCmd.RunE(getCmd, []string{"storage.provider"})
	assert.NoError(t, err)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verificar se a saída contém o valor esperado
	assert.Contains(t, output, "minio")
}

func TestConfigSetCommand(t *testing.T) {
	// Preparar uma configuração de teste
	cfg := config.DefaultConfig()

	// Mock da função de salvamento
	saveCount := 0
	saveFn := func() error {
		saveCount++
		return nil
	}

	// Criar os comandos
	cmds := CreateConfigCommands(cfg, saveFn)
	rootCmd := cmds[0]

	// Encontrar o comando set
	var setCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "set <key> <value>" {
			setCmd = c
			break
		}
	}

	assert.NotNil(t, setCmd)

	// Testar o comando set sem redirecionamento, pois não precisamos verificar o output
	err := setCmd.RunE(setCmd, []string{"storage.provider", "local"})
	assert.NoError(t, err)

	// Verificar se a configuração foi alterada
	assert.Equal(t, "local", cfg.StorageProvider)

	// Verificar se a função de salvamento foi chamada
	assert.Equal(t, 1, saveCount)
}

func TestConfigResetCommand(t *testing.T) {
	// Preparar uma configuração modificada
	cfg := config.DefaultConfig()
	cfg.StorageProvider = "s3" // Alterar do padrão

	// Mock da função de salvamento
	saveCount := 0
	saveFn := func() error {
		saveCount++
		return nil
	}

	// Criar os comandos
	cmds := CreateConfigCommands(cfg, saveFn)
	rootCmd := cmds[0]

	// Encontrar o comando reset
	var resetCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Use == "reset" {
			resetCmd = c
			break
		}
	}

	assert.NotNil(t, resetCmd)

	// Não podemos testar a funcionalidade completa sem simular a entrada
	// do usuário, mas podemos verificar se o código existe
	assert.NotNil(t, resetCmd.RunE)
}
