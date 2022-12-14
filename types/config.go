package types

var GConfig *Config

type Config struct {
	ProxyPort       int  `yaml:"proxy_port"`
	ApiPort         int  `yaml:"api_port"`
	GrpcPort        int  `yaml:"grpc_port"`
	ApiWriteTimeout int  `yaml:"api_write_timeout"`
	ApiReadTimeout  int  `yaml:"api_read_timeout"`
	ApiIdleTimeout  int  `yaml:"api_idle_timeout"`
	APIEnable       bool `yaml:"api_enable"`
	GRPCEnable      bool `yaml:"grpc_enable"`
}
