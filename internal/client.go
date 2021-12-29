package internal

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"got/pkg"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func CreateClient(addr string) (GotClient, error) {
	client := &defaultClient{addr: addr}
	return client, client.Init()
}

type GotClient interface {
	Init() error
	ListFiles() (string, error)
	ChangeDir(dstDir string) (string, error)
	UploadFile(filePath string) error
	DownloadFile(filePath string) error
}

type defaultClient struct {
	addr       string
	grpcClient GotServiceClient
}

func (d *defaultClient) Init() error {
	creds := insecure.NewCredentials()
	options := []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	conn, err := grpc.Dial(d.addr, options...)
	if err != nil {
		return err
	}
	d.grpcClient = NewGotServiceClient(conn)
	return err
}

func (d *defaultClient) ListFiles() (string, error) {
	resp, err := d.grpcClient.ListFile(context.Background(),
		&ListFilesRequest{})
	if err != nil {
		return "", nil
	}
	return resp.Info, nil
}

func (d *defaultClient) ChangeDir(dstDir string) (string, error) {
	resp, err := d.grpcClient.ChangeDir(context.Background(),
		&ChangeDirRequest{DstDir: dstDir})
	if err != nil {
		return "", err
	}
	return resp.Info, nil
}

func (d *defaultClient) UploadFile(filePath string) error {
	// get file information
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// metadata map
	var mdMap = make(map[string]string)

	// if the specified upload is a directory
	if info.IsDir() {
		// make new name for tar file
		dirTarPath := filepath.Join(filepath.Dir(filePath),
			fmt.Sprintf("%s%d.tar", filepath.Base(filePath), time.Now().Unix()))

		// pack directory as temporary tar file for transfer
		err := pkg.Tar(filePath, dirTarPath)
		if err != nil {
			return err
		}

		// set up information for new tar file
		mdMap["type"] = dirType
		filePath = dirTarPath
		info, err = os.Stat(dirTarPath)

		// remove temporary tar file.
		defer func() {
			err = os.Remove(filePath)
		}()
	}
	mdMap["name"] = filePath

	// open file which will be transfer
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0664)
	if err != nil {
		return err
	}
	defer file.Close()

	// prepare metadata and grpc stream
	md := metadata.New(mdMap)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	stream, err := d.grpcClient.UploadFile(ctx)
	if err != nil {
		return err
	}

	// prepare process bar
	var pushCh = make(chan int64, 2)
	var procCtx, cancel = context.WithCancel(context.Background())
	defer func() {
		cancel()
		close(pushCh)
	}()
	procBar, _ := pkg.ProcessBar("upload", 0, info.Size(), pushCh, procCtx)

	// data transfer
	chunk := make([]byte, 4*(1<<10))
	for {
		n, err := file.Read(chunk)
		if err == io.EOF {
			break
		} else if err != nil {
			cancel()
			return err
		}
		if n < len(chunk) {
			chunk = chunk[:n]
		}
		err = stream.Send(&UploadFileRequest{Data: chunk})
		if err != nil {
			cancel()
			return err
		}
		pushCh <- int64(len(chunk))
	}
	<-procBar

	// close stream
	_, err = stream.CloseAndRecv()
	return err
}

func (d *defaultClient) DownloadFile(filePath string) error {
	stream, err := d.grpcClient.DownloadFile(context.Background(),
		&DownloadFileRequest{Filepath: filePath})
	if err != nil {
		return err
	}

	// get metadata from header
	md, err := stream.Header()
	if err != nil {
		return err
	}
	if e := md.Get("err"); e != nil {
		return errors.New(e[0])
	}
	// get size
	var size int64
	if s := md.Get("size"); s != nil {
		size, err = strconv.ParseInt(s[0], 0, 64)
		if err != nil {
			return err
		}
	}
	// get download file's type
	var downloadType string
	if t := md.Get("type"); t != nil {
		if t[0] == dirType {
			filePath = fmt.Sprintf("%s.tar", filePath)
		}
		downloadType = t[0]
	}

	// new file for receiving
	file, err := os.OpenFile(filepath.Base(filePath), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Println(err)
		return err
	}
	defer func() {
		_ = file.Close()
		// if the specified download is a directory, remove temporary tar file
		if downloadType == dirType {
			err = os.Remove(filePath)
		}
	}()

	// prepare process bar
	var pushCh = make(chan int64, 2)
	var procCtx, cancel = context.WithCancel(context.Background())
	defer func() {
		cancel()
		close(pushCh)
	}()
	procBar, _ := pkg.ProcessBar("download", 0, size, pushCh, procCtx)

	// receive data
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			_ = os.Remove(file.Name())
			cancel()
			return err
		}
		_, err = file.Write(resp.Data)
		if err != nil {
			_ = os.Remove(file.Name())
			cancel()
			return err
		}
		pushCh <- int64(len(resp.Data))
	}
	<-procBar

	// if the specified download is a directory, unpack the tar file as a directory
	if downloadType == dirType {
		if err = pkg.UnTar(filePath, "."); err != nil {
			return err
		}
	}
	return err
}
