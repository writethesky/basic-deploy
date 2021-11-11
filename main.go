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

func deploy() {
	github.SetToken(internal.Config.Github.Token)
	for _, deployEntity := range internal.Config.Deploys {
		err := deployRepo(deployEntity)
		if nil != err {
			panic(err)
		}
	}
}

func deployRepo(deployEntity internal.DeployEntity) (err error) {
	log.Println("deploying")

	artifacts, err := getArtifacts(deployEntity.Owner, deployEntity.Repo)
	if nil != err || len(artifacts) == 0 {
		return
	}

	latestArtifact := artifacts[0]
	err = removeOldArtifacts(deployEntity.Owner, deployEntity.Repo, artifacts)
	if nil != err {
		return
	}

	if latestArtifact.ID == getLatestDeployID(deployEntity) {
		return
	}

	log.Printf("ID: %d\nName: %s\nDownload URL: %s\n", latestArtifact.ID, latestArtifact.Name, latestArtifact.ArchiveDownloadURL)
	fileBytes, err := github.DownloadArtifact(deployEntity.Owner, deployEntity.Repo, latestArtifact.ID)
	if nil != err {
		return
	}

	fileMap := unzip(fileBytes)
	if !execFileIsExist(fileMap, deployEntity.ExecFile) {
		return errors.New("exec file not found")
	}
	for key, value := range fileMap {
		err = saveFile(deployEntity.SavePath, key, 0777, value)
		if nil == err {
			return
		}
	}

	err = reload(deployEntity.ServiceName)
	if nil != err {
		return
	}

	saveLatestDeployID(deployEntity, latestArtifact.ID)
	log.Println("deployed")
	return
}

var deployedMap map[string]int

func init() {
	deployedMap = make(map[string]int)
}

func getDeployUID(deployEntity internal.DeployEntity) string {
	return fmt.Sprintf("%s/%s", deployEntity.Owner, deployEntity.Repo)
}

func saveLatestDeployID(deployEntity internal.DeployEntity, artifactID int) {
	deployedMap[getDeployUID(deployEntity)] = artifactID
}

func getLatestDeployID(deployEntity internal.DeployEntity) int {
	return deployedMap[getDeployUID(deployEntity)]
}

func removeOldArtifacts(owner, repo string, artifacts []github.Artifact) (err error) {
	if len(artifacts) <= 1 {
		return
	}
	oldArtifacts := artifacts[1:]
	return deleteArtifacts(owner, repo, oldArtifacts)
}

func execFileIsExist(fileMap map[string][]byte, execFile string) bool {
	for fileName := range fileMap {
		if fileName == execFile {
			return true
		}
	}
	return false
}

func reload(serviceName string) (err error) {
	cmd := exec.Command("systemctl", "restart", serviceName)
	err = cmd.Start()
	if nil != err {
		return
	}
	return cmd.Wait()
}

func getArtifacts(owner, repo string) (list []github.Artifact, err error) {
	artifactResponse, err := github.GetArtifacts(owner, repo)
	if nil != err {
		return
	}

	if 0 == artifactResponse.TotalCount {
		return nil, errors.New("artifacts not found")
	}
	return artifactResponse.Artifacts, nil
}

func deleteArtifacts(owner, repo string, artifacts []github.Artifact) (err error) {
	log.Println("There are currently multiple artifacts, the excess artifacts will be removed and only one artifact will be retained")
	for _, artifact := range artifacts {
		err = github.DeleteArtifact(owner, repo, artifact.ID)
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

func saveFile(savePath, fileName string, perm os.FileMode, fileBytes []byte) (err error) {
	err = os.Mkdir(savePath, 0755)
	if nil != err && !os.IsExist(err) {
		return
	}

	fileFullName := path.Join(savePath, fileName)
	err = os.Remove(fileFullName)
	if nil != err && !os.IsNotExist(err) {
		return
	}

	file, err := os.OpenFile(fileFullName, os.O_CREATE|os.O_WRONLY, perm)
	if nil != err {
		return
	}
	defer file.Close()
	_, err = file.Write(fileBytes)
	return
}
