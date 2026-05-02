package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// ─────────────────────────────────────────────
// MODELO
// ─────────────────────────────────────────────

type Config struct {
	KeyID        string
	AppKey       string
	Region       string
	Endpoint     string
	CustomDomain string
}

// ─────────────────────────────────────────────
// ARQUIVO DE CREDENCIAIS
// ─────────────────────────────────────────────

func credFilePath() string {
	const fileName = "_b2-credentials.not-share-this-file"
	exe, err := os.Executable()
	if err != nil {
		return fileName
	}
	dir := filepath.Dir(exe)
	// go run compila em /tmp/go-build.../; nesse caso usa o diretório atual
	if strings.Contains(dir, "go-build") {
		cwd, err := os.Getwd()
		if err != nil {
			return fileName
		}
		return filepath.Join(cwd, fileName)
	}
	return filepath.Join(dir, fileName)
}

// ─────────────────────────────────────────────
// PERSISTÊNCIA
// ─────────────────────────────────────────────

func salvarCredenciais(cfg Config) {
	payload := map[string]string{
		"key_id":        cfg.KeyID,
		"app_key":       cfg.AppKey,
		"region":        cfg.Region,
		"endpoint":      cfg.Endpoint,
		"custom_domain": cfg.CustomDomain,
	}
	raw, _ := json.Marshal(payload)
	defer zeroizar(raw)

	enc, err := criptografar(raw)
	if err != nil {
		fmt.Println(yellow.Render("  ⚠  Não foi possível salvar credenciais: ") + err.Error())
		return
	}

	// Garante permissões corretas mesmo se o arquivo já existir
	path := credFilePath()
	if err := os.WriteFile(path, enc, 0600); err != nil {
		fmt.Println(yellow.Render("  ⚠  Não foi possível salvar credenciais: ") + err.Error())
		return
	}
	// Dupla verificação de permissões — defesa contra umask incorreto
	if err := os.Chmod(path, 0600); err != nil {
		fmt.Println(yellow.Render("  ⚠  Não foi possível ajustar permissões: ") + err.Error())
	}

	fmt.Println(dim.Render("  💾 Credenciais salvas em " + path))
}

func carregarCredenciais() (Config, bool) {
	path := credFilePath()

	// Verifica permissões antes de ler — avisa se o arquivo estiver exposto
	if info, err := os.Stat(path); err == nil {
		if mode := info.Mode().Perm(); mode&0o077 != 0 {
			fmt.Println(yellow.Render(fmt.Sprintf(
				"  ⚠  Permissões inseguras no arquivo de credenciais (%s) — corrija com: chmod 600 %s",
				mode, path,
			)))
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, false
	}

	raw, err := descriptografar(data)
	if err != nil {
		fmt.Println(yellow.Render("  ⚠  " + err.Error() + " — recrie as credenciais."))
		return Config{}, false
	}
	defer zeroizar(raw)

	var payload map[string]string
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Config{}, false
	}
	return Config{
		KeyID:        payload["key_id"],
		AppKey:       payload["app_key"],
		Region:       payload["region"],
		Endpoint:     payload["endpoint"],
		CustomDomain: payload["custom_domain"],
	}, true
}

// ─────────────────────────────────────────────
// CARREGAMENTO PRINCIPAL
// ─────────────────────────────────────────────

func carregarConfig() Config {
	// 1. Tenta .env
	_ = godotenv.Load()

	cfg := Config{
		KeyID:        os.Getenv("B2_KEY_ID"),
		AppKey:       os.Getenv("B2_APP_KEY"),
		Region:       os.Getenv("B2_REGION"),
		Endpoint:     os.Getenv("B2_ENDPOINT"),
		CustomDomain: os.Getenv("CUSTOM_DOMAIN"),
	}

	if cfg.KeyID != "" && cfg.AppKey != "" {
		if cfg.Region == "" {
			cfg.Region = prompt("Região B2 (ex: us-west-004)")
		}
		if cfg.Endpoint == "" {
			cfg.Endpoint = fmt.Sprintf("https://s3.%s.backblazeb2.com", cfg.Region)
		}
		return cfg
	}

	// 2. Tenta arquivo de credenciais criptografado
	if saved, ok := carregarCredenciais(); ok {
		fmt.Println(dim.Render("  🔑 Usando credenciais salvas de " + credFilePath()))
		return saved
	}

	// 3. Pede manualmente e salva
	fmt.Println(panelStyle.Render(
		yellow.Render("⚠  Credenciais não encontradas\n") +
			"Informe manualmente (serão salvas de forma criptografada):\n",
	))
	cfg.KeyID = prompt("Key ID")
	cfg.AppKey = promptSecret("Application Key")
	cfg.Region = prompt("Região B2 (ex: us-west-004)")
	if cfg.Endpoint == "" {
		cfg.Endpoint = fmt.Sprintf("https://s3.%s.backblazeb2.com", cfg.Region)
	}
	cfg.CustomDomain = prompt("Custom Domain (enter para pular)")

	salvarCredenciais(cfg)
	return cfg
}