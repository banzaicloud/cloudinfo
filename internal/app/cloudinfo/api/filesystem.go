// Copyright © 2020 Banzai Cloud
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

package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-contrib/static"
)

func fileSystem(fsys fs.FS) static.ServeFileSystem {
	return staticFileSystem{
		httpfsys: http.FS(fsys),
		fsys:     fsys,
	}
}

type staticFileSystem struct {
	httpfsys http.FileSystem
	fsys     fs.FS
}

func (f staticFileSystem) Open(name string) (http.File, error) {
	return f.httpfsys.Open(name)
}

func (f staticFileSystem) Exists(prefix string, filePath string) bool {
	if p := strings.TrimPrefix(filePath, prefix); len(p) < len(filePath) {
		_, err := fs.Stat(f.fsys, strings.TrimLeft(p, "/"))

		return err == nil
	}

	return false
}
