package gocode

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostprocessGoFile_withDiscoveredImport(t *testing.T) {
	req := require.New(t)
	input := GenGoFile{
		Path: "file.go",
		In: []byte(`package thema

func Hello(name string) string {
	return fmt.Sprintf("Hello %w", name)
}`),
		IgnoreDiscoveredImports: false,
	}

	_, err := PostprocessGoFile(input)
	req.Error(err)
	req.ErrorContains(err, "goimports added the following import statements")
}

func TestPostprocessGoFile_withIgnoredDiscoveredImport(t *testing.T) {
	req := require.New(t)
	input := GenGoFile{
		Path: "file.go",
		In: []byte(`package thema

func Hello(name string) string {
	return fmt.Sprintf("Hello %w", name)
}`),
		IgnoreDiscoveredImports: true,
	}

	output, err := PostprocessGoFile(input)
	req.NoError(err)

	req.Equal(`package thema

import "fmt"

func Hello(name string) string {
	return fmt.Sprintf("Hello %w", name)
}
`, string(output))
}
