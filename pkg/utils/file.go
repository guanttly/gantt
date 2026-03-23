package utils

import (
	"fmt"
	"io"
	"os"
)

// CopyFile 将文件从 src 复制到 dst。
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy from %s to %s: %w", src, dst, err)
	}
	return destFile.Sync()
}
