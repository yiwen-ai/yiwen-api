package util

import "unicode/utf8"

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
