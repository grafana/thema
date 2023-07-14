// Copyright 2020 CUE Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vanilla

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/pkg/encoding/json"
	"cuelang.org/go/pkg/encoding/yaml"
	"github.com/grafana/thema/internal/envvars"
	"github.com/grafana/thema/internal/util"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/txtar"
)

// A TxTarTest represents a test run that process all CUE tests in the txtar
// format rooted in a given directory. See the [Test] documentation for
// more details.
type TxTarTest struct {
	// Run TxTarTest on this directory.
	Root string

	ThemaFS fs.FS

	// Name is a unique name for this test. The golden file for this test is
	// derived from the out/<name> file in the .txtar file.
	//
	// TODO: by default derive from the current base directory name.
	Name string

	// Skip is a map of tests to skip; the key is the test name; the value is the
	// skip message.
	Skip map[string]string

	// ToDo is a map of tests that should be skipped now, but should be fixed.
	ToDo map[string]string
}

// A Test represents a single test based on a .txtar file.
//
// A Test embeds *[testing.T] and should be used to report errors.
//
// Entries within the txtar file define CUE files (available via the
// Instances and RawInstances methods) and expected output
// (or "golden") files (names starting with "out/\(testname)"). The "main" golden
// file is "out/\(testname)" itself, used when [Test] is used directly as an [io.Writer]
// and with [Test.WriteFile].
//
// When the test function has returned, output written with [Test.Write], [Test.Writer]
// and friends is checked against the expected output files.
//
// A txtar file can define test-specific tags and values in the comment section.
// These are available via the [Test.HasTag] and [Test.Value] methods.
// The #skip tag causes a [Test] to be skipped.
// The #noformat tag causes the $THEMA_FORMAT_TXTAR value
// to be ignored.
//
// If the output differs and $THEMA_UPDATE_GOLDEN is non-empty, the txtar file will be
// updated and written to disk with the actual output data replacing the
// out files.
//
// If $THEMA_FORMAT_TXTAR is non-empty, any CUE files in the txtar
// file will be updated to be properly formatted, unless the #noformat
// tag is present.
type Test struct {
	// Allow Test to be used as a T.
	*testing.T

	prefix   string
	buf      *bytes.Buffer // the default buffer
	outFiles []file

	Archive *txtar.Archive

	ThemaFS fs.FS

	// The absolute path of the current test directory.
	Dir string

	hasGold bool
}

// Write implements [io.Writer] by writing to the output for the test,
// which will be tested against the main golden file.
func (t *Test) Write(b []byte) (n int, err error) {
	if t.buf == nil {
		t.buf = &bytes.Buffer{}
		t.outFiles = append(t.outFiles, file{t.prefix, t.buf})
	}
	return t.buf.Write(b)
}

// HasTag reports whether the tag with the given key is defined
// for the current test. A tag x is defined by a line in the comment
// section of the txtar file like:
//
//	#x
func (t *Test) HasTag(key string) bool {
	prefix := []byte("#" + key)
	s := bufio.NewScanner(bytes.NewReader(t.Archive.Comment))
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
func (t *Test) Value(key string) (value string, ok bool) {
	prefix := []byte("#" + key + ":")
	s := bufio.NewScanner(bytes.NewReader(t.Archive.Comment))
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
func (t *Test) Bool(key string) bool {
	s, ok := t.Value(key)
	return ok && s == "true"
}

// Rel converts filename to a normalized form so that it will given the same
// output across different runs and OSes.
func (t *Test) Rel(filename string) string {
	rel, err := filepath.Rel(t.Dir, filename)
	if err != nil {
		return filepath.Base(filename)
	}
	return filepath.ToSlash(rel)
}

// WriteErrors writes the full list of errors in err to the test output.
func (t *Test) WriteErrors(err errors.Error) {
	if err != nil {
		errors.Print(t, err, &errors.Config{
			Cwd:     t.Dir,
			ToSlash: true,
		})
	}
}

// WriteFile formats f and writes it to the main output,
// prefixed by a line of the form:
//
//	== name
//
// where name is the base name of f.Filename.
func (t *Test) WriteFile(f *ast.File) {
	// TODO: use FileWriter instead in separate CL.
	fmt.Fprintln(t, "==", filepath.Base(f.Filename))
	_, _ = t.Write(formatVanillaNode(t.T, f))
}

// Writer returns a Writer with the given name. Data written will
// be checked against the file with name "out/\(testName)/\(name)"
// in the txtar file. If name is empty, data will be written to the test
// output and checked against "out/\(testName)".
func (t *Test) Writer(name string) io.Writer {
	switch name {
	case "":
		name = t.prefix
	default:
		name = path.Join(t.prefix, name)
	}

	for _, f := range t.outFiles {
		if f.name == name {
			return f.buf
		}
	}

	w := &bytes.Buffer{}
	t.outFiles = append(t.outFiles, file{name, w})

	if name == t.prefix {
		t.buf = w
	}

	return w
}

// Instance returns the single instance representing the
// root directory in the txtar file.
func (t *Test) Instance() *build.Instance {
	return t.Instances()[0]
}

// Instances returns the valid instances for this .txtar file or skips the
// test if there is an error loading the instances.
func (t *Test) Instances(args ...string) []*build.Instance {
	t.Helper()

	a := t.RawInstances(args...)
	for _, i := range a {
		if i.Err != nil {
			if t.hasGold {
				t.Fatal("Parse error: ", i.Err)
			}
			t.Skip("Parse error: ", i.Err)
		}
	}
	return a
}

func formatVanillaNode(t *testing.T, n ast.Node) []byte {
	t.Helper()

	var byt []byte
	var err error
	ctx := cuecontext.New()
	if f, is := n.(*ast.File); is {
		switch filepath.Ext(f.Filename) {
		case ".json":
			jbyt, err := ctx.BuildFile(f).MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			str, err := json.Indent(jbyt, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			byt = []byte(str)
		case ".yaml", ".yml":
			str, err := yaml.Marshal(ctx.BuildFile(f))
			if err != nil {
				t.Fatal(err)
			}
			byt = []byte(str)
		default:
			byt, err = format.Node(f)
		}
	} else {
		byt, err = format.Node(f)
	}

	if err != nil {
		t.Fatal(err)
	}
	return byt
}

// RawInstances returns the instances represented by this .txtar file. The
// returned instances are not checked for errors.
func (t *Test) RawInstances(args ...string) []*build.Instance {
	return LoadVanilla(t.ThemaFS, t.Archive, args...)
}

// LoadVanilla loads the instances of a txtar file. By default, it only loads
// files in the root directory.
func LoadVanilla(themaFS fs.FS, a *txtar.Archive, args ...string) []*build.Instance {
	vfsRootDir := "/"

	auto := len(args) == 0
	overlay := map[string]load.Source{}
	for _, f := range a.Files {
		if auto && !strings.Contains(f.Name, "/") {
			args = append(args, f.Name)
		}
		overlay[filepath.Join(vfsRootDir, f.Name)] = load.FromBytes(f.Data)
	}

	if err := util.ToOverlay(filepath.Join(vfsRootDir, "cue.mod/pkg/github.com/grafana/thema"), themaFS, overlay); err != nil {
		// util.ToOverlay() explores a virtual filesystem and any error is extremely unlikely.
		// Having a panic here is probably alright since this function is only used in tests.
		panic(err)
	}

	cfg := &load.Config{
		Dir:     vfsRootDir,
		Overlay: overlay,
	}

	return load.Instances(args, cfg)
}

// Run runs tests defined in txtar files in x.Root or its subdirectories.
//
// The function f is called for each such txtar file. See the [Test] documentation
// for more details.
func (x *TxTarTest) Run(t *testing.T, f func(tc *Test)) {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	err = filepath.WalkDir(x.Root, func(fullpath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(fullpath) != ".txtar" {
			return nil
		}

		str := filepath.ToSlash(fullpath)
		p := strings.Index(str, "/testdata/")
		testName := str[p+len("/testdata/") : len(str)-len(".txtar")]

		t.Run(testName, func(t *testing.T) {
			a, err := txtar.ParseFile(fullpath)
			if err != nil {
				t.Fatalf("error parsing txtar file: %v", err)
			}

			tc := &Test{
				T:       t,
				Archive: a,
				ThemaFS: x.ThemaFS,
				Dir:     filepath.Dir(filepath.Join(dir, fullpath)),
				prefix:  path.Join("out", x.Name),
			}

			if tc.HasTag("skip") {
				t.Skip()
			}
			if testing.Short() && tc.HasTag("slow") {
				tc.Skip("case is tagged #slow, skipping for -short")
			}

			if msg, ok := x.Skip[testName]; ok {
				t.Skip(msg)
			}
			if msg, ok := x.ToDo[testName]; ok {
				t.Skip(msg)
			}

			update := false
			for i, f := range a.Files {
				if strings.HasPrefix(f.Name, tc.prefix) && (f.Name == tc.prefix || f.Name[len(tc.prefix)] == '/') {
					// It's either "\(tc.prefix)" or "\(tc.prefix)/..." but not some other name
					// that happens to start with tc.prefix.
					tc.hasGold = true
				}

				// Format CUE files as required
				if tc.HasTag("noformat") || !strings.HasSuffix(f.Name, ".cue") {
					continue
				}
				if ff, err := format.Source(f.Data); err == nil {
					if bytes.Equal(f.Data, ff) {
						continue
					}
					if envvars.FormatTxtar {
						update = true
						a.Files[i].Data = ff
					}
				}
			}

			f(tc)

			// TODO we MAY need the below if trying to enable parallel tests
			//
			// Lock and re-parse the txtar file now that test execution is done. This does
			// make for some weird edge cases, but as long as underlying fs supports file
			// locking (windows? :scream:) it should make it safe to run multiple tests on same
			// txtar archive in parallel.
			// lock := flock.New(fullpath)
			// defer lock.Unlock()
			// a, err = txtar.ParseFile(fullpath)
			// if err != nil {
			// 	t.Fatalf("error parsing txtar file: %v", err)
			// }

			index := make(map[string]int, len(a.Files))
			for i, f := range a.Files {
				index[f.Name] = i
			}

			// Insert results of this test at first location of any existing
			// test or at end of list otherwise.
			k := len(a.Files)
			for _, sub := range tc.outFiles {
				if i, ok := index[sub.name]; ok {
					k = i
					break
				}
			}

			files := a.Files[:k:k]

			for _, sub := range tc.outFiles {
				result := sub.buf.Bytes()

				files = append(files, txtar.File{Name: sub.name})
				gold := &files[len(files)-1]

				if i, ok := index[sub.name]; ok {
					gold.Data = a.Files[i].Data
					delete(index, sub.name)

					if bytes.Equal(gold.Data, result) || bytes.Equal(bytes.TrimRight(gold.Data, "\n"), result) {
						continue
					}
				}

				if envvars.UpdateGoldenFiles {
					update = true
					gold.Data = result
					continue
				}

				require.Equal(t, string(gold.Data), string(result), "result for %s differs", sub.name)
			}

			// Add remaining unrelated files, ignoring files that were already
			// added.
			for _, f := range a.Files[k:] {
				if _, ok := index[f.Name]; ok {
					files = append(files, f)
				}
			}
			a.Files = files

			if update {
				err = os.WriteFile(fullpath, txtar.Format(a), 0644)
				if err != nil {
					t.Fatal(err)
				}
			}
		})

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}

func dumpTestInfo(tc *Test) {
	fmt.Println("=== TEST:", tc.Dir, tc.prefix)
	fmt.Println("=== Files")
	for _, f := range tc.Archive.Files {
		fmt.Printf("===   %s\n", f.Name)
	}
	fmt.Println("=== END TEST")
}

type file struct {
	name string
	buf  *bytes.Buffer
}
