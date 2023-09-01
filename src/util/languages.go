package util

import "strings"

const LanguagesKey = "languages"

type Languages [][]string //[["eng","English","English"],["zho","Chinese","中文"], ...]

func (l Languages) Get(lang string) []string {
	if len([]byte(lang)) >= 3 {
		la := strings.ToLower(lang)
		for _, vv := range l {
			if vv[0] == la || strings.ToLower(vv[1]) == la || strings.ToLower(vv[2]) == la {
				return vv[:]
			}
		}
	}

	return nil
}
