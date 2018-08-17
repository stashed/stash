package local

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/graymeta/stow"
)

type container struct {
	name string
	path string
}

func (c *container) ID() string {
	return c.path
}

func (c *container) Name() string {
	return c.name
}

func (c *container) URL() *url.URL {
	return &url.URL{
		Scheme: "file",
		Path:   filepath.Clean(c.path),
	}
}

func (c *container) CreateItem(name string) (stow.Item, io.WriteCloser, error) {
	path := filepath.Join(c.path, name)
	item := &item{
		path: path,
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return item, f, nil
}

func (c *container) RemoveItem(id string) error {
	var path string
	if filepath.IsAbs(id) {
		path = id
	} else {
		path = filepath.Join(c.path, id)
	}
	return os.Remove(path)
}

func (c *container) Put(name string, r io.Reader, size int64, metadata map[string]interface{}) (stow.Item, error) {
	if len(metadata) > 0 {
		return nil, stow.NotSupported("metadata")
	}

	path := filepath.Join(c.path, name)
	item := &item{
		path: path,
	}
	err := os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return nil, err
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	if err != nil {
		return nil, err
	}
	if n != size {
		return nil, errors.New("bad size")
	}
	return item, nil
}

func (c *container) Browse(prefix, delimiter, cursor string, count int) (*stow.ItemPage, error) {
	var files []os.FileInfo
	var err error
	r, sz := utf8.DecodeRuneInString(delimiter)
	if r == utf8.RuneError {
		if sz == 0 {
			files, err = flatdirs(c.path)
		} else {
			return nil, fmt.Errorf("Bad delimiter %v", delimiter)
		}
	} else if sz == len(delimiter) && r == os.PathSeparator {
		var dir string
		if filepath.IsAbs(prefix) {
			dir = prefix
		} else {
			dir = filepath.Join(c.path, prefix)
		}
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return &stow.ItemPage{}, nil
		}
		var fis []os.FileInfo
		fis, err = ioutil.ReadDir(dir)
		for _, fi := range fis {
			n, err := filepath.Rel(c.path, filepath.Join(dir, fi.Name()))
			if err != nil {
				return nil, err
			}
			files = append(files, fileinfo{
				FileInfo: fi,
				name:     n,
			})
		}
	} else {
		return nil, errors.New("Unknown delimeter " + delimiter)
	}
	if err != nil {
		return nil, err
	}
	if cursor != stow.CursorStart {
		// seek to the cursor
		ok := false
		for i, file := range files {
			if file.Name() == cursor {
				files = files[i:]
				ok = true
				break
			}
		}
		if !ok {
			return nil, stow.ErrBadCursor
		}
	}
	if len(files) > count {
		cursor = files[count].Name()
		files = files[:count]
	} else if len(files) <= count {
		cursor = "" // end
	}

	var prefixes []string
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		path, err := filepath.Abs(filepath.Join(c.path, f.Name()))
		if err != nil {
			return nil, err
		}
		prefixes = append(prefixes, path)
	}

	var items []stow.Item
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		path, err := filepath.Abs(filepath.Join(c.path, f.Name()))
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(f.Name(), prefix) {
			continue
		}
		item := &item{
			path: path,
		}
		items = append(items, item)
	}
	return &stow.ItemPage{Prefixes: prefixes, Items: items, Cursor: cursor}, nil
}

func (c *container) Items(prefix, cursor string, count int) ([]stow.Item, string, error) {
	page, err := c.Browse(prefix, "", cursor, count)
	if err != nil {
		return nil, "", err
	}
	return page.Items, cursor, err
}

func (c *container) Item(id string) (stow.Item, error) {
	var path string
	if filepath.IsAbs(id) {
		path = id
	} else {
		path = filepath.Join(c.path, id)
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, stow.ErrNotFound
	}
	if info.IsDir() {
		return nil, errors.New("unexpected directory")
	}

	item := &item{
		path: path,
	}
	return item, nil
}

// flatdirs walks the entire tree returning a list of
// os.FileInfo for all items encountered.
func flatdirs(path string) ([]os.FileInfo, error) {
	var list []os.FileInfo
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		flatname, err := filepath.Rel(path, p)
		if err != nil {
			return err
		}
		list = append(list, fileinfo{
			FileInfo: info,
			name:     flatname,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

type fileinfo struct {
	os.FileInfo
	name string
}

func (f fileinfo) Name() string {
	return f.name
}

func (c *container) HasWriteAccess() error {
	r := bytes.NewReader([]byte("CheckBucketAccess"))
	item, err := c.Put(".objectstore", r, r.Size(), nil)
	if err != nil {
		return err
	}
	if err := c.RemoveItem(item.ID()); err != nil {
		return err
	}
	return nil
}
