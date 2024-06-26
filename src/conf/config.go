package conf

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/fxamacker/cbor/v2"
	"github.com/ldclabs/cose/key"
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

type Keys struct {
	Hmac   string `json:"hmac" toml:"hmac"`
	Aesgcm string `json:"aesgcm" toml:"aesgcm"`
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
	BaseUrl         string `json:"base_url" toml:"base_url"`
}

type Wechat struct {
	AppID  string `json:"appid" toml:"appid"`
	Secret string `json:"secret" toml:"secret"`
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
	Env             string             `json:"env" toml:"env"`
	Logger          Logger             `json:"log" toml:"log"`
	Server          Server             `json:"server" toml:"server"`
	Keys            Keys               `json:"keys" toml:"keys"`
	Redis           Redis              `json:"redis" toml:"redis"`
	Base            Base               `json:"base" toml:"base"`
	OSS             OSS                `json:"oss" toml:"oss"`
	OSSPic          OSS                `json:"oss_pic" toml:"oss_pic"`
	Wechat          Wechat             `json:"wechat" toml:"wechat"`
	TokensRate      map[string]float32 `json:"tokens_rate" toml:"tokens_rate"`
	Recommendations []Recommendation   `json:"recommendations" toml:"recommendations"`
	COSEKeys        struct {
		Hmac   key.Key
		Aesgcm key.Key
	}

	globalJobs int64 // global async jobs counter for graceful shutdown
}

func (c *ConfigTpl) Validate() error {
	var err error
	execDir := os.Getenv("EXEC_DIR_PATH")
	if execDir != "" {
		c.Keys.Hmac = filepath.Join(execDir, c.Keys.Hmac)
		c.Keys.Aesgcm = filepath.Join(execDir, c.Keys.Aesgcm)
	}

	if c.COSEKeys.Hmac, err = readKey(c.Keys.Hmac); err != nil {
		return err
	}
	if c.COSEKeys.Aesgcm, err = readKey(c.Keys.Aesgcm); err != nil {
		return err
	}
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

func readKey(filePath string) (k key.Key, err error) {
	var data []byte
	data, err = os.ReadFile(filePath)
	if err != nil {
		return
	}
	data, err = base64.RawURLEncoding.DecodeString(string(data))
	if err != nil {
		return
	}
	err = cbor.Unmarshal(data, &k)
	return
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
