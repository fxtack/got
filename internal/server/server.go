package server

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"got/internal/pb"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
)

func Create(port int) (GotServer, error) {
	server := &defaultServer{
		port: port,
	}
	return server, server.init()
}

type GotServer interface {
	init() error
	Run() error
	pb.GotServiceServer
}

type defaultServer struct {
	port int
	grpcServer *grpc.Server
}

func (d *defaultServer) init() error {
	d.grpcServer = grpc.NewServer()
	pb.RegisterGotServiceServer(d.grpcServer, &defaultServer{})
	return nil
}

func (d *defaultServer) Run() error {
	lisn, err := net.Listen("tcp", fmt.Sprintf(":%d", d.port))
	if err != nil {
		return err
	}
	return d.grpcServer.Serve(lisn)
}

func (d *defaultServer) ListFile(ctx context.Context, req *pb.ListFilesRequest) (*pb.ListFilesResponse, error) {
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
	return &pb.ListFilesResponse{Info: info}, nil
}

func (d *defaultServer) ChangeDir(ctx context.Context, req *pb.ChangeDirRequest) (*pb.ChangeDirResponse, error) {
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
	return &pb.ChangeDirResponse{Info: info}, nil
}

func (d *defaultServer) UploadFile(stream pb.GotService_UploadFileServer) error {
	p, _ := peer.FromContext(stream.Context())
	log.Printf("%-12s called from: %s\n", "UploadFile", p.Addr.String())

	var fileName string
	md, ok := metadata.FromIncomingContext(stream.Context())
	if ok {
		fileName = md.Get("name")[0]
	}else {
		return errors.New("save file name not defined")
	}

	saveFile, err := os.OpenFile(fileName, os.O_CREATE | os.O_EXCL | os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer saveFile.Close()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.UploadFileResponse{Ok: ok})
		}else if err != nil {
			_ = os.Remove(fileName)
			return err
		}

		_, err = saveFile.Write(resp.Data)
		if err != nil {
			_ = os.Remove(fileName)
			return err
		}
	}
}

func (d *defaultServer) DownloadFile(req *pb.DownloadFileRequest, stream pb.GotService_DownloadFileServer) error {
	p, _ := peer.FromContext(stream.Context())
	log.Printf("%-12s called from: %s\n", "DownloadFile", p.Addr.String())

	var filePath = req.Filepath
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0664)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	md := metadata.New(map[string]string{"size": strconv.FormatInt(stat.Size(), 10)})
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
		err = stream.Send(&pb.DownloadFileResponse{
			Data: chunk,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

