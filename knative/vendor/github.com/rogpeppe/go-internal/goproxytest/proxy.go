// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package goproxytest serves Go modules from a proxy server designed to run on
localhost during tests, both to make tests avoid requiring specific network
servers and also to make them significantly faster.

Each module archive is either a file named path_vers.txt or a directory named
path_vers, where slashes in path have been replaced with underscores. The
archive or directory must contain two files ".info" and ".mod", to be served as
the info and mod files in the proxy protocol (see
https://research.swtch.com/vgo-module).  The remaining files are served as the
content of the module zip file.  The path@vers prefix required of files in the
zip file is added automatically by the proxy: the files in the archive have
names without the prefix, like plain "go.mod", "x.go", and so on.

See ../cmd/txtar-addmod and ../cmd/txtar-savedir for tools generate txtar
files, although it's fine to write them by hand.
*/
package goproxytest

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rogpeppe/go-internal/module"
	"github.com/rogpeppe/go-internal/par"
	"github.com/rogpeppe/go-internal/semver"
	"github.com/rogpeppe/go-internal/txtar"
)

type Server struct {
	server       *http.Server
	URL          string
	dir          string
	modList      []module.Version
	zipCache     par.Cache
	archiveCache par.Cache
}

// StartProxy starts the Go module proxy listening on the given
// network address. It serves modules taken from the given directory
// name. If addr is empty, it will listen on an arbitrary
// localhost port. If dir is empty, "testmod" will be used.
//
// The returned Server should be closed after use.
func NewServer(dir, addr string) (*Server, error) {
	var srv Server
	if addr == "" {
		addr = "localhost:0"
	}
	if dir == "" {
		dir = "testmod"
	}
	srv.dir = dir
	if err := srv.readModList(); err != nil {
		return nil, fmt.Errorf("cannot read modules: %v", err)
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("cannot listen on %q: %v", addr, err)
	}
	srv.server = &http.Server{
		Handler: http.HandlerFunc(srv.handler),
	}
	addr = l.Addr().String()
	srv.URL = "http://" + addr + "/mod"
	go func() {
		if err := srv.server.Serve(l); err != nil && err != http.ErrServerClosed {
			log.Printf("go proxy: http.Serve: %v", err)
		}
	}()
	return &srv, nil
}

// Close shuts down the proxy.
func (srv *Server) Close() {
	srv.server.Close()
}

func (srv *Server) readModList() error {
	infos, err := ioutil.ReadDir(srv.dir)
	if err != nil {
		return err
	}
	for _, info := range infos {
		name := info.Name()
		if !strings.HasSuffix(name, ".txt") && !info.IsDir() {
			continue
		}
		name = strings.TrimSuffix(name, ".txt")
		i := strings.LastIndex(name, "_v")
		if i < 0 {
			continue
		}
		encPath := strings.Replace(name[:i], "_", "/", -1)
		path, err := module.DecodePath(encPath)
		if err != nil {
			return fmt.Errorf("cannot decode module path in %q: %v", name, err)
		}
		encVers := name[i+1:]
		vers, err := module.DecodeVersion(encVers)
		if err != nil {
			return fmt.Errorf("cannot decode module version in %q: %v", name, err)
		}
		srv.modList = append(srv.modList, module.Version{Path: path, Version: vers})
	}
	return nil
}

// handler serves the Go module proxy protocol.
// See the proxy section of https://research.swtch.com/vgo-module.
func (srv *Server) handler(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/mod/") {
		http.NotFound(w, r)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/mod/")
	i := strings.Index(path, "/@v/")
	if i < 0 {
		http.NotFound(w, r)
		return
	}
	enc, file := path[:i], path[i+len("/@v/"):]
	path, err := module.DecodePath(enc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "go proxy_test: %v\n", err)
		http.NotFound(w, r)
		return
	}
	if file == "list" {
		n := 0
		for _, m := range srv.modList {
			if m.Path == path && !isPseudoVersion(m.Version) {
				if err := module.Check(m.Path, m.Version); err == nil {
					fmt.Fprintf(w, "%s\n", m.Version)
					n++
				}
			}
		}
		if n == 0 {
			http.NotFound(w, r)
		}
		return
	}

	i = strings.LastIndex(file, ".")
	if i < 0 {
		http.NotFound(w, r)
		return
	}
	encVers, ext := file[:i], file[i+1:]
	vers, err := module.DecodeVersion(encVers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "go proxy_test: %v\n", err)
		http.NotFound(w, r)
		return
	}

	if allHex(vers) {
		var best string
		// Convert commit hash (only) to known version.
		// Use latest version in semver priority, to match similar logic
		// in the repo-based module server (see modfetch.(*codeRepo).convert).
		for _, m := range srv.modList {
			if m.Path == path && semver.Compare(best, m.Version) < 0 {
				var hash string
				if isPseudoVersion(m.Version) {
					hash = m.Version[strings.LastIndex(m.Version, "-")+1:]
				} else {
					hash = srv.findHash(m)
				}
				if strings.HasPrefix(hash, vers) || strings.HasPrefix(vers, hash) {
					best = m.Version
				}
			}
		}
		if best != "" {
			vers = best
		}
	}

	a := srv.readArchive(path, vers)
	if a == nil {
		fmt.Fprintf(os.Stderr, "go proxy: no archive %s %s\n", path, vers)
		http.Error(w, "cannot load archive", 500)
		return
	}

	switch ext {
	case "info", "mod":
		want := "." + ext
		for _, f := range a.Files {
			if f.Name == want {
				w.Write(f.Data)
				return
			}
		}

	case "zip":
		type cached struct {
			zip []byte
			err error
		}
		c := srv.zipCache.Do(a, func() interface{} {
			var buf bytes.Buffer
			z := zip.NewWriter(&buf)
			for _, f := range a.Files {
				if strings.HasPrefix(f.Name, ".") {
					continue
				}
				zf, err := z.Create(path + "@" + vers + "/" + f.Name)
				if err != nil {
					return cached{nil, err}
				}
				if _, err := zf.Write(f.Data); err != nil {
					return cached{nil, err}
				}
			}
			if err := z.Close(); err != nil {
				return cached{nil, err}
			}
			return cached{buf.Bytes(), nil}
		}).(cached)

		if c.err != nil {
			fmt.Fprintf(os.Stderr, "go proxy: %v\n", c.err)
			http.Error(w, c.err.Error(), 500)
			return
		}
		w.Write(c.zip)
		return

	}
	http.NotFound(w, r)
}

func (srv *Server) findHash(m module.Version) string {
	a := srv.readArchive(m.Path, m.Version)
	if a == nil {
		return ""
	}
	var data []byte
	for _, f := range a.Files {
		if f.Name == ".info" {
			data = f.Data
			break
		}
	}
	var info struct{ Short string }
	json.Unmarshal(data, &info)
	return info.Short
}

func (srv *Server) readArchive(path, vers string) *txtar.Archive {
	enc, err := module.EncodePath(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "go proxy: %v\n", err)
		return nil
	}
	encVers, err := module.EncodeVersion(vers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "go proxy: %v\n", err)
		return nil
	}

	prefix := strings.Replace(enc, "/", "_", -1)
	name := filepath.Join(srv.dir, prefix+"_"+encVers+".txt")
	a := srv.archiveCache.Do(name, func() interface{} {
		a, err := txtar.ParseFile(name)
		if os.IsNotExist(err) {
			// we fallback to trying a directory
			name = strings.TrimSuffix(name, ".txt")

			a = new(txtar.Archive)

			err = filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if path == name && !info.IsDir() {
					return fmt.Errorf("expected a directory root")
				}
				if info.IsDir() {
					return nil
				}
				arpath := filepath.ToSlash(strings.TrimPrefix(path, name+string(os.PathSeparator)))
				data, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}
				a.Files = append(a.Files, txtar.File{
					Name: arpath,
					Data: data,
				})
				return nil
			})
		}
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "go proxy: %v\n", err)
			}
			a = nil
		}
		return a
	}).(*txtar.Archive)
	return a
}
