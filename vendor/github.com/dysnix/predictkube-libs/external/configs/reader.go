package configs

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func ReadConfigFile(file string, conf interface{}) (err error) {
	var yamlFile []byte
	_, err = os.Stat(file)
	if os.IsNotExist(err) && err != nil {
		file, err = filepath.EvalSymlinks(file)
		if err != nil {
			return err
		}
		yamlFile, err = ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(yamlFile, conf)
		return err
	}

	yamlFile, err = ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlFile, conf)
	return err
}
