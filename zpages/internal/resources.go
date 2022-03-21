// Code generated by "esc -pkg internal -o resources.go templates/"; DO NOT EDIT.

package internal

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type _escLocalFS struct{}

var _escLocal _escLocalFS

type _escStaticFS struct{}

var _escStatic _escStaticFS

type _escDirectory struct {
	fs   http.FileSystem
	name string
}

type _escFile struct {
	compressed string
	size       int64
	modtime    int64
	local      string
	isDir      bool

	once sync.Once
	data []byte
	name string
}

func (_escLocalFS) Open(name string) (http.File, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_escStaticFS) prepare(name string) (*_escFile, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		f.name = path.Base(name)
		if f.size == 0 {
			return
		}
		var gr *gzip.Reader
		b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(f.compressed))
		gr, err = gzip.NewReader(b64)
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (fs _escStaticFS) Open(name string) (http.File, error) {
	f, err := fs.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (dir _escDirectory) Open(name string) (http.File, error) {
	return dir.fs.Open(dir.name + name)
}

func (f *_escFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*_escFile
	}
	return &httpFile{
		Reader:   bytes.NewReader(f.data),
		_escFile: f,
	}, nil
}

func (f *_escFile) Close() error {
	return nil
}

func (f *_escFile) Readdir(count int) ([]os.FileInfo, error) {
	if !f.isDir {
		return nil, fmt.Errorf(" escFile.Readdir: '%s' is not directory", f.name)
	}

	fis, ok := _escDirs[f.local]
	if !ok {
		return nil, fmt.Errorf(" escFile.Readdir: '%s' is directory, but we have no info about content of this dir, local=%s", f.name, f.local)
	}
	limit := count
	if count <= 0 || limit > len(fis) {
		limit = len(fis)
	}

	if len(fis) == 0 && count > 0 {
		return nil, io.EOF
	}

	return fis[0:limit], nil
}

func (f *_escFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *_escFile) Name() string {
	return f.name
}

func (f *_escFile) Size() int64 {
	return f.size
}

func (f *_escFile) Mode() os.FileMode {
	return 0
}

func (f *_escFile) ModTime() time.Time {
	return time.Unix(f.modtime, 0)
}

func (f *_escFile) IsDir() bool {
	return f.isDir
}

func (f *_escFile) Sys() interface{} {
	return f
}

// FS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func FS(useLocal bool) http.FileSystem {
	if useLocal {
		return _escLocal
	}
	return _escStatic
}

// Dir returns a http.Filesystem for the embedded assets on a given prefix dir.
// If useLocal is true, the filesystem's contents are instead used.
func Dir(useLocal bool, name string) http.FileSystem {
	if useLocal {
		return _escDirectory{fs: _escLocal, name: name}
	}
	return _escDirectory{fs: _escStatic, name: name}
}

// FSByte returns the named file from the embedded assets. If useLocal is
// true, the filesystem's contents are instead used.
func FSByte(useLocal bool, name string) ([]byte, error) {
	if useLocal {
		f, err := _escLocal.Open(name)
		if err != nil {
			return nil, err
		}
		b, err := ioutil.ReadAll(f)
		_ = f.Close()
		return b, err
	}
	f, err := _escStatic.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.data, nil
}

// FSMustByte is the same as FSByte, but panics if name is not present.
func FSMustByte(useLocal bool, name string) []byte {
	b, err := FSByte(useLocal, name)
	if err != nil {
		panic(err)
	}
	return b
}

// FSString is the string version of FSByte.
func FSString(useLocal bool, name string) (string, error) {
	b, err := FSByte(useLocal, name)
	return string(b), err
}

// FSMustString is the string version of FSMustByte.
func FSMustString(useLocal bool, name string) string {
	return string(FSMustByte(useLocal, name))
}

var _escData = map[string]*_escFile{

	"/templates/footer.html": {
		name:    "footer.html",
		local:   "templates/footer.html",
		size:    16,
		modtime: 1647459395,
		compressed: `
H4sIAAAAAAAC/7LRT8pPqbTjstHPKMnNseMCBAAA//8ATCBFEAAAAA==
`,
	},

	"/templates/header.html": {
		name:    "header.html",
		local:   "templates/header.html",
		size:    479,
		modtime: 1647459395,
		compressed: `
H4sIAAAAAAAC/5TRv04EIRAG8H6fAmnNgRsbY1gs1MLCaHGNJbLDMh5/NjBnsrncuxuyp4naaMWEj/yY
5FNnd0+325fne+YpBt2pdrBg0jRwSFwrD2bUHWOMqQhkmPWmVKCB78ltrvgpIqQA+nAQ2zYcj0quN2sa
MO1YgTDw6nMhuyeGNifOfAE3cE8012sp8wyJIEAEKovALJ15b+/q5yDQZi5/o7QEqB6AfoouJ6piynkK
YGaswuYom3TjTMSwDI+GoKAJ5w/tH/4P2uYRxAQUx9BW7cWluJDxxAlMI055M2PaiYhJ2PqFV1twJjaC
g8JqsX8lG/NWuVZyFXSn5FqOes3j0qrrvzXge919BAAA//9o89W63wEAAA==
`,
	},

	"/templates/summary.html": {
		name:    "summary.html",
		local:   "templates/summary.html",
		size:    1631,
		modtime: 1647459395,
		compressed: `
H4sIAAAAAAAC/6yVTY/aMBCG7/srRiniVGC3t9LEVSvtbdXDbm9VD04yhAgziWzDFlz/98of4SObttqF
C3Kc8cw87+shqea5QFB6JzBL8kaWKCeq5UVN1RxuE3YDAJBqGRbhoYSiEarllN0BF3VFmcCFZmnOnlpO
8I2vMZ3lLJ3p8uwYG1Ou2k/h9/fpgw8dSpwUSBpl4pI/bohqqt6SerD5jwM1HrhGKnbwxNetQHW1WkM8
91I2crhSOusU/7v0V2rsbWmMkZwqhGkU7OumWKF2zitrU738J/gPY6bW/ozIS2YMUmntNVGCgMaMOMwz
mH6XvEB1T2Xb1KStdW9ETSvl3z64ld8MUCPZPNdU4q/3fuljHpvnEDKBEfE1+k3HGzfrBeAW6XjWySAP
g8WLVSWbDZVzeIeIiWMWCn1Q4IdJVMDRGxNzd4PxPz1iCxHqJNUL+TksJS6yxGlj7ee9k84BZcZ4MGvH
e71rMbt1TU6/FLre+kY4O5ZyvfcbPkYeonpUr2Y4GNK5seViE6SP984JeAHj3XgvQqLcX2AXEN1z9KHe
CXx3U8806M+C74n5O/7y0KWSXGzrhz5y8Nn/Gakz9wY8HoiKQN3AedR05j8r7OZPAAAA//8Uml6vXwYA
AA==
`,
	},

	"/templates/traces.html": {
		name:    "traces.html",
		local:   "templates/traces.html",
		size:    420,
		modtime: 1647459395,
		compressed: `
H4sIAAAAAAAC/4yQsU7EMBBEe3/FKhLSXUHCpaA4jDuQaCgCErUdLxDh2Mab6EBm/x05RwHKFWxljWdG
miejkkY9RO3hXo+4h5zr8mAG2Rglm6iEjKqo88gMHb7PSBP9fCRU4ukVPZy+G6cjoYUNYb8V5/88kXPS
/gWh7sKBmHOOafDTM1Rn7SVVsBm8xQ+obwd0luBiywy/PLt27dkdPX/FtlTXXxS178KBWeSM3jLLZtkl
TVq2SwM0fTq8rvrgQtobN+NVpR6T7vHOFkowovYEpMfo0EI6MqpFQbBO6/7tZNyHaVWxYP4OAAD//55D
m5KkAQAA
`,
	},

	"/templates": {
		name:  "templates",
		local: `templates/`,
		isDir: true,
	},
}

var _escDirs = map[string][]os.FileInfo{

	"templates/": {
		_escData["/templates/footer.html"],
		_escData["/templates/header.html"],
		_escData["/templates/summary.html"],
		_escData["/templates/traces.html"],
	},
}
