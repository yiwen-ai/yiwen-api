package conf

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/teambition/gear"
)

// Config ...
var Config ConfigTpl

var AppName = "yiwen-api"
var AppVersion = "0.1.0"
var BuildTime = "unknown"
var GitSHA1 = "unknown"

var once sync.Once

func init() {
	p := &Config
	readConfig(p)
	if err := p.Validate(); err != nil {
		panic(err)
	}
	p.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	p.GlobalCtx = gear.ContextWithSignal(context.Background())
}

type Logger struct {
	Level string `json:"level" toml:"level"`
}

type Server struct {
	Addr             string `json:"addr" toml:"addr"`
	GracefulShutdown uint   `json:"graceful_shutdown" toml:"graceful_shutdown"`
}

type Base struct {
	Userbase   string `json:"userbase" toml:"userbase"`
	Writing    string `json:"writing" toml:"writing"`
	Jarvis     string `json:"jarvis" toml:"jarvis"`
	Webscraper string `json:"webscraper" toml:"webscraper"`
}

type OSS struct {
	Bucket          string `json:"bucket" toml:"bucket"`
	Endpoint        string `json:"endpoint" toml:"endpoint"`
	AccessKeyId     string `json:"access_key_id" toml:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret" toml:"access_key_secret"`
	Prefix          string `json:"prefix" toml:"prefix"`
	UrlBase         string `json:"url_base" toml:"url_base"`
}

// ConfigTpl ...
type ConfigTpl struct {
	Rand      *rand.Rand
	GlobalCtx context.Context
	Env       string `json:"env" toml:"env"`
	Logger    Logger `json:"log" toml:"log"`
	Server    Server `json:"server" toml:"server"`
	Base      Base   `json:"base" toml:"base"`
	OSS       OSS    `json:"oss" toml:"oss"`
}

func (c *ConfigTpl) Validate() error {
	return nil
}

func readConfig(v interface{}, path ...string) {
	once.Do(func() {
		filePath, err := getConfigFilePath(path...)
		if err != nil {
			panic(err)
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			panic(err)
		}

		_, err = toml.Decode(string(data), v)
		if err != nil {
			panic(err)
		}
	})
}

func getConfigFilePath(path ...string) (string, error) {
	// 优先使用的环境变量
	filePath := os.Getenv("CONFIG_FILE_PATH")

	// 或使用指定的路径
	if filePath == "" && len(path) > 0 {
		filePath = path[0]
	}

	if filePath == "" {
		return "", fmt.Errorf("config file not specified")
	}

	return filePath, nil
}
