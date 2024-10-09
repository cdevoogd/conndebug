package command

type Root struct {
	HTTP      HTTP      `cmd:"" name:"http" help:"send an HTTP request"`
	HTTPTrace HTTPTrace `cmd:"" name:"httptrace" help:"trace an HTTP GET request"`
	Reachable Reachable `cmd:"" name:"reachable" help:"attempt to connect and immediately close a connection to test if an address is reachable"`
}
