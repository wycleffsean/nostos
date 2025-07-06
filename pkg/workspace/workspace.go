package workspace

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/wycleffsean/nostos/pkg/urispec"
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

// SetSpec resolves the provided URI spec and sets the workspace directory
// accordingly. If spec is empty, it behaves like Set("").
func SetSpec(spec string) error {
	if spec == "" {
		Set("")
		return nil
	}
	u := urispec.Parse(spec)
	dir, err := u.LocalPath()
	if err != nil {
		return err
	}
	Set(dir)
	return nil
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
