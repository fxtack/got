package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"got/internal/client"
	"os"
	"time"
)

func main() {
	app := cli.NewApp()
	app.Usage = "got is a simply tool for upload file to remote server"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "addr, a",
			Aliases:  []string{"a"},
			Usage:    "Got server address",
			Required: true,
		},
		&cli.BoolFlag{
			Name: "time, t",
			Aliases: []string{"t"},
			Usage: "show time cost",
			Value: false,
			Required: false,
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "list remote directory content",
			Action:  list,
		},
		{
			Name:    "change",
			Aliases: []string{"c"},
			Usage:   "change remote directory content",
			Action:  change,
		},
		{
			Name:    "upload",
			Aliases: []string{"u"},
			Usage:   "upload file to remote directory",
			Action:  upload,
		},
		{
			Name:    "download",
			Aliases: []string{"d"},
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

func createClient(ctx *cli.Context) (client.GotClient, error) {
	addr := ctx.String("addr")
	return client.Create(addr)
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

	filePath := ctx.Args().First()
	err = gotClient.UploadFile(filePath)
	if err != nil {
		return err
	}
	fmt.Println("upload finish")

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

	filePath := ctx.Args().First()
	err = gotClient.DownloadFile(filePath)
	if err != nil {
		return err
	}
	fmt.Println("download finish")

	cost := time.Since(now)
	if ctx.Bool("time") {
		fmt.Printf("cost: %s\n", cost)
	}
	return nil
}
