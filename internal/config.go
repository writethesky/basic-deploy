package internal

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type ConfigEntity struct {
	Deploys []DeployEntity `yaml:"deploy"`
	Github  GithubEntity   `yaml:"github"`
}

type DeployEntity struct {
	SavePath    string `yaml:"save_path"`
	ExecFile    string `yaml:"exec_file"`
	ServiceName string `yaml:"service_name"`
	Owner       string `yaml:"owner"`
	Repo        string `yaml:"repo"`
}

type GithubEntity struct {
	Token string `yaml:"token"`
}

var Config *ConfigEntity

const defaultConfigFileName = "config/config.yaml"

func init() {

	flagSet := flag.NewFlagSet("server commend", flag.ContinueOnError)
	bufferBytes := make([]byte, 0)
	flagSet.SetOutput(bytes.NewBuffer(bufferBytes))
	configFileName := flagSet.String("config", "", "Configuration file location")
	err := flagSet.Parse(os.Args[1:])
	if nil != err && !strings.HasPrefix(err.Error(), "flag provided but not defined:") {
		panic(nil)
	}

	configFileNames := make([]string, 0)
	if "" != *configFileName {
		log.Printf("Use the user-defined configuration file %s", *configFileName)
		configFileNames = append(configFileNames, *configFileName)
	} else {
		defaultFileNames := getDefaultFileNames()
		log.Printf("Use default configuration file %s", defaultFileNames)
		configFileNames = append(configFileNames, defaultFileNames...)
	}

	var configFile *os.File
	for _, fileName := range configFileNames {
		log.Printf("try load config file %s", fileName)
		configFile, err = getConfigFile(fileName)
		if nil != err {
			log.Println(err)
			continue
		}
	}
	if nil == configFile {
		flagSet.SetOutput(nil)
		flagSet.Usage()
		os.Exit(0)
	}

	fileBytes, err := ioutil.ReadAll(configFile)
	if nil != err {
		panic(err)
	}
	Config = new(ConfigEntity)
	err = yaml.Unmarshal(fileBytes, Config)
	if nil != err {
		panic(err)
	}

}

func getConfigFile(configFileName string) (*os.File, error) {
	file, err := os.Open(configFileName)
	if nil != err {
		err = fmt.Errorf("failed to load config file with error message : '%s'", err)
	}
	return file, err
}

func getDefaultFileNames() (defaultFileNames []string) {
	defaultFileNames = make([]string, 0)

	ex, _ := os.Executable()
	// Executable DIR
	defaultFileNames = append(defaultFileNames, filepath.Join(filepath.Dir(ex), defaultConfigFileName))
	// Relative
	_, b, _, _ := runtime.Caller(0)
	defaultFileNames = append(defaultFileNames, path.Join(path.Dir(b), "../", defaultConfigFileName))
	return
}
