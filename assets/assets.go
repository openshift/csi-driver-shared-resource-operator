package assets

import (
	"embed"
)

//go:embed *.yaml rbac/*.yaml webhook/*.yaml
var f embed.FS

// ReadFile reads and returns the content of the named file.
func ReadFile(name string) ([]byte, error) {
	return f.ReadFile(name)
}

// MustAsset reads and returns the content of the file at path or panics
// if something went wrong.
func MustAsset(path string) []byte {
	data, err := f.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}
