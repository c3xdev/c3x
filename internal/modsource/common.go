// Copyright (c) C3X Dev


package modsource

import (
	"os"
)

func tmpFile(dir, pattern string) (string, error) {
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	_ = f.Close()
	return f.Name(), nil
}
