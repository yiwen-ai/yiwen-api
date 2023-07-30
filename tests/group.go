package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/yiwen-ai/yiwen-api/src/util"
)

type UserInfo struct {
	ID      *util.ID `json:"id,omitempty" cbor:"id,omitempty"` // should clear this field when return to client
	CN      string   `json:"cn" cbor:"cn"`
	Name    string   `json:"name" cbor:"name"`
	Picture string   `json:"picture" cbor:"picture"`
	Status  int8     `json:"status" cbor:"status"`
	Kind    int8     `json:"kind" cbor:"kind"`
}

type GroupInfo struct {
	ID     util.ID `json:"id" cbor:"id"`
	CN     string  `json:"cn" cbor:"cn"`
	Name   string  `json:"name" cbor:"name"`
	Logo   string  `json:"logo" cbor:"logo"`
	Status int8    `json:"status" cbor:"status"`
	MyRole *int8   `json:"_role,omitempty" cbor:"_role,omitempty"`
}

type Group struct {
	ID         *util.ID  `json:"id,omitempty" cbor:"id,omitempty"`
	CN         string    `json:"cn" cbor:"cn"`
	Name       string    `json:"name" cbor:"name"`
	Logo       *string   `json:"logo,omitempty" cbor:"logo,omitempty"`
	Website    *string   `json:"website,omitempty" cbor:"website,omitempty"`
	Status     *int8     `json:"status,omitempty" cbor:"status,omitempty"`
	Kind       *int8     `json:"kind,omitempty" cbor:"kind,omitempty"`
	CreatedAt  *int64    `json:"created_at,omitempty" cbor:"created_at,omitempty"`
	UpdatedAt  *int64    `json:"updated_at,omitempty" cbor:"updated_at,omitempty"`
	Email      *string   `json:"email,omitempty" cbor:"email,omitempty"`
	LegalName  *string   `json:"legal_name,omitempty" cbor:"legal_name,omitempty"`
	Keywords   *[]string `json:"keywords,omitempty" cbor:"keywords,omitempty"`
	Slogan     *string   `json:"slogan,omitempty" cbor:"slogan,omitempty"`
	Address    *string   `json:"address,omitempty" cbor:"address,omitempty"`
	MyRole     *int8     `json:"_role,omitempty" cbor:"_role,omitempty"`
	MyPriority *int8     `json:"_priority,omitempty" cbor:"_priority,omitempty"`
	UID        *util.ID  `json:"uid,omitempty" cbor:"uid,omitempty"` // should clear this field when return to client
	Owner      *UserInfo `json:"owner,omitempty" cbor:"owner,omitempty"`
}

func ListMyGroups(ctx context.Context) ([]Group, error) {
	output := SuccessResponse[[]Group]{}
	if err := util.RequestCBOR(ctx, util.ExternalHTTPClient, "POST", apiHost+"/v1/group/list_my", map[string]string{}, &output); err != nil {
		return nil, err
	}

	if len(output.Result) == 0 {
		return nil, errors.New("no group found")
	}

	data, err := json.Marshal(output.Result)
	if err != nil {
		return nil, err
	}
	fmt.Printf("listMyGroup: %s\n", string(data))
	return output.Result, nil
}
