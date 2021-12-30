package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"got/internal"
	"net"
	"os"
	"path/filepath"
	"time"
)

const defaultPort = "9876"

var version = ""

func main() {
	app := cli.NewApp()
	app.Version = version
	app.Usage = "got is a simply tool for upload file to remote server"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "addr, a",
			Aliases:  []string{"a"},
			Usage:    "Got server address",
			Required: true,
		},
		&cli.BoolFlag{
			Name:     "time, t",
			Aliases:  []string{"t"},
			Usage:    "show time cost",
			Value:    false,
			Required: false,
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:    "list",
			Aliases: []string{"l", "ls", "ll"},
			Usage:   "list remote directory content",
			Action:  list,
		},
		{
			Name:    "change",
			Aliases: []string{"c", "cd"},
			Usage:   "change remote directory content",
			Action:  change,
		},
		{
			Name:    "upload",
			Aliases: []string{"u", "up"},
			Usage:   "upload file to remote directory",
			Action:  upload,
		},
		{
			Name:    "download",
			Aliases: []string{"d", "down"},
			Usage:   "download file from remote directory",
			Action:  download,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func createClient(ctx *cli.Context) (internal.GotClient, error) {
	addr := ctx.String("addr")
	addr = parseAddr(addr)
	return internal.CreateClient(addr)
}

func parseAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		addr = net.JoinHostPort(addr, defaultPort)
	} else {
		if port == "" {
			addr = net.JoinHostPort(host, defaultPort)
		}
	}
	return addr
}

func list(ctx *cli.Context) error {
	now := time.Now()

	gotClient, err := createClient(ctx)
	if err != nil {
		return err
	}

	filesInfo, err := gotClient.ListFiles()
	if err != nil {
		return err
	}
	fmt.Println(filesInfo)

	cost := time.Since(now)
	if ctx.Bool("time") {
		fmt.Printf("cost: %s\n", cost)
	}
	return nil
}

func change(ctx *cli.Context) error {
	now := time.Now()

	gotClient, err := createClient(ctx)
	if err != nil {
		return err
	}

	dstDir := ctx.Args().First()
	dirInfo, err := gotClient.ChangeDir(dstDir)
	if err != nil {
		return err
	}
	fmt.Println(dirInfo)

	cost := time.Since(now)
	if ctx.Bool("time") {
		fmt.Printf("cost: %s\n", cost)
	}
	return nil
}

func upload(ctx *cli.Context) error {
	now := time.Now()

	gotClient, err := createClient(ctx)
	if err != nil {
		return err
	}

	filePath := filepath.Clean(ctx.Args().First())
	err = gotClient.UploadFile(filePath)
	if err != nil {
		return err
	}

	cost := time.Since(now)
	if ctx.Bool("time") {
		fmt.Printf("cost: %s\n", cost)
	}
	return nil
}

func download(ctx *cli.Context) error {
	now := time.Now()

	gotClient, err := createClient(ctx)
	if err != nil {
		return err
	}

	filePath := filepath.Clean(ctx.Args().First())
	err = gotClient.DownloadFile(filePath)
	if err != nil {
		return err
	}

	cost := time.Since(now)
	if ctx.Bool("time") {
		fmt.Printf("cost: %s\n", cost)
	}
	return nil
}
