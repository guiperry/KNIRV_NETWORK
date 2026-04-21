package keyfile

import (
	"backend_server/internal/proto"
	"backend_server/internal/utils"
	"os"
	"path/filepath"
)

type RootKeyFileContentProto = proto.RootKeyFileContentProto
type EncryptedRootKeyFile = proto.EncryptedRootKeyFile

func GetRootKeyPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".knirv", "root.key"), nil
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