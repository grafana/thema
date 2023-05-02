package storage

import (
	"os"
	"path/filepath"

	"github.com/grafana/thema"
	"github.com/grafana/thema/server/utils"
)

var _ Store = &FileStore{}

type FileStore struct {
	root string
}

func NewFileStore(root string) *FileStore {
	return &FileStore{
		root: root,
	}
}

func (s *FileStore) StoreLineage(registry string, name string, lineage []byte) error {
	err := os.WriteFile(filepath.Join(s.root, registry, name+".cue"), lineage, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (s *FileStore) GetLineage(registry string, name string) (thema.Lineage, error) {
	bytes, err := os.ReadFile(filepath.Join(s.root, registry, name+".cue"))
	if err != nil {
		return nil, err
	}

	return utils.GetLineageFromBytes(bytes)
}

func (s *FileStore) GetSchema(registry string, lineage string, version thema.SyntacticVersion) (thema.Schema, error) {
	return nil, nil
}
