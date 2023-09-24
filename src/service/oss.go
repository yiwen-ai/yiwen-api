package service

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

func init() {
	util.DigProvide(NewOSS)
}

type OSS struct {
	cfg       conf.OSS
	bucket    *oss.Bucket
	cfgPic    conf.OSS
	bucketPic *oss.Bucket
}

func NewOSS() *OSS {
	cfg := conf.Config.OSS
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyId, cfg.AccessKeySecret)
	if err != nil {
		panic(err)
	}
	bucket, err := client.Bucket(cfg.Bucket)
	if err != nil {
		panic(err)
	}

	cfgPic := conf.Config.OSSPic
	clientPic, err := oss.New(cfgPic.Endpoint, cfgPic.AccessKeyId, cfgPic.AccessKeySecret)
	if err != nil {
		panic(err)
	}
	bucketPic, err := clientPic.Bucket(cfgPic.Bucket)
	if err != nil {
		panic(err)
	}

	return &OSS{
		cfg:       cfg,
		bucket:    bucket,
		cfgPic:    cfgPic,
		bucketPic: bucketPic,
	}
}

// https://help.aliyun.com/zh/oss/developer-reference/postobject
// 如 base_url 为 https://fs.yiwen.pub/grv5...9pjc/cil6...4f0/1/zho/
// 上传了文件 yiwen.ai.png，
// 则该文件访问链接为 https://fs.yiwen.pub/grv5...9pjc/cil6...4f0/1/zho/yiwen.ai.png
type PostFilePolicy struct {
	Host      string `json:"host" cbor:"host"`
	Dir       string `json:"dir" cbor:"dir"`
	AccessKey string `json:"access_key" cbor:"access_key"`
	Policy    string `json:"policy" cbor:"policy"`
	Signature string `json:"signature" cbor:"signature"`
	BaseUrl   string `json:"base_url" cbor:"base_url"`
}

// 指定过期时间，单位为秒。
const ossExpiration = 3600 * time.Second
const ossMinContentLength = 1024
const ossMaxContentLength = 1024 * 1024 * 10
const ossCacheControl = "public, max-age=604800, immutable"
const ossContentDisposition = "inline"

var ossContentType = []string{"image/jpg", "image/png", "image/gif", "image/jpeg", "image/webp"}

// https://help.aliyun.com/zh/oss/use-cases/client-direct-transmission-overview
func (s *OSS) SignPostPolicy(gid, cid, lang string, version uint) PostFilePolicy {
	expiration := time.Now().Add(ossExpiration).UTC().Format("2006-01-02T15:04:05.999Z")
	// https://help.aliyun.com/zh/oss/use-cases/oss-performance-and-scalability-best-practices
	// 反转打散分区，避免热点
	dir := fmt.Sprintf("%s/%s/%d/%s/", util.Reverse(cid), gid, version, lang)

	data, _ := json.Marshal(map[string]any{
		"expiration": expiration,
		"conditions": []any{
			// map[string]string{"bucket": "ywfs"},
			[]any{"content-length-range", ossMinContentLength, ossMaxContentLength},
			[]any{"starts-with", "$key", dir},
			[]any{"in", "$content-type", ossContentType},
			[]any{"eq", "$cache-control", ossCacheControl},
			[]any{"eq", "$content-disposition", ossContentDisposition},
		},
	})

	policy := base64.StdEncoding.EncodeToString(data)
	hm := hmac.New(sha1.New, []byte(s.cfg.AccessKeySecret))
	hm.Write([]byte(policy))
	pp := PostFilePolicy{
		Host:      fmt.Sprintf("https://%s.%s", s.cfg.Bucket, s.cfg.Endpoint),
		Dir:       dir,
		AccessKey: s.cfg.AccessKeyId,
		Policy:    policy,
		Signature: base64.StdEncoding.EncodeToString(hm.Sum(nil)),
		BaseUrl:   s.cfg.BaseUrl + dir,
	}
	return pp
}

func (s *OSS) SignPicturePolicy(id string) PostFilePolicy {
	expiration := time.Now().Add(60 * time.Second).UTC().Format("2006-01-02T15:04:05.999Z")
	// https://help.aliyun.com/zh/oss/use-cases/oss-performance-and-scalability-best-practices
	// 反转打散分区，避免热点
	key := fmt.Sprintf("%s/%s/%s", s.cfgPic.Prefix, util.Reverse(id), util.RandString(4))
	data, _ := json.Marshal(map[string]any{
		"expiration": expiration,
		"conditions": []any{
			// map[string]string{"bucket": "ywfs"},
			[]any{"content-length-range", ossMinContentLength, ossMaxContentLength},
			[]any{"eq", "$key", key},
			[]any{"in", "$content-type", ossContentType},
			[]any{"eq", "$cache-control", ossCacheControl},
			[]any{"eq", "$content-disposition", ossContentDisposition},
		},
	})

	policy := base64.StdEncoding.EncodeToString(data)
	hm := hmac.New(sha1.New, []byte(s.cfgPic.AccessKeySecret))
	hm.Write([]byte(policy))
	pp := PostFilePolicy{
		Host:      fmt.Sprintf("https://%s.%s", s.cfgPic.Bucket, s.cfgPic.Endpoint),
		Dir:       key,
		AccessKey: s.cfgPic.AccessKeyId,
		Policy:    policy,
		Signature: base64.StdEncoding.EncodeToString(hm.Sum(nil)),
		BaseUrl:   s.cfgPic.BaseUrl + key,
	}
	return pp
}

func (s *OSS) ListObjects(cid string) (any, error) {
	return s.bucket.ListObjectsV2(oss.Prefix(fmt.Sprintf("%s/", util.Reverse(cid))))
}
