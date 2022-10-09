package main

import (
	"bytes"
	"errors"
	"flag"
	"os"

	httpProxy "github.com/luoruofeng/dockermanagersingle/proxy/http"
	"github.com/luoruofeng/dockermanagersingle/types"

	"gopkg.in/yaml.v3"
)

func main() {
	filePath, proxyPort, proxyHost := ReadConfigFlag()

	_, err := ReadConfig(filePath)
	if err != nil {
		panic(err)
	}

	if *proxyPort != 0 {
		types.GConfig.ProxyPort = *proxyPort
	}

	if *proxyHost != "" {
		types.GConfig.ProxyHost = *proxyHost
	}

	httpProxy.Start(types.GConfig.ProxyHost, types.GConfig.ProxyPort)

}

func ReadConfigFlag() (configFile string, proxyPort *int, proxyHost *string) {
	configFile = *flag.String("config", "./config.yaml", "The configuration yaml file")
	proxyPort = flag.Int("proxy_port", 0, "The proxy url's port")
	proxyHost = flag.String("proxy_host", "", "The proxy listen url's host")
	flag.Parse()
	return
}

func ReadConfig(filePath string) (*types.Config, error) {
	result := &types.Config{}

	b, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("config file is not exist")
		} else {
			return nil, err
		}
	}

	r := bytes.NewReader(b)

	d := yaml.NewDecoder(r)
	err = d.Decode(result)
	if err != nil {
		return nil, err
	}
	types.GConfig = result
	return result, nil
}
