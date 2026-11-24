package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist all:static
var files embed.FS

func Handler() http.Handler {
	dir := "dist"
	if _, e := files.Open("dist/index.html"); e != nil {
		dir = "static"
	}
	root, e := fs.Sub(files, dir)
	if e != nil {
		panic(e)
	}
	return http.FileServer(http.FS(root))
}
