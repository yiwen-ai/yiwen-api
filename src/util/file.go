package util

import (
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/saintfish/chardet"
	"github.com/teambition/gear"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"
)

func init() {
	mimetype.SetLimit(1024 * 1024) // 1MB
}

func NormalizeFileEncodingAndType(buf []byte, mtype string) ([]byte, string, error) {
	mt := mimetype.Detect(buf)

	var de *chardet.Detector
	switch {
	case mtype == "application/pdf" && mt.Is("application/pdf"):
		return buf, mtype, nil
	case mtype == "text/html" && mt.Is("text/html"):
		de = chardet.NewHtmlDetector()
	case mtype == "text/markdown" && mt.Is("text/plain"):
		de = chardet.NewHtmlDetector()
	case mtype == "text/plain" && mt.Is("text/plain"):
		de = chardet.NewTextDetector()
	default:
		return nil, "", gear.ErrUnsupportedMediaType.WithMsgf("unsupported media type: %s", mt.String())
	}

	rt, err := de.DetectBest(buf)
	if err != nil {
		return nil, "", gear.ErrUnsupportedMediaType.From(err)
	}

	enc, err := ianaindex.IANA.Encoding(rt.Charset)
	if err != nil {
		enc, err = ianaindex.IANA.Encoding(strings.ReplaceAll(rt.Charset, "-", ""))
	}

	if err != nil {
		return nil, "", gear.ErrUnsupportedMediaType.From(err)
	}

	if enc != unicode.UTF8 {
		decoder := enc.NewDecoder()
		buf, err = decoder.Bytes(buf)
		if err != nil {
			return nil, "", gear.ErrUnsupportedMediaType.From(err)
		}
	}

	return buf, mtype, nil
}
