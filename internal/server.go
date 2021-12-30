package internal

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"got/pkg"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const dirType = "dir"
const fileType = "file"

func CreateServer(port int) (GotServer, error) {
	server := &defaultServer{
		port: port,
	}
	return server, server.init()
}

type GotServer interface {
	init() error
	Run() error
	GotServiceServer
}

type defaultServer struct {
	port       int
	grpcServer *grpc.Server
}

func (d *defaultServer) init() error {
	d.grpcServer = grpc.NewServer()
	RegisterGotServiceServer(d.grpcServer, &defaultServer{})
	return nil
}

func (d *defaultServer) Run() error {
	lisn, err := net.Listen("tcp", fmt.Sprintf(":%d", d.port))
	if err != nil {
		return err
	}
	return d.grpcServer.Serve(lisn)
}

func (d *defaultServer) ListFile(ctx context.Context, req *ListFilesRequest) (*ListFilesResponse, error) {
	p, _ := peer.FromContext(ctx)
	log.Printf("%-12s called from: %s\n", "ListFile", p.Addr.String())

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	filesInfo, err := ioutil.ReadDir(wd)
	if err != nil {
		return nil, err
	}
	var info = fmt.Sprintf("%s:\n", wd)
	for i := range filesInfo {
		info += fmt.Sprintf("%-12s%-20s%-10d\n",
			filesInfo[i].Mode(),
			filesInfo[i].Name(),
			filesInfo[i].Size(),
		)
	}
	info += fmt.Sprint("count: ", len(filesInfo))
	return &ListFilesResponse{Info: info}, nil
}

func (d *defaultServer) ChangeDir(ctx context.Context, req *ChangeDirRequest) (*ChangeDirResponse, error) {
	p, _ := peer.FromContext(ctx)
	log.Printf("%-12s called from: %s\n", "ChangeDir", p.Addr.String())

	err := os.Chdir(req.DstDir)
	if err != nil {
		return nil, err
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	filesInfo, err := ioutil.ReadDir(wd)
	if err != nil {
		return nil, err
	}
	var info = fmt.Sprintf("%s:\n", wd)
	for i := range filesInfo {
		info += fmt.Sprintf("%-12s%-20s%-10d\n",
			filesInfo[i].Mode(),
			filesInfo[i].Name(),
			filesInfo[i].Size(),
		)
	}
	info += fmt.Sprint("count: ", len(filesInfo))
	return &ChangeDirResponse{Info: info}, nil
}

func (d *defaultServer) UploadFile(stream GotService_UploadFileServer) error {
	p, _ := peer.FromContext(stream.Context())
	log.Printf("%-12s called from: %s\n", "UploadFile", p.Addr.String())

	var fileName string
	var uploadType string
	md, ok := metadata.FromIncomingContext(stream.Context())
	if ok {
		if n := md.Get("name"); n != nil {
			fileName = n[0]
		}
		if t := md.Get("type"); t != nil {
			uploadType = t[0]
		}
	} else {
		return errors.New("file name not defined")
	}

	saveFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer func() {
		_ = saveFile.Close()
		if uploadType == dirType {
			err = os.Remove(fileName)
		}
	}()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			_ = os.Remove(fileName)
			return err
		}

		_, err = saveFile.Write(resp.Data)
		if err != nil {
			_ = os.Remove(fileName)
			return err
		}
	}

	if uploadType == dirType {
		if err = pkg.UnTar(fileName, "."); err != nil {
			return err
		}
	}
	return stream.SendAndClose(&UploadFileResponse{Ok: ok})
}

func (d *defaultServer) DownloadFile(req *DownloadFileRequest, stream GotService_DownloadFileServer) error {
	p, _ := peer.FromContext(stream.Context())
	log.Printf("%-12s called from: %s\n", "DownloadFile", p.Addr.String())

	var err error
	var filePath = req.Filepath
	var mdMap = make(map[string]string)
	defer func() {
		if err != nil {
			_ = stream.SendHeader(metadata.Pairs("err", err.Error()))
		}
	}()

	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		dirTarPath := filepath.Join(filepath.Dir(filePath),
			fmt.Sprintf("%s%d.tar", filepath.Base(filePath), time.Now().Unix()))
		err := pkg.Tar(filePath, dirTarPath)
		if err != nil {
			return err
		}
		mdMap["type"] = dirType

		info, err = os.Stat(dirTarPath)
		filePath = dirTarPath

		defer func() {
			_ = os.Remove(filePath)
		}()
	} else {
		mdMap["type"] = fileType
	}

	file, err := os.OpenFile(filePath, os.O_RDONLY, 0664)
	if err != nil {
		return err
	}
	defer file.Close()

	mdMap["size"] = strconv.FormatInt(info.Size(), 10)
	md := metadata.New(mdMap)
	err = stream.SetHeader(md)
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
		err = stream.Send(&DownloadFileResponse{
			Data: chunk,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
