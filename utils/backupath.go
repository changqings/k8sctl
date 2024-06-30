package utils

import (
	"log/slog"
	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)

func GetBackupPath() (*string, error) {

	backupPath := filepath.Join(homedir.HomeDir(), ".kube", "k8sctl-backups")

	_, err := os.Lstat(backupPath)
	if err != nil {
		slog.Error("not found", "path", backupPath, "msg", "create new one")
		if err := os.MkdirAll(backupPath, 0755); err != nil {
			return nil, err
		}
	}

	return &backupPath, nil

}
