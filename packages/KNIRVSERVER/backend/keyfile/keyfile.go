package keyfile

import (
	"backend_server/internal/proto"
	"backend_server/internal/utils"
	"fmt"
	"os"
	"path/filepath"
)

type RootKeyFileContentProto = proto.RootKeyFileContentProto
type EncryptedRootKeyFile = proto.EncryptedRootKeyFile

func GetRootKeyPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "knirv-server", "root.key"), nil
}

func DeriveKeyFromPassword(password, salt []byte, n, r, p, keyLen int) ([]byte, error) {
	return utils.DeriveKeyFromPassword(password, salt, n, r, p, keyLen)
}

func GenerateSalt(length int) ([]byte, error) {
	return utils.GenerateSalt(length)
}

func Encrypt(data, key []byte) ([]byte, error) {
	return utils.Encrypt(data, key)
}
