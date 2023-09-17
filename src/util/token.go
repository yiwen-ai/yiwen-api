package util

import (
	"sync"

	"github.com/pkoukk/tiktoken-go"
	tiktoken_loader "github.com/pkoukk/tiktoken-go-loader"
)

var onceTK sync.Once
var tk *tiktoken.Tiktoken

const MAX_CREATION_TOKENS = 64 * 1024
const MAX_TOKENS = 128 * 1024

func init() {
	onceTK.Do(func() {
		tiktoken.SetBpeLoader(tiktoken_loader.NewOfflineLoader())
		var err error
		tk, err = tiktoken.GetEncoding("cl100k_base")
		if err != nil {
			panic(err)
		}
	})
}

func Tiktokens(input string) uint32 {
	return uint32(len(tk.Encode(input, nil, nil)))
}
