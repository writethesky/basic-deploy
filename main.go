package main

import (
	"archive/zip"
	"basic-deploy/github"
	"basic-deploy/internal"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"

	"github.com/robfig/cron/v3"
)

func main() {
	c := cron.New()
	_, _ = c.AddFunc("@every 1m", deploy)
	c.Start()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan
}

var lastDeployID int

func deploy() {
	log.Println("deploying")
	github.SetToken(internal.Config.Github.Token)

	artifacts, err := getArtifacts()
	if nil != err {
		panic(err)
	}

	latestArtifact := artifacts[0]
	if len(artifacts) > 1 {
		err = deleteArtifacts(artifacts[1:])
		if nil != err {
			panic(err)
		}
	}

	if latestArtifact.ID == lastDeployID {
		return
	}
	log.Printf("ID: %d\nName: %s\nDownload URL: %s\n", latestArtifact.ID, latestArtifact.Name, latestArtifact.ArchiveDownloadURL)
	fileBytes, err := github.DownloadArtifact(internal.Config.Github.Owner, internal.Config.Github.Repo, latestArtifact.ID)
	if nil != err {
		panic(err)
	}

	fileMap := unzip(fileBytes)
	if !execFileIsExist(fileMap) {
		panic("exec file not found")
	}
	for key, value := range fileMap {
		saveFile(key, 0777, value)
	}

	err = reload()
	if nil != err {
		panic(err)
	}

	lastDeployID = latestArtifact.ID
	log.Println("deployed")
}

func execFileIsExist(fileMap map[string][]byte) bool {
	for fileName := range fileMap {
		if fileName == internal.Config.Deploy.ExecFile {
			return true
		}
	}
	return false
}

func reload() (err error) {
	cmd := exec.Command("systemctl", "restart", internal.Config.Deploy.ServiceName)
	err = cmd.Start()
	if nil != err {
		return
	}
	return cmd.Wait()
}

func getArtifacts() (list []github.Artifact, err error) {
	artifactResponse, err := github.GetArtifacts(internal.Config.Github.Owner, internal.Config.Github.Repo)
	if nil != err {
		return
	}

	if 0 == artifactResponse.TotalCount {
		return nil, errors.New("artifacts not found")
	}
	return artifactResponse.Artifacts, nil
}

func deleteArtifacts(artifacts []github.Artifact) (err error) {
	log.Println("There are currently multiple artifacts, the excess artifacts will be removed and only one artifact will be retained")
	for _, artifact := range artifacts {
		err = github.DeleteArtifact(internal.Config.Github.Owner, internal.Config.Github.Repo, artifact.ID)
		if nil != err {
			return err
		}
	}
	return
}

func unzip(fileBytes []byte) (fileMap map[string][]byte) {
	fileMap = make(map[string][]byte)
	zipReader, err := zip.NewReader(bytes.NewReader(fileBytes), int64(len(fileBytes)))
	if nil != err {
		panic(err)
	}

	for _, file := range zipReader.File {
		fmt.Printf("Contents of %s:\n", file.Name)
		rc, err := file.Open()
		if err != nil {
			panic(err)
		}
		tmpFileBytes, err := ioutil.ReadAll(rc)
		if err != nil {
			panic(err)
		}
		fileMap[file.Name] = tmpFileBytes
		rc.Close()
	}
	return
}

func saveFile(fileName string, perm os.FileMode, fileBytes []byte) {
	fileErr := os.Mkdir(internal.Config.Deploy.SavePath, 0755)
	if nil != fileErr && !os.IsExist(fileErr) {
		panic(fileErr)
	}

	fileFullName := path.Join(internal.Config.Deploy.SavePath, fileName)
	err := os.Remove(fileFullName)
	if nil != err && !os.IsNotExist(err) {
		panic(fileErr)
	}

	file, err := os.OpenFile(fileFullName, os.O_CREATE|os.O_WRONLY, perm)
	if nil != err {
		panic(err)
	}
	defer file.Close()
	_, err = file.Write(fileBytes)
	if nil != err {
		panic(err)
	}
}
