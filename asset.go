package flotilla

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"
)

type (
	FakeFile struct {
		Path string
		Dir  bool
		Len  int64
	}

	AssetFile struct {
		*bytes.Reader
		io.Closer
		FakeFile
	}

	AssetDirectory struct {
		AssetFile
		ChildrenRead int
		Children     []os.FileInfo
	}

	// A pseudo-file structure constructed from functions & optional prefix
	// Flotilla can use binary data, and is the currently optimal way to define
	// inbuilt assets for extensions.
	// See:
	// https://github.com/jteeuwen/go-bindata
	// https://github.com/elazarl/go-bindata-assetfs
	AssetFS struct {
		Asset      func(path string) ([]byte, error)
		AssetDir   func(path string) ([]string, error)
		AssetNames func() []string
		Prefix     string
	}

	// An array of AssetFS instances
	Assets []*AssetFS
)

func (f *FakeFile) Name() string {
	_, name := filepath.Split(f.Path)
	return name
}

func (f *FakeFile) Mode() os.FileMode {
	mode := os.FileMode(0644)
	if f.Dir {
		return mode | os.ModeDir
	}
	return mode
}

func (f *FakeFile) ModTime() time.Time {
	return time.Unix(0, 0)
}

func (f *FakeFile) Size() int64 {
	return f.Len
}

func (f *FakeFile) IsDir() bool {
	return f.Mode().IsDir()
}

func (f *FakeFile) Sys() interface{} {
	return nil
}

func NewAssetFile(name string, content []byte) *AssetFile {
	return &AssetFile{
		bytes.NewReader(content),
		ioutil.NopCloser(nil),
		FakeFile{name, false, int64(len(content))},
	}
}

func (f *AssetFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.New("not a directory")
}

func (f *AssetFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func NewAssetDirectory(name string, children []string, fs *AssetFS) *AssetDirectory {
	fileinfos := make([]os.FileInfo, 0, len(children))
	for _, child := range children {
		_, err := fs.AssetDir(filepath.Join(name, child))
		fileinfos = append(fileinfos, &FakeFile{child, err == nil, 0})
	}
	return &AssetDirectory{
		AssetFile{
			bytes.NewReader(nil),
			ioutil.NopCloser(nil),
			FakeFile{name, true, 0},
		},
		0,
		fileinfos}
}

func (f *AssetDirectory) Readdir(count int) ([]os.FileInfo, error) {
	fmt.Println(f, count)
	if count <= 0 {
		return f.Children, nil
	}
	if f.ChildrenRead+count > len(f.Children) {
		count = len(f.Children) - f.ChildrenRead
	}
	rv := f.Children[f.ChildrenRead : f.ChildrenRead+count]
	f.ChildrenRead += count
	return rv, nil
}

func (f *AssetDirectory) Stat() (os.FileInfo, error) {
	return f, nil
}

func (fs *AssetFS) HasAsset(requested string) (string, bool) {
	for _, filename := range fs.AssetNames() {
		if path.Base(filename) == requested {
			return filename, true
		}
	}
	return "", false
}

func (fs *AssetFS) GetAsset(requested string) (http.File, error) {
	if hasasset, ok := fs.HasAsset(requested); ok {
		f, err := fs.Open(hasasset)
		return f, err
	}
	return nil, newError("asset %s unvailable", requested)
}

func (fs *AssetFS) Open(name string) (http.File, error) {
	name = path.Join(fs.Prefix, name)
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	if children, err := fs.AssetDir(name); err == nil {
		return NewAssetDirectory(name, children, fs), nil
	}
	b, err := fs.Asset(name)
	if err != nil {
		return nil, err
	}
	return NewAssetFile(name, b), nil
}

// Return the requested asset as http.File from the AssetFS's contained
// in Asset, by supplying a string
func (a Assets) Get(requested string) (http.File, error) {
	for _, x := range a {
		f, err := x.GetAsset(requested)
		if err == nil {
			return f, nil
		}
	}
	return nil, newError("asset %s unavailable", requested)
}

func (a Assets) GetByte(requested string) ([]byte, error) {
	for _, x := range a {
		b, err := x.Asset(requested)
		if err == nil {
			return b, nil
		}
	}
	return nil, newError("asset %s unavailable", requested)
}
