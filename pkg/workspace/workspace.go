package workspace

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const key = "workspace_dir"

// Set assigns the workspace directory. If dir is empty, it defaults to the
// current working directory. The value is stored in viper for global access.
func Set(dir string) {
	if dir == "" {
		if cwd, err := os.Getwd(); err == nil {
			dir = cwd
		}
	}
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	viper.Set(key, dir)
}

// Dir returns the currently configured workspace directory. If none was set,
// it falls back to the current working directory.
func Dir() string {
	if d := viper.GetString(key); d != "" {
		return d
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}
