package main

import (
	"net/http"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/cdevoogd/conndebug/internal/command"
)

var supportedHTTPMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPut,
	http.MethodPost,
	http.MethodPatch,
	http.MethodDelete,
}

func main() {
	rootCommand := command.Root{}
	ctx := kong.Parse(&rootCommand, kong.Vars{
		"http_methods": strings.Join(supportedHTTPMethods, ","),
	})
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
