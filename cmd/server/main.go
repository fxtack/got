package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"got/internal"
	"os"
)

var version string

func main() {
	app := cli.NewApp()
	app.Version = version
	app.Usage = "start Got server"
	app.Flags = []cli.Flag{
		&cli.IntFlag{
			Name:    "port, p",
			Aliases: []string{"p"},
			Value:   9876,
			Usage:   "server port",
		},
	}
	app.Action = func(ctx *cli.Context) error {
		var port = ctx.Int("port")
		srv, err := internal.CreateServer(port)
		if err != nil {
			return err
		}
		fmt.Printf("Got server started at %d\n", port)
		return srv.Run()
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		return
	}
}
