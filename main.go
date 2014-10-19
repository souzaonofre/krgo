package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/registry"
)

const (
	IMAGE_NAME  string = "dockerfile/elasticsearch"
	ROOTFS_DEST string = "./rootfs"
)

func assertErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var imageName string
	var imageTag string

	if strings.Contains(IMAGE_NAME, ":") {
		imageName = strings.Split(IMAGE_NAME, ":")[0]
		imageTag = strings.Split(IMAGE_NAME, ":")[1]
	} else {
		imageName = IMAGE_NAME
		imageTag = "latest"
	}

	//resolving endpoint
	registryEndpoint, err := resolveEndpointForImage(imageName)
	assertErr(err)

	fmt.Println("Using:", registryEndpoint.URL, "API version:", registryEndpoint.Version)

	//opening a session
	//empty auth config (probably used only for private repository or private images I guess)
	authConfig := &registry.AuthConfig{}
	var metaHeaders map[string][]string

	session, err := registry.NewSession(authConfig, registry.HTTPRequestFactory(metaHeaders), registryEndpoint, true)
	assertErr(err)

	//Get back token and endpoint for the repository
	repoData, err := session.GetRepositoryData(imageName)
	assertErr(err)

	tokens := repoData.Tokens
	repoEndpoint := repoData.Endpoints[0]

	fmt.Println("Fetching", repoEndpoint, " with tokens", tokens)

	tagsList, err := session.GetRemoteTags(repoData.Endpoints, imageName, tokens)
	assertErr(err)
	imageId := tagsList[imageTag]
	fmt.Println(imageName, "with tag", imageTag, "has ID", imageId)

	//Download image history (get back all the layers)
	history, err := session.GetRemoteHistory(imageId, repoEndpoint, tokens)
	assertErr(err)

	fmt.Println("Image", imageName, "is made of", len(history), "layers:", history)

	os.MkdirAll(ROOTFS_DEST, 0777)

	for i := len(history) - 1; i >= 0; i-- {
		layerId := history[i]

		fmt.Println("Downloading layer", layerId, "...")
		layerData, err := downloadImageLayer(session, layerId, repoEndpoint, tokens)
		defer layerData.Close()
		assertErr(err)

		fmt.Println("Untaring layer", layerId)
		err = archive.Untar(layerData, ROOTFS_DEST, nil)
		assertErr(err)

		fmt.Println("done", layerId)
	}

	fmt.Println("All good, rootfs in", ROOTFS_DEST)
}

func resolveEndpointForImage(imageName string) (*registry.Endpoint, error) {
	hostname, _, err := registry.ResolveRepositoryName(imageName)
	if err != nil {
		return nil, err
	}
	return registry.NewEndpoint(hostname)
}

func downloadImageLayer(session *registry.Session, imageId, endpoint string, tokens []string) (io.ReadCloser, error) {
	//Get back image information
	_, imgSize, err := session.GetRemoteImageJSON(imageId, endpoint, tokens)
	if err != nil {
		return nil, err
	}
	return session.GetRemoteImageLayer(imageId, endpoint, tokens, int64(imgSize))
}