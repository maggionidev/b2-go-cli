package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

// ─────────────────────────────────────────────
// SEGREDO INTERNO DO BINÁRIO
//
// Não é uma senha — é entropia extra embutida no binário.
// Dificulta ataques genéricos offline mesmo que o atacante
// conheça todo o ambiente: ele ainda precisa deste binário
// ou de engenharia reversa para extrair esses bytes.
//
// Para gerar seus próprios bytes:
//   python3 -c "import secrets; print([hex(b) for b in secrets.token_bytes(32)])"
// ─────────────────────────────────────────────
var internalSecret = []byte{
	0x4b, 0x7e, 0x2a, 0x91, 0xf3, 0x05, 0xc8, 0x6d,
	0x3b, 0xa4, 0x17, 0x8f, 0xe2, 0x59, 0x0c, 0xd6,
	0x72, 0x3e, 0xb1, 0x48, 0x9a, 0xf7, 0x2c, 0x84,
	0x61, 0xd0, 0x5f, 0x13, 0xe8, 0x96, 0x4a, 0x2b,
}

// ─────────────────────────────────────────────
// PARÂMETROS KDF — versionados no envelope
// ─────────────────────────────────────────────

type kdfParams struct {
	Time    uint32 `json:"time"`
	Memory  uint32 `json:"memory"`
	Threads uint8  `json:"threads"`
	KeyLen  uint32 `json:"key_len"`
}

// kdfAtual define os parâmetros usados ao salvar.
// Para endurecer no futuro: ajuste aqui e bump envelopeVersion.
// Na leitura, os parâmetros gravados no envelope são sempre usados
// — não kdfAtual — garantindo compatibilidade retroativa.
var kdfAtual = kdfParams{
	Time:    3,
	Memory:  128 * 1024, // 128 MB — caro para brute-force, ok para CLI desktop
	Threads: 4,
	KeyLen:  32,
}

// ─────────────────────────────────────────────
// ENVELOPE — o que fica gravado em disco
// ─────────────────────────────────────────────

type envelope struct {
	Version    int       `json:"v"`          // versão do formato para upgrades
	KDF        kdfParams `json:"kdf"`        // parâmetros KDF usados — lidos na descriptografia
	Salt       []byte    `json:"salt"`       // salt aleatório de 32 bytes por arquivo
	Nonce      []byte    `json:"nonce"`      // nonce AES-GCM
	Ciphertext []byte    `json:"ciphertext"` // payload cifrado + tag GCM
	CreatedAt  int64     `json:"created_at"` // unix timestamp de criação
}

const envelopeVersion = 3

// ─────────────────────────────────────────────
// COLETA DE ENTROPIA DO AMBIENTE
// ─────────────────────────────────────────────

// coletarSeed reúne material que um atacante não consegue
// reconstituir só com o arquivo de credenciais.
//
// Removido: os.Executable() — mover/renomear o binário invalidaria
// as credenciais sem motivo. O internalSecret cobre esse papel.
func coletarSeed() []byte {
	var b strings.Builder

	hostname, _ := os.Hostname()
	b.WriteString(hostname)
	b.WriteByte(0x00)

	b.WriteString(machineID())
	b.WriteByte(0x00)

	if u, err := user.Current(); err == nil {
		b.WriteString(u.Uid)
		b.WriteByte(0x00)
		b.WriteString(u.Username)
	}
	b.WriteByte(0x00)

	// Segredo embutido — diferencia este binário de um genérico
	b.Write(internalSecret)

	seed := []byte(b.String())
	return seed
}

// machineID retorna um identificador estável da máquina com fallbacks
// para Linux, macOS e Windows.
func machineID() string {
	switch runtime.GOOS {
	case "linux":
		for _, path := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
			if data, err := os.ReadFile(path); err == nil {
				return strings.TrimSpace(string(data))
			}
		}

	case "darwin":
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			return strings.TrimSpace(string(data))
		}

	case "windows":
		// Sem import de registry — usa stat de paths de sistema como fingerprint indireto
		paths := []string{
			`C:\ProgramData\Microsoft\Crypto\RSA\MachineKeys`,
			os.Getenv("SYSTEMROOT") + `\system32\config\systemprofile`,
		}
		for _, p := range paths {
			if info, err := os.Stat(p); err == nil {
				return fmt.Sprintf("win-%d", info.ModTime().UnixNano())
			}
		}
	}

	// Fallback universal
	home, _ := os.UserHomeDir()
	return fmt.Sprintf("fallback-%s-%s", runtime.GOOS, home)
}

// ─────────────────────────────────────────────
// DERIVAÇÃO DE CHAVE
// ─────────────────────────────────────────────

func derivarChave(salt []byte, params kdfParams) []byte {
	seed := coletarSeed()
	defer zeroizar(seed)
	return argon2.IDKey(seed, salt, params.Time, params.Memory, params.Threads, params.KeyLen)
}

// ─────────────────────────────────────────────
// CRIPTOGRAFAR
// ─────────────────────────────────────────────

func criptografar(plaintext []byte) ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("erro ao gerar salt: %w", err)
	}

	key := derivarChave(salt, kdfAtual)
	defer zeroizar(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("erro ao gerar nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	env := envelope{
		Version:    envelopeVersion,
		KDF:        kdfAtual,
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
		CreatedAt:  time.Now().Unix(),
	}

	return json.Marshal(env)
}

// ─────────────────────────────────────────────
// DESCRIPTOGRAFAR
// ─────────────────────────────────────────────

func descriptografar(data []byte) ([]byte, error) {
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("formato de arquivo inválido: %w", err)
	}

	if env.Version != envelopeVersion {
		return nil, fmt.Errorf(
			"versão incompatível (arquivo: v%d, app: v%d) — recrie as credenciais",
			env.Version, envelopeVersion,
		)
	}

	// Usa os parâmetros gravados no envelope, não kdfAtual —
	// assim um arquivo antigo ainda abre mesmo após mudar kdfAtual.
	key := derivarChave(env.Salt, env.KDF)
	defer zeroizar(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, env.Nonce, env.Ciphertext, nil)
	if err != nil {
		// Não diferenciamos "ambiente diferente" de "corrompido" para não vazar info
		return nil, errors.New("falha na autenticação — ambiente diferente ou arquivo corrompido")
	}

	return plaintext, nil
}

// ─────────────────────────────────────────────
// LIMPEZA DE MEMÓRIA
// ─────────────────────────────────────────────

func zeroizar(b []byte) {
	for i := range b {
		b[i] = 0
	}
}