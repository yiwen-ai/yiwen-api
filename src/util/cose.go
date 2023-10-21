package util

import (
	"encoding/base64"
	"errors"

	"github.com/ldclabs/cose/cose"
	"github.com/ldclabs/cose/key"
)

func EncodeMac0[T any](macer key.MACer, obj T, externalData []byte) (string, error) {
	m := &cose.Mac0Message[T]{
		Protected:   cose.Headers{},
		Unprotected: cose.Headers{},
		Payload:     obj,
	}
	data, err := m.ComputeAndEncode(macer, externalData)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func DecodeMac0[T any](macer key.MACer, input string, externalData []byte) (*T, error) {
	if input == "" {
		return nil, errors.New("empty input")
	}
	data, err := base64.RawURLEncoding.DecodeString(input)
	if err != nil {
		return nil, err
	}

	obj, err := cose.VerifyMac0Message[T](macer, data, externalData)
	if err != nil {
		return nil, err
	}

	return &obj.Payload, nil
}

func EncodeEncrypt0[T any](encryptor key.Encryptor, obj T, externalData []byte) (string, error) {
	m := &cose.Encrypt0Message[T]{
		Protected:   cose.Headers{},
		Unprotected: cose.Headers{},
		Payload:     obj,
	}
	data, err := m.EncryptAndEncode(encryptor, externalData)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func DecodeEncrypt0[T any](encryptor key.Encryptor, input string, externalData []byte) (*T, error) {
	if input == "" {
		return nil, errors.New("empty input")
	}
	data, err := base64.RawURLEncoding.DecodeString(input)
	if err != nil {
		return nil, err
	}

	obj, err := cose.DecryptEncrypt0Message[T](encryptor, data, externalData)
	if err != nil {
		return nil, err
	}

	return &obj.Payload, nil
}
