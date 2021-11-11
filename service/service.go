package service

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
)

func Run(serviceName, execFile string) (err error) {
	if !isInstall(serviceName) {
		err = install(serviceName, execFile)
		if nil != err {
			return
		}
		fmt.Printf("Modify the configuration file(%s) and run the following command\n", getConfigFile(serviceName))
		fmt.Printf("systemctl start %s\n", serviceName)
		return
	}

	if isStart(serviceName) {
		err = restart(serviceName)
		return
	}
	err = start(serviceName)
	return
}

func isInstall(serviceName string) bool {
	_, err := status(serviceName)
	return nil == err
}

const (
	systemdPath            = "/lib/systemd/system/"
	configPath             = "/etc/"
	systemdServiceTemplate = `
[Unit]
Description=%s Server
After=network.target
After=syslog.target

[Install]
WantedBy=multi-user.target

[Service]
User=root
Group=root

Type=simple


# Execute pre and post scripts as root
PermissionsStartOnly=true

# Start main service
ExecStart=%s -config %s

# Sets open_files_limit
LimitNOFILE = 10000

Restart=on-failure

RestartPreventExitStatus=1

PrivateTmp=false
`
)

func generateSystemdServiceContent(serviceName, execFile, configFile string) string {
	return fmt.Sprintf(systemdServiceTemplate, serviceName, execFile, configFile)
}

func getConfigFile(serviceName string) string {
	return path.Join(configPath, fmt.Sprintf("%s.yaml", serviceName))
}

func install(serviceName, execFile string) (err error) {
	configFile := getConfigFile(serviceName)
	err = saveFile(configFile, "")
	if nil != err {
		return
	}
	err = saveFile(path.Join(systemdPath, serviceName), generateSystemdServiceContent(serviceName, execFile, configFile))
	if nil != err {
		return
	}
	_, err = runSystemctlCMD(serviceName, "enable")
	return
}

func saveFile(filePath, fileContent string) (err error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if nil != err {
		return
	}
	_, err = file.WriteString(fileContent)
	return
}

func isStart(serviceName string) bool {
	serviceStatus, err := status(serviceName)
	if nil != err {
		log.Println(err)
		return false
	}
	if serviceStatus == StatusActive {
		return true
	}
	if serviceStatus == StatusInactive {
		return false
	}
	log.Println("unknown status: " + serviceStatus)
	return false
}

func start(serviceName string) (err error) {
	_, err = runSystemctlCMD(serviceName, "start")
	return
}

func restart(serviceName string) (err error) {
	_, err = runSystemctlCMD(serviceName, "restart")
	return
}

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
)

var statusReg = regexp.MustCompile(`Active: (\w+) `)

func status(serviceName string) (serviceStatus Status, err error) {
	output, err := runSystemctlCMD(serviceName, "status")
	if nil != err {
		return
	}

	matches := statusReg.FindStringSubmatch(output)
	if nil == matches {
		log.Println(output)
		err = errors.New("not found service status")
		return
	}
	serviceStatus = Status(matches[1])
	return
}

func runSystemctlCMD(serviceName, subCMD string) (outputString string, err error) {
	cmd := exec.Command("systemctl", subCMD, serviceName)
	output, err := cmd.Output()
	if nil != err {
		log.Printf("not found service %s\n", serviceName)
		return
	}
	return string(output), err
}
