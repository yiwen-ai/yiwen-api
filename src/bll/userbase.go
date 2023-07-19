package bll

import (
	"github.com/yiwen-ai/yiwen-api/src/service"
)

type Userbase struct {
	svc service.APIHost
	oss *service.OSS
}
