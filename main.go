package main

import (
	"github.com/alecthomas/kong"
	"github.com/cdevoogd/conndebug/internal/command"
)

func main() {
	rootCommand := command.Root{}
	ctx := kong.Parse(&rootCommand)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
