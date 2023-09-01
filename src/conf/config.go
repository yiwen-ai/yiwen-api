package conf

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/util"
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
	readConfig(p, "../../config/default.toml")
	if err := p.Validate(); err != nil {
		panic(err)
	}
	p.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	p.GlobalSignal = gear.ContextWithSignal(context.Background())

	var cancel context.CancelFunc
	p.GlobalShutdown, cancel = context.WithCancel(context.Background())
	go func() {
		<-p.GlobalSignal.Done()
		time.AfterFunc(time.Duration(p.Server.GracefulShutdown)*time.Second, cancel)
	}()
}

type Logger struct {
	Level string `json:"level" toml:"level"`
}

type Server struct {
	Addr             string `json:"addr" toml:"addr"`
	GracefulShutdown uint   `json:"graceful_shutdown" toml:"graceful_shutdown"`
}

type Redis struct {
	Prefix string `json:"prefix" toml:"prefix"`
	Node   string `json:"node" toml:"node"`
}

type Base struct {
	Userbase   string `json:"userbase" toml:"userbase"`
	Writing    string `json:"writing" toml:"writing"`
	Jarvis     string `json:"jarvis" toml:"jarvis"`
	Logbase    string `json:"logbase" toml:"logbase"`
	Taskbase   string `json:"taskbase" toml:"taskbase"`
	Webscraper string `json:"webscraper" toml:"webscraper"`
	Walletbase string `json:"walletbase" toml:"walletbase"`
}

type OSS struct {
	Bucket          string `json:"bucket" toml:"bucket"`
	Endpoint        string `json:"endpoint" toml:"endpoint"`
	AccessKeyId     string `json:"access_key_id" toml:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret" toml:"access_key_secret"`
	Prefix          string `json:"prefix" toml:"prefix"`
	UrlBase         string `json:"url_base" toml:"url_base"`
}

type Recommendation struct {
	GID util.ID `json:"gid" toml:"gid"`
	CID util.ID `json:"cid" toml:"cid"`
}

// ConfigTpl ...
type ConfigTpl struct {
	Rand            *rand.Rand
	GlobalSignal    context.Context
	GlobalShutdown  context.Context
	Env             string           `json:"env" toml:"env"`
	Logger          Logger           `json:"log" toml:"log"`
	Server          Server           `json:"server" toml:"server"`
	Redis           Redis            `json:"redis" toml:"redis"`
	Base            Base             `json:"base" toml:"base"`
	OSS             OSS              `json:"oss" toml:"oss"`
	Recommendations []Recommendation `json:"recommendations" toml:"recommendations"`

	globalJobs int64 // global async jobs counter for graceful shutdown
}

func (c *ConfigTpl) Validate() error {
	return nil
}

func (c *ConfigTpl) ObtainJob() {
	atomic.AddInt64(&c.globalJobs, 1)
}

func (c *ConfigTpl) ReleaseJob() {
	atomic.AddInt64(&c.globalJobs, -1)
}

func (c *ConfigTpl) JobsIdle() bool {
	return atomic.LoadInt64(&c.globalJobs) <= 0
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
