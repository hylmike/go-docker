package utils

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type Manifest []struct {
	Config   string
	RepoTags []string
	Layers   []string
}

var Commands = []string{"run", "inner-mode", "setup-netns", "setup-veth", "ps", "exec", "images", "clean"}

const dockerHomePath = "/var/lib/go-docker"
const dockerTempPath = dockerHomePath + "/tmp"
const dockerImagesPath = dockerHomePath + "/images"
const dockerContainersPath = "/var/run/go-docker/containers"
const dockerNetNsPath = "/var/run/go-docker/net-ns"

const File_OtherReadExecute = 0755
const File_OtherNoPermit = 0700
const File_OtherReadOnly = 0644
const File_AllPermit = 0777

func GetDockerHomePath() string {
	return dockerHomePath
}

func GetDockerTempPath() string {
	return dockerTempPath
}

func GetDockerImagePath() string {
	return dockerImagesPath
}

func GetDockerContainerPath() string {
	return dockerContainersPath
}

func GetDockerNetNsPath() string {
	return dockerNetNsPath
}

func ShowGuide() {
	fmt.Println("Welcome to Go-Docker!")
	fmt.Println("Supported commands:")
	fmt.Println("go-docker run [--mem] [--swap] [--pids] [--cpus] <image> <command>")
	fmt.Println("go-docker clean --containerId")
	fmt.Println("go-docker exec <container-id> <command>")
	fmt.Println("go-docker images")
	fmt.Println("go-docker rmi <image-id>")
	fmt.Println("go-docker ps")
}

func ValidCommand(command string) bool {
	for _, element := range Commands {
		if element == command {
			return true
		}
	}
	return false
}

func CreateDirIfNotExist(dirs []string) error {
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err = os.MkdirAll(dir, 0755); err != nil {
				log.Printf("Failed to create directory: %v\n", err)
				return err
			}
		}
	}

	return nil
}

func InitDockerDirs() error {
	dirs := []string{dockerHomePath, dockerImagesPath, dockerNetNsPath, dockerContainersPath, dockerTempPath}
	return CreateDirIfNotExist(dirs)
}

func UnCompress(source, target string) error {
	hardLinks := make(map[string]string)
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	fileExt := filepath.Ext(source)
	var tarReader *tar.Reader
	if fileExt == ".gz" {
		//If compress file is .tar.gz file
		unzipStream, err := gzip.NewReader(reader)
		if err != nil {
			log.Fatalf("Failed to extract gz file %s: %v\n", source, err)
		}
		tarReader = tar.NewReader(unzipStream)
	} else if fileExt == ".tar" {
		//If compress file is .tar file
		tarReader = tar.NewReader(reader)
	} else {
		//If not these 2 types, then log error and exit
		log.Fatalf("Invalid compress file type %s\n", fileExt)
	}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		path := filepath.Join(target, header.Name)
		info := header.FileInfo()

		switch header.Typeflag {
		case tar.TypeDir:
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue

		case tar.TypeLink:
			/* Store details of hard links, which we process finally */
			linkPath := filepath.Join(target, header.Linkname)
			linkPath2 := filepath.Join(target, header.Name)
			hardLinks[linkPath2] = linkPath
			continue

		case tar.TypeSymlink:
			linkPath := filepath.Join(target, header.Name)
			if err := os.Symlink(header.Linkname, linkPath); err != nil {
				if os.IsExist(err) {
					continue
				}
				return err
			}
			continue

		case tar.TypeReg:
			/* Ensure any missing directories are created */
			if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
				os.MkdirAll(filepath.Dir(path), 0755)
			}
			file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
			if os.IsExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				return err
			}

		default:
			log.Printf("Warning: File type %d unhandled by untar function!\n", header.Typeflag)
		}
	}

	/* To create hard links the targets must exist, so we do this finally */
	for k, v := range hardLinks {
		if err := os.Link(v, k); err != nil {
			return err
		}
	}
	return nil
}

func ParseManifest(mfPath string, mf *Manifest) error {
	data, err := os.ReadFile(mfPath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, mf); err != nil {
		return err
	}

	return nil
}

func CopyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return nil
}
