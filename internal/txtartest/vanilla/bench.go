package vanilla

import (
	"bufio"
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue/build"
	"golang.org/x/tools/txtar"
)

// Benchmark is largely like [Test] but for standard Go benchmarks.
//
// Unlike in [Test], there is no support for writing output to the txtar file.
type Benchmark struct {
	*testing.B

	Archive *txtar.Archive

	// The absolute path of the current test directory.
	Dir string

	prefix   string
	buf      *bytes.Buffer // the default buffer
	outFiles []file
}

// HasTag reports whether the tag with the given key is defined
// for the current test. A tag x is defined by a line in the comment
// section of the txtar file like:
//
//	#x
func (b *Benchmark) HasTag(key string) bool {
	prefix := []byte("#" + key)
	s := bufio.NewScanner(bytes.NewReader(b.Archive.Comment))
	for s.Scan() {
		b := s.Bytes()
		if bytes.Equal(bytes.TrimSpace(b), prefix) {
			return true
		}
	}
	return false
}

// Value returns the value for the given key for this test and
// reports whether it was defined.
//
// A value is defined by a line in the comment section of the txtar
// file like:
//
//	#key: value
//
// White space is trimmed from the value before returning.
func (b *Benchmark) Value(key string) (value string, ok bool) {
	prefix := []byte("#" + key + ":")
	s := bufio.NewScanner(bytes.NewReader(b.Archive.Comment))
	for s.Scan() {
		b := s.Bytes()
		if bytes.HasPrefix(b, prefix) {
			return string(bytes.TrimSpace(b[len(prefix):])), true
		}
	}
	return "", false
}

// Bool searches for a line starting with #key: value in the comment and
// reports whether the key exists and its value is true.
func (b *Benchmark) Bool(key string) bool {
	s, ok := b.Value(key)
	return ok && s == "true"
}

// Instance returns the single instance representing the
// root directory in the txtar file.
func (b *Benchmark) Instance() *build.Instance {
	return b.Instances()[0]
}

// Instances returns the valid instances for this .txtar file or skips the
// test if there is an error loading the instances.
func (b *Benchmark) Instances(args ...string) []*build.Instance {
	b.Helper()

	a := b.RawInstances(args...)
	for _, i := range a {
		if i.Err != nil {
			b.Fatal("Parse error: ", i.Err)
		}
	}
	return a
}

// RawInstances returns the instances represented by this .txtar file. The
// returned instances are not checked for errors.
func (b *Benchmark) RawInstances(args ...string) []*build.Instance {
	return LoadVanilla(b.Archive, b.Dir, args...)
}

// RunBenchmark runs a benchmark on inputs defined in txtar files in x.Root or its subdirectories.
//
// The function f is called for each such txtar file.
func (x *TxTarTest) RunBenchmark(b *testing.B, f func(bc *Benchmark)) {
	b.Helper()

	dir, err := os.Getwd()
	if err != nil {
		b.Fatal(err)
	}

	root := x.Root

	err = filepath.WalkDir(root, func(fullpath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(fullpath) != ".txtar" {
			return nil
		}

		str := filepath.ToSlash(fullpath)
		p := strings.Index(str, "/testdata/")
		testName := str[p+len("/testdata/") : len(str)-len(".txtar")]

		b.Run(testName, func(b *testing.B) {
			a, err := txtar.ParseFile(fullpath)
			if err != nil {
				b.Fatalf("error parsing txtar file: %v", err)
			}

			bc := &Benchmark{
				B:       b,
				Archive: a,
				Dir:     filepath.Dir(filepath.Join(dir, fullpath)),
				prefix:  path.Join("out", x.Name),
			}

			if bc.HasTag("skip") {
				b.Skip()
			}
			if testing.Short() && bc.HasTag("slow") {
				bc.Skip("case is tagged #slow, skipping for -short")
			}

			if msg, ok := x.Skip[testName]; ok {
				b.Skip(msg)
			}
			if msg, ok := x.ToDo[testName]; ok {
				b.Skip(msg)
			}
			b.ResetTimer()

			f(bc)
		})

		return nil
	})

	if err != nil {
		b.Fatal(err)
	}
}
