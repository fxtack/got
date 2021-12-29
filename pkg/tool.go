package pkg

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Tar is called for zip up file or directory.
// `src` is source of file for tar zip, `dst` is the save path of tar file.
func Tar(src string, dst string) error {
	var err error
	tarFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	tarWriter := tar.NewWriter(tarFile)
	defer tarWriter.Close()

	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(path)
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if !info.IsDir() {
			file, err := os.OpenFile(path, os.O_RDONLY, 0644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}
		return err
	})
}

// UnTar is called for unzip tar file.
// `src` is tar file path, `dst` is target path for unzip.
func UnTar(src string, dst string) error {
	var err error

	tarFile, err := os.OpenFile(src, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	tarReader := tar.NewReader(tarFile)
	for header, err := tarReader.Next(); err != io.EOF; header, err = tarReader.Next() {
		if err != nil {
			return err
		}
		info := header.FileInfo()
		path := filepath.Join(dst, header.Name)
		if header.Typeflag == tar.TypeDir {
			_ = os.MkdirAll(path, info.Mode().Perm())
			_ = os.Chmod(path, info.Mode().Perm())
		} else {
			_ = os.MkdirAll(filepath.Dir(path), os.ModeDir)
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			if _, err = io.Copy(file, tarReader); err != nil {
				return err
			}
			_ = file.Close()
			_ = os.Chmod(path, info.Mode().Perm())
		}
	}
	return err
}

func ProcessBar(tag string, start int64, end int64, push <-chan int64, ctx context.Context) (<-chan struct{}, error) {
	if end < start || push == nil {
		return nil, errors.New("invalid argument")
	}
	var processCh = make(chan struct{})
	go func(int64, int64, <-chan struct{}) {
		var barLen = 16
		var totalProgress = end - start
		var stepProgress = totalProgress / int64(barLen)
		var currentStepProgress = stepProgress

		for i := 0; i < barLen; {
			select {
			case progress := <-push:
				currentStepProgress -= progress
				for ; currentStepProgress <= 0; currentStepProgress += stepProgress {
					fmt.Printf("\r%-12s%-12s: [%s%s]", tag, "processing", strings.Repeat("█", i),
						strings.Repeat("-", barLen-i))
					i++
				}
			case <-ctx.Done():
				fmt.Printf("\r%-12s%-12s: [%s]\n", tag, "abort", strings.Repeat("█", i))
				processCh <- struct{}{}
				return
			}
		}
		fmt.Printf("\r%-12s%-12s: [%s]\n", tag, "finish", strings.Repeat("█", barLen))
		processCh <- struct{}{}
		close(processCh)
	}(start, end, processCh)
	return processCh, nil
}
