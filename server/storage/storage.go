package storage

import "github.com/grafana/thema"

type Store interface {
	StoreLineage(registry string, name string, lineage []byte) error
	GetLineage(registry string, name string) (thema.Lineage, error)
	GetSchema(registry string, lineage string, version thema.SyntacticVersion) (thema.Schema, error)
}
