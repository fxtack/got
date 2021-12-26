package client

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"got/internal/pb"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

func Create(addr string) (GotClient, error) {
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
	addr string
	grpcClient pb.GotServiceClient
}

func (d *defaultClient) Init() error {
	creds := insecure.NewCredentials()
	options := []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	conn, err :=  grpc.Dial(d.addr, options ...)
	if err != nil {
		return err
	}
	d.grpcClient = pb.NewGotServiceClient(conn)
	return err
}

func (d *defaultClient) ListFiles() (string, error) {
	resp, err := d.grpcClient.ListFile(context.Background(),
		&pb.ListFilesRequest{})
	if err != nil {
		return "", nil
	}
	return resp.Info, nil
}

func (d *defaultClient) ChangeDir(dstDir string) (string, error) {
	resp, err := d.grpcClient.ChangeDir(context.Background(),
		&pb.ChangeDirRequest{DstDir: dstDir})
	if err != nil {
		return "", err
	}
	return resp.Info, nil
}

func (d *defaultClient) UploadFile(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0664)
	if err != nil {
		return err
	}
	defer file.Close()

	md := metadata.New(map[string]string{"name": filepath.Base(filePath)})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	stream, err := d.grpcClient.UploadFile(ctx)
	if err != nil {
		return err
	}

	chunk := make([]byte, 4*(1<<10))
	for {
		n, err := file.Read(chunk)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if n < len(chunk) {
			chunk = chunk[:n]
		}
		err = stream.Send(&pb.UploadFileRequest{Data: chunk})
		if err != nil {
			return err
		}
	}
	_, err = stream.CloseAndRecv()
	return err
}

// DownloadFile TODO Show process
func (d *defaultClient) DownloadFile(filePath string) error {
	var err error
	stream, err := d.grpcClient.DownloadFile(context.Background(),
		&pb.DownloadFileRequest{Filepath: filePath})
	if err != nil {
		return err
	}

	var size int64
	md, err := stream.Header()
	if err != nil {
		return err
	}
	size, err = strconv.ParseInt(md.Get("size")[0], 0, 64)
	fmt.Println("file size: ", size)

	file, err := os.OpenFile(filepath.Base(filePath), os.O_CREATE | os.O_WRONLY | os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	//var pushCh = make(chan bool)
	//var processCh = showProcess(0, part, pushCh)
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			_ = os.Remove(file.Name())
			break
		}
		// TODO Test io.EOF
		_, err = file.Write(resp.Data)
		if err != nil {
			_ = os.Remove(file.Name())
			break
		}
		//pushCh<-true
	}
	if err != nil {
		//pushCh <- false
	}
	//<-processCh
	return err
}

// TODO show the process of file upload or download
func showProcess(start int, end int, push <-chan bool) <-chan struct{} {
	var processCh = make(<-chan struct{})
	go func(int, int, <-chan struct{}) {
		for v := range push {
			if !v {
				fmt.Printf("[]%5s", "Abort")
			}else {
				fmt.Printf("[]%5s", "Processing")
			}
		}
	}(start, end, processCh)
	return processCh
}

