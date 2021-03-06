package main

import (
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/maxence-charriere/app/internal/http"
	"github.com/pkg/errors"
	"github.com/segmentio/conf"
)

type buildConfig struct {
	Force   bool `conf:"force" help:"Force rebuilding of package that are already up-to-date."`
	Race    bool `conf:"race"  help:"Enable data race detection."`
	Verbose bool `conf:"v"     help:"Enable verbose mode."`

	rootDir string
}

func buildProject(ctx context.Context, args []string) {
	c := buildConfig{}

	ld := conf.Loader{
		Name:    "goapp build",
		Args:    args,
		Usage:   "[options...] [package]",
		Sources: []conf.Source{conf.NewEnvSource("GOAPP", os.Environ()...)},
	}

	_, args = conf.LoadWith(&c, ld)
	verbose = c.Verbose

	pkg := "."
	if len(args) != 0 {
		pkg = args[0]
	}

	rootDir, err := filepath.Abs(pkg)
	if err != nil {
		fail("%s", err)
	}
	c.rootDir = rootDir

	if err := build(ctx, c); err != nil {
		fail("%s", err)
	}

	success("build succeeded")
}

func build(ctx context.Context, c buildConfig) error {
	log("building wasm app")
	if err := buildWasm(ctx, c); err != nil {
		return err
	}

	log("building server")
	if err := buildServer(ctx, c); err != nil {
		return err
	}

	log("installing wasm_exec.js")
	if err := installWasmExec(c.rootDir); err != nil {
		return err
	}

	log("generating etag")
	if err := generateEtag(c.rootDir); err != nil {
		return err
	}

	log("compressing static resources")
	return compressStaticResources(c.rootDir)
}

func buildWasm(ctx context.Context, c buildConfig) error {
	pkgName := filepath.Base(c.rootDir) + "-wasm"
	pkg := filepath.Join(c.rootDir, "cmd", pkgName)
	out := filepath.Join(c.rootDir, "web", "goapp.wasm")

	os.Setenv("GOOS", "js")
	os.Setenv("GOARCH", "wasm")
	defer os.Unsetenv("GOOS")
	defer os.Unsetenv("GOARCH")

	cmd := []string{
		"go", "build",
		"-o", out,
	}

	if c.Force {
		cmd = append(cmd, "-a")
	}

	if c.Verbose {
		cmd = append(cmd, "-v")
	}

	cmd = append(cmd, pkg)
	return execute(ctx, cmd[0], cmd[1:]...)
}

func buildServer(ctx context.Context, c buildConfig) error {
	pkgName := filepath.Base(c.rootDir) + "-server"
	pkg := filepath.Join(c.rootDir, "cmd", pkgName)

	out := filepath.Join(c.rootDir, pkgName)
	if runtime.GOOS == "windows" {
		out += ".exe"
	}

	cmd := []string{
		"go", "build",
		"-o", out,
	}

	if c.Force {
		cmd = append(cmd, "-a")
	}

	if c.Race {
		cmd = append(cmd, "-race")
	}

	if c.Verbose {
		cmd = append(cmd, "-v")
	}

	cmd = append(cmd, pkg)
	return execute(ctx, cmd[0], cmd[1:]...)
}

func installWasmExec(rootDir string) error {
	wasmExec := filepath.Join(runtime.GOROOT(), "misc", "wasm", "wasm_exec.js")
	webWasmExec := filepath.Join(rootDir, "web", filepath.Base(wasmExec))

	src, err := os.Open(wasmExec)
	if err != nil {
		return errors.Wrapf(err, "opening %s failed", wasmExec)
	}
	defer src.Close()

	dst, err := os.Create(webWasmExec)
	if err != nil {
		return errors.Wrapf(err, "creating %s failed", webWasmExec)
	}
	defer src.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return errors.Wrapf(err, "copying %s failed", wasmExec)
	}
	return nil
}

func compressStaticResources(rootDir string) error {
	walk := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if !gzipRequired(path) {
			return nil
		}

		log("gzipping %s", path)

		src, err := os.Open(path)
		if err != nil {
			return errors.Wrapf(err, "opening %s failed", path)
		}
		defer src.Close()

		filename := path + ".gz"
		dst, err := os.Create(filename)
		if err != nil {
			return errors.Wrapf(err, "creating %s failed", filename)
		}
		defer dst.Close()

		gz := gzip.NewWriter(dst)
		defer gz.Close()

		if _, err := io.Copy(gz, src); err != nil {
			return errors.Wrapf(err, "compressing %s failed", path)
		}
		return nil
	}

	return filepath.Walk(filepath.Join(rootDir, "web"), walk)
}

func gzipRequired(filename string) bool {
	mimeType := mime.TypeByExtension(filepath.Ext(filename))

	allowedMimeTypes := []string{
		"application/javascript",
		"application/json",
		"application/wasm",
		"application/x-javascript",
		"application/x-tar",
		"image/svg+xml",
		"text/css",
		"text/html",
		"text/plain",
		"text/xml",
	}

	for _, m := range allowedMimeTypes {
		if strings.Contains(mimeType, m) {
			return true
		}
	}

	return false
}

func generateEtag(rootDir string) error {
	etagname := filepath.Join(rootDir, "web", ".etag")
	if err := ioutil.WriteFile(etagname, []byte(http.GenerateEtag()), 0666); err != nil {
		return errors.Wrap(err, "generating etag failed")
	}
	return nil
}
