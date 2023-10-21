package util

import (
	"fmt"
	"testing"
	"time"

	"github.com/ldclabs/cose/iana"
	"github.com/ldclabs/cose/key/aesgcm"
	"github.com/ldclabs/cose/key/hmac"
	"github.com/stretchr/testify/assert"
)

type PaymentCode struct {
	Kind     int8   `cbor:"1,keyasint"` // 0: subscribe creation; 2: subscribe collection
	ExpireAt uint64 `cbor:"2,keyasint"` // code 的失效时间，unix 秒
	Payee    ID     `cbor:"3,keyasint"` // 收款人 id
	Amount   uint64 `cbor:"4,keyasint"` // 花费的亿文币数量
	UID      ID     `cbor:"5,keyasint"` // 受益人 id
	CID      ID     `cbor:"6,keyasint"` // 订阅对象 id
	Duration uint64 `cbor:"7,keyasint"` // 增加的订阅时长，单位秒
}

func TestMac0(t *testing.T) {
	assert := assert.New(t)

	k, err := hmac.GenerateKey(iana.AlgorithmHMAC_256_64)
	assert.NoError(err)
	macer, err := k.MACer()
	assert.NoError(err)

	obj := PaymentCode{
		Kind:     2,
		ExpireAt: uint64(time.Now().Add(time.Hour).Unix()),
		Payee:    NewID(),
		Amount:   100000,
		UID:      NewID(),
		CID:      NewID(),
		Duration: 3600 * 24 * 7,
	}

	text, err := EncodeMac0(macer, obj, []byte("PaymentCode"))
	assert.NoError(err)
	fmt.Println(len(text), text)

	obj2, err := DecodeMac0[PaymentCode](macer, text, []byte("PaymentCode"))
	assert.NoError(err)
	assert.Equal(obj, *obj2)
}

func TestEncrypt0(t *testing.T) {
	assert := assert.New(t)

	k, err := aesgcm.GenerateKey(iana.AlgorithmA256GCM)
	assert.NoError(err)
	encryptor, err := k.Encryptor()
	assert.NoError(err)

	obj := PaymentCode{
		Kind:     2,
		ExpireAt: uint64(time.Now().Add(time.Hour).Unix()),
		Payee:    NewID(),
		Amount:   100000,
		UID:      NewID(),
		CID:      NewID(),
		Duration: 3600 * 24 * 7,
	}

	text, err := EncodeEncrypt0(encryptor, obj, []byte("PaymentCode"))
	assert.NoError(err)
	fmt.Println(len(text), text)

	obj2, err := DecodeEncrypt0[PaymentCode](encryptor, text, []byte("PaymentCode"))
	assert.NoError(err)
	assert.Equal(obj, *obj2)
}
