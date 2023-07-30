package tests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yiwen-ai/yiwen-api/src/util"
)

type SessionOutput struct {
	Sub         *util.UUID `json:"sub,omitempty" cbor:"sub,omitempty"`
	AccessToken string     `json:"access_token,omitempty" cbor:"access_token,omitempty"`
	IDToken     string     `json:"id_token,omitempty" cbor:"id_token,omitempty"`
	ExpiresIn   uint       `json:"expires_in,omitempty" cbor:"expires_in,omitempty"`
}

func GetToken(ctx context.Context) (*SessionOutput, error) {
	output := SessionOutput{}
	if err := util.RequestCBOR(ctx, util.ExternalHTTPClient, "GET", "https://auth.yiwen.ltd/access_token", nil, &output); err != nil {
		return nil, err
	}

	data, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	fmt.Printf("GetToken: %s\n", string(data))
	return &output, nil
}
