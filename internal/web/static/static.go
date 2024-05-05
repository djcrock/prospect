package static

import (
	"embed"
	"net/http"
)

//go:embed *.css vendor
var files embed.FS

var FileServer = http.FileServer(http.FS(files))
