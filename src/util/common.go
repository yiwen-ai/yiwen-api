package util

import (
	crand "crypto/rand"
	"encoding/base64"
	"math/rand"
	"time"
	"unicode/utf8"
)

var mathR *rand.Rand

func init() {
	mathR = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func Int63n(n int64) int64 {
	return mathR.Int63n(n)
}

func RandString(n int) string {
	b := make([]byte, n)
	crand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func SliceHas[T comparable](sl []T, v T) bool {
	for _, s := range sl {
		if v == s {
			return true
		}
	}
	return false
}

func Ptr[T any](t T) *T {
	return &t
}

func RemoveDuplicates[T comparable](sl []T) []T {
	var res []T
	for _, s := range sl {
		if !SliceHas(res, s) {
			res = append(res, s)
		}
	}
	return res
}

func Reverse(s string) string {
	size := len(s)
	buf := make([]byte, size)
	for start := 0; start < size; {
		r, n := utf8.DecodeRuneInString(s[start:])
		start += n
		utf8.EncodeRune(buf[size-start:], r)
	}
	return string(buf)
}
