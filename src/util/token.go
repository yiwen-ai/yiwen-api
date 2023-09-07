package util

import (
	"strings"
	"sync"

	"github.com/pkoukk/tiktoken-go"
	tiktoken_loader "github.com/pkoukk/tiktoken-go-loader"
)

var onceTK sync.Once
var tk *tiktoken.Tiktoken
var tokensRate = map[string]float32{
	"eng": 1.00,
	"zho": 1.20,
	"jpn": 1.65,
	"fra": 1.31,
	"kor": 1.57,
	"ara": 2.10,
}

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

func getTokensRate(lang string) float32 {
	if v, ok := tokensRate[strings.ToLower(lang)]; ok {
		return v
	}
	return 1.0
}

func EstimateTranslatingTokens(text, srcLang, dstLang string) uint32 {
	tokens := Tiktokens(text)
	return tokens + uint32(float32(tokens)*getTokensRate(dstLang)/getTokensRate(srcLang))
}
