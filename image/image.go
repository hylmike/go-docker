package image

import (
	"encoding/json"
	"fmt"
	"go-docker/utils"
	"log"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type imageEntries map[string]string

type imagesDB map[string]imageEntries

type imageConfig struct {
	Env []string `json:"Env"`
	Cmd []string `json:"Cmd"`
}

type imageInfo struct {
	Config imageConfig `json:"config"`
}

func GetImageNameAndTag(src string) (string, string) {
	imageInfo := strings.Split(src, ":")
	var imgName, tag string
	if len(imageInfo) > 1 {
		imgName, tag = imageInfo[0], imageInfo[1]
	} else {
		imgName = imageInfo[0]
		tag = "latest"
	}

	return imgName, tag
}

func parseImageMetadata(iDB *imagesDB) {
	imagesDBPath := utils.GetDockerImagePath() + "/" + "images.json"
	if _, err := os.Stat(imagesDBPath); os.IsNotExist(err) {
		//Create empty DB if path not exist
		os.WriteFile(imagesDBPath, []byte("{}"), 0644)
	}

	data, err := os.ReadFile(imagesDBPath)
	if err != nil {
		log.Fatalf("Failed to read image DB: %v\n", err)
	}

	if err := json.Unmarshal(data, iDB); err != nil {
		log.Fatalf("Failed to parse image DB: %v\n", err)
	}
}

func getImageByTag(imageName string, tag string) (bool, string) {
	idb := imagesDB{}
	parseImageMetadata(&idb)
	for k1, v1 := range idb {
		if k1 == imageName {
			for k2, v2 := range v1 {
				if k2 == tag {
					return true, v2
				}
			}
		}
	}

	return false, ""
}

func imageExistByHash(imageShaHex string) (string, string) {
	idb := imagesDB{}
	parseImageMetadata(&idb)

	for imgName, avlImages := range idb {
		for imgTag, imgHash := range avlImages {
			if imgHash == imageShaHex {
				return imgName, imgTag
			}
		}
	}

	return "", ""
}

func marshalImageMetadata(idb imagesDB) {
	fileBytes, err := json.Marshal(idb)
	if err != nil {
		log.Fatalf("Failed to marshall images data: %v\n", err)
	}

	imagesDBPath := utils.GetDockerImagePath() + "/" + "images.json"
	if err := os.WriteFile(imagesDBPath, fileBytes, 0644); err != nil {
		log.Fatalf("Failed to save images DB: %v\n", err)
	}
}

func storeImageMetadata(imgName string, tag string, imageShaHex string) {
	idb := imagesDB{}
	imgEntry := imageEntries{}

	parseImageMetadata(&idb)
	if idb[imgName] != nil {
		imgEntry = idb[imgName]
	}
	imgEntry[tag] = imageShaHex
	idb[imgName] = imgEntry

	marshalImageMetadata(idb)
}

func downloadImage(img v1.Image, imageShaHex string, src string) {
	path := utils.GetDockerTempPath() + "/" + imageShaHex
	os.Mkdir(path, 0755)
	path += "/package.tar"

	//Save image as tar file
	if err := crane.SaveLegacy(img, src, path); err != nil {
		log.Fatalf("Failed to download and save image as tar file %s: %v", path, err)
	}

	log.Printf("Success downloaded %s\n", src)
}

func untarFile(imageShaHex string) {
	pathDir := utils.GetDockerTempPath() + "/" + imageShaHex
	pathTar := pathDir + "/package.tar"
	if err := utils.UnCompress(pathTar, pathDir); err != nil {
		log.Fatalf("Failed to untarging file: %v\n", err)
	}
}

func GetBasePathForImage(imgShaHex string) string {
	return utils.GetDockerImagePath() + "/" + imgShaHex
}

func GetManifestPathForImage(imgShaHex string) string {
	return GetBasePathForImage(imgShaHex) + "/manifest.json"
}

func GetConfigPathForImage(imgShaHex string) string {
	return GetBasePathForImage(imgShaHex) + "/" + imgShaHex + ".json"
}

func processLayerTarballs(imageShaHex string, fullImageHex string) {
	tempPathDir := utils.GetDockerTempPath() + "/" + imageShaHex
	pathManifest := tempPathDir + "/manifest.json"
	pathConfig := tempPathDir + "/" + fullImageHex + ".json"

	mf := utils.Manifest{}
	utils.ParseManifest(pathManifest, &mf)
	if len(mf) == 0 || len(mf[0].Layers) == 0 {
		log.Fatal("Can't find any layers\n")
	}
	if len(mf) > 1 {
		log.Fatal("Can't handle more than one manifest")
	}

	imageDir := utils.GetDockerImagePath() + "/" + imageShaHex
	_ = os.Mkdir(imageDir, 0755)

	//Untar all layer files, which is basis of container root fs
	for _, layer := range mf[0].Layers {
		imgLayerDir := imageDir + "/" + layer[:12] + "/fs"
		log.Printf("Umcompressing layer to %s\n", imgLayerDir)
		_ = os.MkdirAll(imgLayerDir, 0755)
		srcLayer := tempPathDir + "/" + layer
		if err := utils.UnCompress(srcLayer, imgLayerDir); err != nil {
			log.Fatalf("Failed to untar layer file %s: %v\n", srcLayer, err)
		}
	}

	//Copy manifest file for reference later
	utils.CopyFile(pathManifest, GetManifestPathForImage(imageShaHex))
	utils.CopyFile(pathConfig, GetConfigPathForImage(imageShaHex))
}

func deleteTempImageFiles(imgShaHex string) {
	tempPath := utils.GetDockerTempPath() + "/" + imgShaHex
	if err := os.RemoveAll(tempPath); err != nil {
		log.Fatalf("Failed to remove temporary image files %s: %v\n", tempPath, err)
	}
}

func DownloadImageIfRequired(src string) string {
	imgName, tag := GetImageNameAndTag(src)
	if requireDownload, imageShaHex := getImageByTag(imgName, tag); !requireDownload {
		log.Printf("Download metadata for %s:%s, please wait...", imgName, tag)
		img, err := crane.Pull(strings.Join([]string{imgName, tag}, ":"))
		if err != nil {
			log.Fatalf("Failed to pull image %s:%s from server: %v\n", imgName, tag, err)
		}

		mf, _ := img.Manifest()
		imageShaHex = mf.Config.Digest.Hex[:12]
		log.Printf("image hash: %v\n", imageShaHex)
		log.Println("Checking if image exists under with other names")

		altImgName, altTag := imageExistByHash(imageShaHex)
		if len(altImgName) > 0 && len(altTag) > 0 {
			log.Printf("The image you requested - %s:%s is same as %s, %s\n", imgName, tag, altImgName, altTag)
			storeImageMetadata(imgName, tag, imageShaHex)
			return imageShaHex
		} else {
			log.Println("Image don't exist. Downloading...")
			downloadImage(img, imageShaHex, src)
			untarFile(imageShaHex)

			processLayerTarballs(imageShaHex, mf.Config.Digest.Hex)
			storeImageMetadata(imgName, tag, imageShaHex)
			deleteTempImageFiles(imageShaHex)
			return imageShaHex
		}
	} else {
		log.Println("Image already exists, skip download")
		return imageShaHex
	}
}

func ParseContainerConfig(imgShaHex string) imageInfo {
	imagesConfigPath := GetConfigPathForImage(imgShaHex)

	data, err := os.ReadFile(imagesConfigPath)
	if err != nil {
		log.Fatalf("Failed to read image %s config file: %v\n", imgShaHex, err)
	}

	imgInfo := imageInfo{}
	if err := json.Unmarshal(data, &imgInfo); err != nil {
		log.Fatalf("Failed to parse image info data")
	}

	return imgInfo
}

func PrintImages() {
	idb := imagesDB{}
	parseImageMetadata(&idb)

	fmt.Printf("Image\tTag\tID\n")
	for image, details := range idb {
		fmt.Println(image)
		for tag, hash := range details {
			fmt.Printf("\t%16s\t%s\n", tag, hash)
		}
	}
}
