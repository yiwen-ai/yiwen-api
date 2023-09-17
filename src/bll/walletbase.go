package bll

import (
	"context"
	"math"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

var AIModels = []AIModel{
	// 4K context,	i: $0.0015/1K tokens; o: $0.002/1K tokens; 0.157 WEN/1K tokens
	{ID: "gpt-3.5", Name: "GPT-3.5", Price: 1}, // 0.6 wen/1K tokens
	// 8K context,	i: $0.03/1K tokens; o: $0.06/1K tokens; 4.7 WEN/1K tokens
	{ID: "gpt-4", Name: "GPT-4", Price: 10}, // 10 wen/1K tokens
}

var DefaultModel = AIModels[0]

type AIModel struct {
	ID    string  `json:"id" cbor:"id"`
	Name  string  `json:"name" cbor:"name"`
	Price float64 `json:"price" cbor:"price"`
}

func GetAIModel(name string) AIModel {
	for i := range AIModels {
		if AIModels[i].ID == strings.ToLower(name) {
			return AIModels[i]
		}
	}

	return AIModels[0]
}

func (m *AIModel) CostWEN(tokens uint32) int64 {
	f := m.Price * float64(tokens) / 1000
	c := int64(f)
	if f > float64(c) {
		c += 1
	}

	if c < 1 {
		c = 1
	}
	return c
}

type Walletbase struct {
	svc service.APIHost
}

type SpendInput struct {
	UID         util.ID    `json:"uid" cbor:"uid"`
	Amount      int64      `json:"amount" cbor:"amount"`
	Description string     `json:"description,omitempty" cbor:"description,omitempty"`
	Payload     util.Bytes `json:"payload,omitempty" cbor:"payload,omitempty"`
}

type SpendPayload struct {
	GID      util.ID `json:"gid" cbor:"gid"`
	CID      util.ID `json:"cid" cbor:"cid"`
	Language string  `json:"language" cbor:"language"`
	Version  uint16  `json:"version" cbor:"version"`
	Model    string  `json:"model" cbor:"model"`
	Price    float64 `json:"price" cbor:"price"`
	Tokens   uint32  `json:"tokens" cbor:"tokens"`
}

type WalletOutput struct {
	Sequence uint64  `json:"sequence" cbor:"sequence"`
	Award    int64   `json:"award" cbor:"award"`
	Topup    int64   `json:"topup" cbor:"topup"`
	Income   int64   `json:"income" cbor:"income"`
	Credits  uint64  `json:"credits" cbor:"credits"`
	Level    uint8   `json:"level" cbor:"level"`
	Txn      util.ID `json:"txn" cbor:"txn"`
}

func (w *WalletOutput) Balance() int64 {
	return w.Award + w.Topup + w.Income
}

func (w *WalletOutput) SetLevel() {
	if w.Credits > 0 {
		w.Level = uint8(math.Floor(math.Log10(float64(w.Credits))))
	}
}

func (b *Walletbase) Get(ctx context.Context, uid util.ID) (*WalletOutput, error) {
	output := SuccessResponse[WalletOutput]{}
	if err := b.svc.Get(ctx, "/v1/wallet?uid="+uid.String(), &output); err != nil {
		return nil, err
	}
	output.Result.SetLevel()
	return &output.Result, nil
}

func (b *Walletbase) Spend(ctx context.Context, uid util.ID, input *SpendPayload) (*WalletOutput, error) {
	data, err := cbor.Marshal(input)
	if err != nil {
		return nil, err
	}
	m := GetAIModel(input.Model)
	input.Model = m.ID
	input.Price = m.Price

	ex := SpendInput{
		UID:         uid,
		Amount:      m.CostWEN(input.Tokens),
		Description: "publication.create",
		Payload:     data,
	}
	output := SuccessResponse[WalletOutput]{}
	if err := b.svc.Post(ctx, "/v1/wallet/spend", ex, &output); err != nil {
		return nil, err
	}

	output.Result.SetLevel()
	return &output.Result, nil
}

type TransactionPK struct {
	UID util.ID `json:"uid" cbor:"uid" query:"uid" validate:"required"`
	ID  util.ID `json:"id" cbor:"id" query:"id" validate:"required"`
}

type TransactionOutput struct {
	ID          util.ID     `json:"id" cbor:"id"`
	Sequence    int64       `json:"sequence" cbor:"sequence"`
	Payee       *util.ID    `json:"payee,omitempty" cbor:"payee,omitempty"`
	SubPayee    *util.ID    `json:"sub_payee,omitempty" cbor:"sub_payee,omitempty"`
	Status      int8        `json:"status" cbor:"status"`
	Kind        string      `json:"kind" cbor:"kind"`
	Amount      int64       `json:"amount" cbor:"amount"`
	SysFee      int64       `json:"sys_fee" cbor:"sys_fee"`
	SubShares   int64       `json:"sub_shares" cbor:"sub_shares"`
	Description string      `json:"description,omitempty" cbor:"description,omitempty"`
	Payload     *util.Bytes `json:"payload,omitempty" cbor:"payload,omitempty"`
}

func (b *Walletbase) CommitExpending(ctx context.Context, input *TransactionPK) error {
	output := SuccessResponse[TransactionOutput]{}
	if err := b.svc.Post(ctx, "/v1/transaction/commit", input, &output); err != nil {
		return err
	}

	return nil
}

func (b *Walletbase) CancelExpending(ctx context.Context, input *TransactionPK) error {
	output := SuccessResponse[TransactionOutput]{}
	if err := b.svc.Post(ctx, "/v1/transaction/cancel", input, &output); err != nil {
		return err
	}

	return nil
}
