package types

import (
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

func GetAbsolutePath(path string) (newPath string, err error) {
	if strings.HasPrefix(path, "~") {
		newPath, err = homedir.Expand(path)
		if err != nil {
			return "", errors.Wrapf(err, "failed to expand path: %q", path)
		}
	} else {
		newPath, err = filepath.Abs(path)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get absolute path of %q", path)
		}
	}

	l := len(newPath)
	if l > 1 && newPath[l-1] == '/' { // if l == 1, the path might be "/"
		newPath = newPath[:l-1]
	}

	return newPath, nil
}
