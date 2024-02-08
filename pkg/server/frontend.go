package server

import (
	"context"
	"io/fs"
	"net/http"
	"strings"

	static "github.com/ethpandaops/tracoor/web"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/sirupsen/logrus"
)

type Frontend struct {
	log logrus.FieldLogger

	handler    http.Handler
	filesystem fs.FS
}

func NewFrontend(log logrus.FieldLogger) *Frontend {
	return &Frontend{
		log: log,
	}
}

func (f *Frontend) Start() error {
	frontendFS, err := fs.Sub(static.FS, "build/frontend")
	if err != nil {
		return err
	}

	f.filesystem = frontendFS
	f.handler = http.FileServer(http.FS(f.filesystem))

	return nil
}

func (f *Frontend) customRoutingErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, req *http.Request, code int) {
	if f.handler != nil && code == http.StatusNotFound {
		if f.fileExists(strings.TrimPrefix(req.URL.Path, "/")) {
			f.handler.ServeHTTP(w, req)
		} else {
			req.URL.Path = "/"
			f.handler.ServeHTTP(w, req)
		}
		return
	}

	runtime.DefaultRoutingErrorHandler(ctx, mux, marshaler, w, req, code)
}

func (f *Frontend) fileExists(path string) bool {
	_, err := fs.Stat(f.filesystem, path)
	return err == nil
}
