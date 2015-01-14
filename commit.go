package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/docker/docker/image"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/tarsum"
	"github.com/docker/docker/utils"
)

func CommitChanges(rootfs, message string) error {
	if !IsGitRepo(rootfs) {
		return fmt.Errorf("%v not a git repository", rootfs)
	}
	gitRepo, _ := NewGitRepo(rootfs)

	layerData, err := gitRepo.ExportUncommitedChangeSet()
	if err != nil {
		return err
	}

	//Load image data
	image, err := image.LoadImage(gitRepo.Path) //reading json file in rootfs
	if err != nil {
		return err
	}

	layerTarSum, err := tarsum.NewTarSum(layerData, true, tarsum.VersionDev)
	if err != nil {
		return err
	}

	//fill new infos
	image.Checksum = layerTarSum.Sum(nil)
	image.Parent = image.ID
	image.ID = utils.GenerateRandomID()
	image.Created = time.Now()
	image.Comment = message

	layer, err := archive.NewTempArchive(layerData, "")
	if err != nil {
		return err
	}
	image.Size = layer.Size
	os.RemoveAll(layer.Name())

	if err := image.SaveSize(rootfs); err != nil {
		return err
	}

	jsonRaw, err := json.Marshal(image)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(rootfs, "json"), jsonRaw, 0600)
	if err != nil {
		return err
	}

	//commit the changes in a new branch
	brNumber, _ := gitRepo.CountBranches()
	br := "layer" + strconv.Itoa(brNumber) + "_" + image.ID
	if _, err = gitRepo.CheckoutB(br); err != nil {
		return err
	}
	if _, err := gitRepo.AddAllAndCommit(message); err != nil {
		return err
	}

	fmt.Printf("Changes commited in %v\n", br)
	fmt.Printf("Image ID: %v\nParent: %v\nChecksum: %v\nLayer size: %v\n", image.ID, image.Parent, image.Checksum, image.Size)

	return nil
}
