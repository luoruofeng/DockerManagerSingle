package types

var GConfig *Config

type Config struct {
	ProxyPort int    `yaml:"proxy_port"`
	ProxyHost string `yaml:"proxy_host"`
}
