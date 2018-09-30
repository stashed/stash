package local

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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
	path := filepath.Join(c.path, filepath.FromSlash(name))
	item := &item{
		path:          path,
		contPrefixLen: len(c.path) + 1,
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return item, f, nil
}

func (c *container) RemoveItem(id string) error {
	return os.Remove(id)
}

func (c *container) Put(name string, r io.Reader, size int64, metadata map[string]interface{}) (stow.Item, error) {
	if len(metadata) > 0 {
		return nil, stow.NotSupported("metadata")
	}

	path := filepath.Join(c.path, filepath.FromSlash(name))
	item := &item{
		path:          path,
		contPrefixLen: len(c.path) + 1,
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
	conPrefixLen := len(c.path) + 1

	var files []os.FileInfo
	var err error
	r, sz := utf8.DecodeRuneInString(delimiter)
	if r == utf8.RuneError {
		if sz == 0 {
			prefixDir, _ := filepath.Split(filepath.FromSlash(prefix))
			dir := filepath.Join(c.path, prefixDir)
			files, err = flatdirs(dir)
		} else {
			return nil, fmt.Errorf("bad delimiter %v", delimiter)
		}
	} else if sz == len(delimiter) && r == '/' {
		dir := filepath.Join(c.path, filepath.FromSlash(prefix))
		files, err = readDir(dir)
	} else {
		return nil, fmt.Errorf("unknown delimiter %v", delimiter)
	}
	if err != nil {
		return nil, err
	}
	if cursor != stow.CursorStart {
		// seek to the cursor
		ok := false
		c := filepath.Join(c.path, cursor)
		for i, file := range files {
			if file.Name() == c {
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
		cursor = files[count].Name()[conPrefixLen:] // next item path as cursor
		files = files[:count]
	} else if len(files) <= count {
		cursor = "" // end
	}

	var prefixes []string
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		path := filepath.ToSlash(f.Name()[conPrefixLen:])
		if strings.HasPrefix(path, prefix) {
			prefixes = append(prefixes, path)
		}
	}

	var items []stow.Item
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		path := filepath.ToSlash(f.Name()[conPrefixLen:])
		if strings.HasPrefix(path, prefix) {
			item := &item{
				path:          f.Name(),
				contPrefixLen: conPrefixLen,
			}
			items = append(items, item)
		}
	}
	return &stow.ItemPage{Prefixes: prefixes, Items: items, Cursor: cursor}, nil
}

func (c *container) Items(prefix, cursor string, count int) ([]stow.Item, string, error) {
	page, err := c.Browse(prefix, "", cursor, count)
	if err != nil {
		return nil, "", err
	}
	return page.Items, page.Cursor, err
}

func (c *container) Item(id string) (stow.Item, error) {
	path := id
	if !filepath.IsAbs(id) {
		path = filepath.Join(c.path, filepath.FromSlash(id))
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, stow.ErrNotFound
	}
	if info.IsDir() {
		return nil, errors.New("unexpected directory")
	}
	_, err = filepath.Rel(c.path, path)
	if err != nil {
		return nil, err
	}
	item := &item{
		path:          path,
		contPrefixLen: len(c.path) + 1,
	}
	return item, nil
}

// readDir reads the directory named by dirname and returns
// a list of directory entries sorted by filename.
func readDir(dirname string) ([]os.FileInfo, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil, err
	}

	for i := range list {
		info := list[i]
		list[i] = fileinfo{
			FileInfo: info,
			path:     filepath.Join(dirname, info.Name()),
		}
	}
	return list, nil
}

// flatdirs walks the entire tree returning a list of
// os.FileInfo for all items encountered.
func flatdirs(path string) ([]os.FileInfo, error) {
	var list []os.FileInfo
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			list = append(list, fileinfo{
				FileInfo: info,
				path:     p,
			})
		}
		return nil
	})
	return list, err
}

type fileinfo struct {
	os.FileInfo
	path string
}

func (f fileinfo) Name() string {
	return f.path
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
