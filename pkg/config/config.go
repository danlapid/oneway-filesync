package config

import (
	"github.com/BurntSushi/toml"
)

type Config struct {
	ReceiverIP       string
	ReceiverPort     int
	BandwidthLimit   int
	ChunkSize        int
	ChunkFecRequired int
	ChunkFecTotal    int
	OutDir           string
	WatchDir         string
}

func GetConfig(file string) (Config, error) {
	conf := Config{}
	_, err := toml.DecodeFile(file, &conf)
	return conf, err
}
