package appsvc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"SimklExpoGter/internal/config"

	"golang.org/x/crypto/scrypt"
)

const (
	SettingsBackupExtension = ".simklsettings"
	settingsBackupVersion   = 1
	settingsBackupSaltLen   = 16
	settingsBackupKeyLen    = 32
	settingsBackupScryptN   = 32768
	settingsBackupScryptR   = 8
	settingsBackupScryptP   = 1
)

type encryptedSettingsBackup struct {
	Version   int                        `json:"version"`
	AppName   string                     `json:"appName"`
	CreatedAt string                     `json:"createdAt"`
	KDF       encryptedSettingsBackupKDF `json:"kdf"`
	Cipher    encryptedSettingsCipher    `json:"cipher"`
}

type encryptedSettingsBackupKDF struct {
	Name   string `json:"name"`
	Salt   string `json:"salt"`
	N      int    `json:"n"`
	R      int    `json:"r"`
	P      int    `json:"p"`
	KeyLen int    `json:"keyLen"`
}

type encryptedSettingsCipher struct {
	Name       string `json:"name"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func (s *Service) ExportEncryptedSettings(path, password string) (string, error) {
	path = normalizeSettingsBackupPath(path)
	if path == "" {
		return "", newUsageError("choose a destination for the encrypted settings backup")
	}

	password = strings.TrimSpace(password)
	if password == "" {
		return "", newUsageError("enter a password before exporting the settings backup")
	}

	settings, err := s.store.Load()
	if err != nil {
		return "", err
	}

	payload, err := encryptSettingsBackup(settings, password)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return "", err
	}

	return path, nil
}

func (s *Service) ImportEncryptedSettings(path, password string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return newUsageError("choose an encrypted settings backup to import")
	}

	password = strings.TrimSpace(password)
	if password == "" {
		return newUsageError("enter the backup password before importing settings")
	}

	payload, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	settings, err := decryptSettingsBackup(payload, password)
	if err != nil {
		return err
	}

	return s.store.Save(settings)
}

func normalizeSettingsBackupPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	if strings.EqualFold(filepath.Ext(path), SettingsBackupExtension) {
		return path
	}

	return path + SettingsBackupExtension
}

func DefaultSettingsBackupFilename(now time.Time) string {
	return "simklexpogter-settings-" + now.UTC().Format("20060102-150405") + SettingsBackupExtension
}

func encryptSettingsBackup(settings config.Settings, password string) ([]byte, error) {
	plaintext, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	salt, err := randomBytes(settingsBackupSaltLen)
	if err != nil {
		return nil, err
	}

	key, err := scrypt.Key(
		[]byte(password),
		salt,
		settingsBackupScryptN,
		settingsBackupScryptR,
		settingsBackupScryptP,
		settingsBackupKeyLen,
	)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, err := randomBytes(gcm.NonceSize())
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, settingsBackupAAD())

	backup := encryptedSettingsBackup{
		Version:   settingsBackupVersion,
		AppName:   DefaultAppName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		KDF: encryptedSettingsBackupKDF{
			Name:   "scrypt",
			Salt:   encodeBase64(salt),
			N:      settingsBackupScryptN,
			R:      settingsBackupScryptR,
			P:      settingsBackupScryptP,
			KeyLen: settingsBackupKeyLen,
		},
		Cipher: encryptedSettingsCipher{
			Name:       "aes-256-gcm",
			Nonce:      encodeBase64(nonce),
			Ciphertext: encodeBase64(ciphertext),
		},
	}

	return json.MarshalIndent(backup, "", "  ")
}

func decryptSettingsBackup(payload []byte, password string) (config.Settings, error) {
	var backup encryptedSettingsBackup
	if err := json.Unmarshal(payload, &backup); err != nil {
		return config.Settings{}, newUsageError("the selected file is not a valid encrypted settings backup")
	}

	if backup.Version != settingsBackupVersion {
		return config.Settings{}, newUsageError(fmt.Sprintf("unsupported settings backup version %d", backup.Version))
	}

	if backup.KDF.Name != "scrypt" || backup.Cipher.Name != "aes-256-gcm" {
		return config.Settings{}, newUsageError("the selected settings backup uses an unsupported encryption format")
	}

	salt, err := decodeBase64(backup.KDF.Salt)
	if err != nil {
		return config.Settings{}, newUsageError("the selected settings backup is missing its encryption salt")
	}

	nonce, err := decodeBase64(backup.Cipher.Nonce)
	if err != nil {
		return config.Settings{}, newUsageError("the selected settings backup is missing its encryption nonce")
	}

	ciphertext, err := decodeBase64(backup.Cipher.Ciphertext)
	if err != nil {
		return config.Settings{}, newUsageError("the selected settings backup is missing its encrypted payload")
	}

	key, err := scrypt.Key(
		[]byte(password),
		salt,
		backup.KDF.N,
		backup.KDF.R,
		backup.KDF.P,
		backup.KDF.KeyLen,
	)
	if err != nil {
		return config.Settings{}, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return config.Settings{}, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return config.Settings{}, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, settingsBackupAAD())
	if err != nil {
		return config.Settings{}, newUsageError("failed to decrypt the settings backup. Check the password and try again")
	}

	var settings config.Settings
	if err := json.Unmarshal(plaintext, &settings); err != nil {
		return config.Settings{}, errors.New("the decrypted settings backup is corrupted")
	}

	if settings.LastActivities == nil {
		settings.LastActivities = map[string]any{}
	}

	return settings, nil
}

func settingsBackupAAD() []byte {
	return []byte(DefaultAppName + ":settings-backup:v1")
}

func randomBytes(length int) ([]byte, error) {
	buffer := make([]byte, length)
	if _, err := rand.Read(buffer); err != nil {
		return nil, err
	}

	return buffer, nil
}

func encodeBase64(value []byte) string {
	return base64.RawStdEncoding.EncodeToString(value)
}

func decodeBase64(value string) ([]byte, error) {
	return base64.RawStdEncoding.DecodeString(strings.TrimSpace(value))
}
