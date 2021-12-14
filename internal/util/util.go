package util

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"cuelang.org/go/cue/load"
)

func ToOverlay(prefix string, vfs fs.FS, overlay map[string]load.Source) error {
	if !filepath.IsAbs(prefix) {
		return fmt.Errorf("must provide absolute path prefix when generating cue overlay, got %q", prefix)
	}
	err := fs.WalkDir(vfs, ".", (func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		f, err := vfs.Open(path)
		if err != nil {
			return err
		}

		b, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		overlay[filepath.Join(prefix, path)] = load.FromBytes(b)
		return nil
	}))

	if err != nil {
		return err
	}

	return nil
}
