package pkg

import (
	"archive/tar"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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
		header.Name = path
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

// UnTarAndRemove is called to unzip the tar file and remove tar file.
func UnTarAndRemove(src string, dst string) error {
	if err := UnTar(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}
