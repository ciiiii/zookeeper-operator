package util

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func IsDirectory(path string) (bool, error) {
	fileOrDir, err := os.Open(path)
	if err != nil {
		return false, errors.WithStack(err)
	}
	defer func() { _ = fileOrDir.Close() }()
	stat, err := fileOrDir.Stat()
	if err != nil {
		return false, errors.WithStack(err)
	}
	return stat.IsDir(), nil
}

func ForceWriteFile(path string, content []byte) error {
	_, err := os.Stat(path)
	switch {
	case err != nil && os.IsNotExist(err):
		parent := filepath.Dir(path)
		if _, err := os.Stat(parent); os.IsNotExist(err) {
			if err := os.MkdirAll(parent, 0755); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(content)
		if err != nil {
			return err
		}
		return nil
	case err != nil:
		return err
	default:
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(content)
		if err != nil {
			return err
		}
		return nil
	}
}
