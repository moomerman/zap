// Code generated by go-bindata.
// sources:
// templates/502.html
// templates/app.html
// templates/layout.html
// templates/log.html
// templates/ngrok.html
// DO NOT EDIT!

package zap

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi bindataFileInfo) Name() string {
	return fi.name
}
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}
func (fi bindataFileInfo) IsDir() bool {
	return false
}
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _templates502Html = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xb2\xc9\x30\xb4\x7b\x34\x6b\x61\x54\x62\x81\x82\xae\x82\xa9\x81\x91\x82\x53\x62\x8a\x82\x7b\x62\x49\x6a\x79\x62\xa5\x8d\x7e\x86\xa1\x1d\x17\x97\x4d\x81\x5d\x75\xb5\x5e\x6d\xad\x8d\x7e\x81\x1d\x17\x20\x00\x00\xff\xff\x33\x67\x2b\x86\x30\x00\x00\x00")

func templates502HtmlBytes() ([]byte, error) {
	return bindataRead(
		_templates502Html,
		"templates/502.html",
	)
}

func templates502Html() (*asset, error) {
	bytes, err := templates502HtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/502.html", size: 48, mode: os.FileMode(420), modTime: time.Unix(1507924147, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesAppHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x94\x94\xc1\x6e\xd3\x4c\x10\xc7\xef\x7e\x8a\xf9\x7c\xa8\x9c\x28\x9f\x9d\x56\x70\x21\xb1\x05\x45\x15\x01\xa5\x20\xd1\x1c\x10\xb7\xad\x3d\xd9\x2c\xb8\xbb\xdb\xdd\x71\xdb\x10\xf9\x45\xb8\xf1\x6a\x3c\x09\xda\xb5\xe3\xd8\x09\x42\x70\xa8\x54\xef\xec\x7f\x66\xfe\xbf\x99\xcd\x7c\x73\x9e\xfd\xfc\xfe\xe3\x33\xd3\xf0\x3f\xec\x76\x10\xbf\x56\x72\x2d\x78\xbc\x50\x96\xa0\xae\xe7\xc9\xe6\x3c\x0b\x82\xb9\xce\x6e\x88\x51\x65\x5f\xc0\xdc\x6a\x26\x41\x14\x69\x68\xfd\x49\x98\x39\x55\x13\xf5\x02\x17\xcf\xe6\x89\xf6\x32\x83\xfe\x6a\xa9\xb8\xbf\x27\xd6\x80\xf7\xdd\xed\xd0\x54\x52\x0a\xc9\x43\xa8\x6b\x97\xe4\x55\xc1\x34\xa1\x89\x2f\x95\xa2\xa5\xe2\xcd\x29\x96\x16\xdb\xf8\x52\xf1\x15\x13\x65\x7b\x2e\x0b\x5f\x4e\x1b\x74\x95\x6c\x6e\x84\x26\xa0\xad\xc6\x34\x24\x7c\xa2\xe4\x0b\x7b\x60\xcd\x69\x98\x05\x00\xc9\x78\x0c\x2f\x5d\x78\x67\xc9\x08\xc9\x6b\x18\x8f\x93\x00\xe0\x81\x19\x68\x9c\x40\x0a\xe1\xc0\x4b\x38\x0b\x02\x80\x75\x25\x73\x12\x4a\xc2\x1a\x29\xdf\x2c\x15\x8f\x46\xb0\x0b\x00\xc0\xb9\x89\x5a\xe9\x7f\xe9\xc1\xcd\x3e\x0c\x60\x91\x56\xe2\x0e\x55\x45\x51\x97\xc5\xa9\x81\x23\x45\x61\xf2\x8d\xe9\x84\x69\x91\x38\x3a\x13\xa8\x74\xc1\x08\x97\x8a\x8f\x66\x50\x4f\xe0\xf9\x74\x3a\x9a\xf9\x3c\x75\xe0\xfe\xfa\x9d\x74\x57\xa3\x82\x11\xdb\xd7\x2b\x54\x5e\xdd\xa1\xa4\x98\x23\x5d\x95\xe8\xfe\xbd\xdc\xbe\x2d\x22\x4f\x7f\x14\x0b\x29\xd1\x2c\x56\xd7\x4b\x48\xc1\xc9\x66\xc7\x1e\xd2\x14\xdc\x4c\x0d\x0d\x4d\x3c\x0a\x59\xa8\xc7\xd8\xe6\x46\x95\xe5\x4a\x45\xd3\xc9\xa1\xd2\xad\x2a\xb6\x6d\x64\x81\x82\x6f\xa8\xd7\x32\xf4\x80\xcd\x4e\x2c\xf8\x98\x23\x8d\x1d\xce\xbf\xa4\xe5\xba\xc5\x8e\x97\x4f\xd1\x10\xbb\x98\xb6\xc8\x7e\x07\xab\x29\x35\xc0\xc5\x88\x41\x0a\xef\x6e\x3e\xbc\x8f\x35\x33\xb6\x8d\x36\x06\xba\x8d\x70\x67\x71\xf3\x35\xfb\x33\xe5\xf6\x39\x9c\x82\x1e\xc8\x8f\x78\x9f\xee\xcc\x3f\x4e\x31\x66\x5a\x1f\x3f\x9c\x93\x19\xb4\x9c\x4f\xd9\x38\xb2\x95\x29\x27\x90\xb3\xb2\xbc\x65\xf9\xd7\x7d\x1f\xee\x55\x3c\xdd\x95\x1b\x22\xdd\x64\x6b\x3f\x20\x05\x89\x8f\xf0\xe9\x7a\xb9\x20\xd2\x1f\xf1\xbe\x42\x4b\xd1\x68\x70\x27\x56\xd2\x20\x2b\xb6\x7e\x50\xf9\x86\x49\x8e\x90\xc2\x60\xa4\xad\x57\x07\x63\x2f\xf2\x12\xdf\xa8\x03\xf3\x0c\xce\xce\xba\x7c\x07\x5e\x17\xd3\xe9\x41\x0d\x5d\xd7\xbd\x24\x56\x2b\x69\x71\x85\x4f\xfb\x55\xdc\x83\xa8\x87\x2d\x6a\x94\x51\xf8\xe6\x6a\xe5\x16\xc9\x01\x20\x53\xe1\x91\x0d\x8b\xb2\xe8\x43\x3b\x22\xd9\xdf\xee\x60\x9e\x34\x3f\x35\x59\xf0\x2b\x00\x00\xff\xff\x43\xde\x7f\x32\x51\x05\x00\x00")

func templatesAppHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesAppHtml,
		"templates/app.html",
	)
}

func templatesAppHtml() (*asset, error) {
	bytes, err := templatesAppHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/app.html", size: 1361, mode: os.FileMode(420), modTime: time.Unix(1507924147, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesLayoutHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x6c\x91\x4d\x6e\x83\x30\x10\x85\xf7\x39\xc5\x48\xdd\x9a\x50\x7e\x1a\x11\x62\xe5\x06\xbd\x40\x77\xc6\x1e\xc7\xa8\xe0\x41\xd8\xb4\x41\x88\x8b\x74\xd7\xab\xf5\x24\x15\x71\x48\x59\x74\x37\x33\xef\xc9\xef\xd3\x33\x37\xbe\x6d\xce\x3b\x00\x6e\x50\xa8\x65\x00\xe0\xbe\xf6\x0d\x9e\x7f\xbe\xbe\xdf\x44\xc7\xe3\xb0\x05\xc5\xf9\x71\x9d\x01\x2a\x52\xe3\xa4\xc9\xfa\x32\x29\xba\x6b\x9c\xec\x73\x70\xa3\xf3\xd8\x46\x43\xcd\x9c\xb0\x2e\x72\xd8\xd7\x7a\xde\xd8\x59\xd7\xe3\xd4\x09\xa5\x6a\x7b\x29\x13\x6c\x57\x6d\x39\xdf\x5e\xda\x17\xd8\xc2\x2b\x59\x21\x89\xb5\x64\xc9\x75\x42\xe2\x89\x3e\xb0\xd7\x0d\x7d\x46\xd7\x52\x0c\x9e\xe6\xdd\x96\xa0\x12\xf2\xfd\xd2\xd3\x60\x55\x24\xa9\xa1\xbe\x7c\x4a\xb2\x34\xcb\x8a\xd3\x7d\xab\x2a\x79\x54\xc7\x35\xc9\x24\xcc\xa4\xcc\x64\xcc\xe4\xcc\xbc\x30\x73\x98\xee\x36\xad\x1f\xa4\xe2\xef\x26\xd3\x5c\x6f\x21\xff\x09\x3b\xa4\x45\xfe\x1c\x3c\x3c\x7e\x14\xc4\xe3\xb5\x50\xbe\x50\x86\xce\xa6\x09\xc6\x1a\x1b\x05\xf3\x7c\xb3\x04\x85\xc7\xe1\x13\x7e\x03\x00\x00\xff\xff\xd6\x6f\x86\x25\x8c\x01\x00\x00")

func templatesLayoutHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesLayoutHtml,
		"templates/layout.html",
	)
}

func templatesLayoutHtml() (*asset, error) {
	bytes, err := templatesLayoutHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/layout.html", size: 396, mode: os.FileMode(420), modTime: time.Unix(1507924147, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesLogHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x74\x53\x4f\x6f\x9b\x4e\x10\xbd\xf3\x29\x9e\x38\x44\xe0\x9f\x65\xac\x9f\xda\x93\x8d\x55\x45\x8a\x9a\x4a\xce\xa5\xe1\xd0\x1e\x37\x30\x60\xda\xf5\x2e\xdd\x1d\x12\xbb\x16\xdf\xbd\x5a\x58\x6c\xec\xa6\x37\x98\x3f\xef\xbd\x79\x33\xbb\x6e\x0c\xa1\x2e\xd2\x50\xea\x2a\xdc\x9c\x4e\x58\x6c\x75\x95\x89\x5a\xa2\xeb\xd6\x49\x63\x68\x13\x04\x6b\x9b\x9b\xba\x61\xf0\xb1\xa1\x34\x64\x3a\x70\xf2\x43\xbc\x8a\x21\x1a\x6e\x02\x20\x99\xcd\xf0\xc9\xa5\x4f\x2f\x5a\x4b\x12\xaa\xc3\x6c\x96\x04\xc0\xab\x30\x28\x6b\x63\x19\x29\xd8\xb4\xb4\x0a\x02\xa0\x6c\x55\xce\xb5\x56\x28\x89\xf3\xdd\x56\x57\x51\x8c\x53\x00\x00\x96\x38\xab\xf7\xa4\x5b\x8e\xce\x45\x2e\x89\x8a\x38\x0a\x93\xdf\xa2\x49\x44\x53\x27\x4e\xeb\x1c\x6d\x53\x08\xa6\xad\xae\xe2\x15\xba\x39\x3e\x2e\x97\xf1\x2a\x00\xba\x2b\x8a\x73\x51\x54\x08\x16\x23\xcf\x3f\xf5\x0e\x8a\x6d\x6e\xb4\x94\x48\x61\x77\xba\x95\xc5\x73\xff\x1b\xc5\xbd\x78\xa0\xd0\x79\xbb\x27\xc5\x8b\x8a\xf8\x41\x92\xfb\xbc\x3f\x7e\x29\xa2\xde\xc2\x78\x51\x2b\x45\xe6\x31\x7b\xda\x22\x85\xe3\xf4\x5d\x75\x89\x68\xc0\x1d\x45\xc0\xf3\x64\xfa\x5e\x33\xeb\x7d\xd4\xcb\xf7\x03\x60\x62\xce\xdf\x53\x5d\xeb\xf2\x78\x8e\xa1\xf7\xfa\x42\x30\x5a\x5f\x0a\x69\x69\xe5\x83\x86\xb8\x35\xca\xaf\x63\x20\x9c\x84\xa3\xb7\x5a\x15\xfa\xcd\x8f\x41\x75\xb5\x63\xfc\x07\x1f\x1c\x04\x7f\x8f\xb1\x49\x2f\x36\xbc\xe8\xe2\xb8\xd0\x65\x69\x89\x87\xfa\x77\xf4\xde\x0c\xea\x05\x5e\xa1\x66\x3a\x5a\xce\x6f\x50\x87\xcc\x80\xfa\x8e\x0d\xee\x2a\x5a\x23\xe7\xc8\x85\x94\x2f\x22\xff\x39\x02\xbb\x25\x1e\xf6\x72\xc7\xdc\x0c\x23\xfa\x1f\xa4\x50\xf4\x86\x6f\x4f\xdb\x47\xe6\xe6\x2b\xfd\x6a\xc9\xf2\xe8\xbb\xaf\x59\x68\x65\x48\x14\x47\xcb\x82\x29\xdf\x09\x55\x91\x73\x70\x7a\x8e\xde\x48\x67\xf8\xd8\xd4\xb7\x3c\xbb\x16\xa4\x29\x3e\xe0\xee\xee\x8c\xe7\x80\x5a\xeb\xc2\xff\x2f\x97\x97\x6e\x9c\x55\x4f\x40\x6c\xa3\x95\xa5\x8c\x0e\x1c\x8f\xeb\xea\x26\x3b\x3a\x4b\x6c\x48\x45\xe1\xe7\x87\xcc\x3d\x03\x67\x80\x5b\xe6\xcd\x18\x96\x54\x31\xbd\x9d\xc9\x39\xad\x93\xe1\xed\x6e\x82\x3f\x01\x00\x00\xff\xff\x5c\x6f\x05\xa1\xff\x03\x00\x00")

func templatesLogHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesLogHtml,
		"templates/log.html",
	)
}

func templatesLogHtml() (*asset, error) {
	bytes, err := templatesLogHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/log.html", size: 1023, mode: os.FileMode(420), modTime: time.Unix(1507924147, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _templatesNgrokHtml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x5c\x8e\xb1\x4a\x04\x31\x10\x86\xfb\x3c\xc5\xcf\xf6\x66\xb8\x2b\x25\x0e\x88\x85\x8a\x72\xc2\xe9\x35\x8a\x45\xe0\xb2\x97\xc5\x33\x1b\x92\x5c\x63\xc8\x8b\xd8\xf9\x6a\x3e\x89\xcc\x0a\x11\xac\x06\x66\xfe\xef\x9b\xdf\xf8\x15\x7f\x7f\x7e\x3d\xdb\x88\x33\xd4\x0a\x7d\x35\x87\x71\x3a\xe8\x9b\x39\x17\xb4\x66\xc8\xaf\x58\x29\xe3\xd7\xbc\xb9\xde\x3e\xdc\x19\xf2\x6b\x56\xaa\x56\x4c\x23\xf4\xe6\x90\xe6\x37\xb4\xa6\x00\x13\x59\x01\x80\xb1\xf0\xc9\x8d\x17\x83\xb8\x96\xbb\xde\x6d\xef\xd1\xda\xc0\xff\x37\x86\x2c\xe3\xa5\x03\xbe\x94\x78\x4e\xf4\x97\xba\xdc\xbf\x4f\xe1\x37\x4a\xb9\xd8\x72\xca\x03\x3f\x2e\x13\xbb\x5b\x81\x5f\xe5\x2f\x45\x96\x3a\xee\x98\x5d\x6f\xd2\x9d\xf4\x61\x23\x05\xb1\x89\x21\x95\x45\x90\x0a\x9e\x4e\x21\xb8\xa3\x38\x3a\x1f\xf6\x82\xff\x04\x00\x00\xff\xff\x64\x10\x0c\x2b\x0f\x01\x00\x00")

func templatesNgrokHtmlBytes() ([]byte, error) {
	return bindataRead(
		_templatesNgrokHtml,
		"templates/ngrok.html",
	)
}

func templatesNgrokHtml() (*asset, error) {
	bytes, err := templatesNgrokHtmlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "templates/ngrok.html", size: 271, mode: os.FileMode(420), modTime: time.Unix(1507924147, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"templates/502.html": templates502Html,
	"templates/app.html": templatesAppHtml,
	"templates/layout.html": templatesLayoutHtml,
	"templates/log.html": templatesLogHtml,
	"templates/ngrok.html": templatesNgrokHtml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}
var _bintree = &bintree{nil, map[string]*bintree{
	"templates": &bintree{nil, map[string]*bintree{
		"502.html": &bintree{templates502Html, map[string]*bintree{}},
		"app.html": &bintree{templatesAppHtml, map[string]*bintree{}},
		"layout.html": &bintree{templatesLayoutHtml, map[string]*bintree{}},
		"log.html": &bintree{templatesLogHtml, map[string]*bintree{}},
		"ngrok.html": &bintree{templatesNgrokHtml, map[string]*bintree{}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}

