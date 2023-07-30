package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/yiwen-ai/yiwen-api/src/util"
)

type CreateCreationInput struct {
	GID         util.ID    `json:"gid" cbor:"gid" validate:"required"`
	Title       string     `json:"title" cbor:"title" validate:"required,gte=4,lte=256"`
	Content     util.Bytes `json:"content" cbor:"content" validate:"required"`
	Language    string     `json:"language" cbor:"language"`
	OriginalUrl *string    `json:"original_url,omitempty" cbor:"original_url,omitempty" validate:"omitempty,http_url"`
	Genre       *[]string  `json:"genre,omitempty" cbor:"genre,omitempty"`
	Cover       *string    `json:"cover,omitempty" cbor:"cover,omitempty" validate:"omitempty,http_url"`
	Keywords    *[]string  `json:"keywords,omitempty" cbor:"keywords,omitempty" validate:"omitempty,gte=0,lte=5"`
	Labels      *[]string  `json:"labels,omitempty" cbor:"labels,omitempty" validate:"omitempty,gte=0,lte=5"`
	Authors     *[]string  `json:"authors,omitempty" cbor:"authors,omitempty" validate:"omitempty,gte=0,lte=10"`
	License     *string    `json:"license,omitempty" cbor:"license,omitempty"`
}

type CreationOutput struct {
	ID          util.ID     `json:"id" cbor:"id"`
	GID         util.ID     `json:"gid" cbor:"gid"`
	Status      *int8       `json:"status,omitempty" cbor:"status,omitempty"`
	Rating      *int8       `json:"rating,omitempty" cbor:"rating,omitempty"`
	Version     *uint16     `json:"version,omitempty" cbor:"version,omitempty"`
	Language    *string     `json:"language,omitempty" cbor:"language,omitempty"`
	Creator     *util.ID    `json:"creator,omitempty" cbor:"creator,omitempty"`
	CreatedAt   *int64      `json:"created_at,omitempty" cbor:"created_at,omitempty"`
	UpdatedAt   *int64      `json:"updated_at,omitempty" cbor:"updated_at,omitempty"`
	OriginalUrl *string     `json:"original_url,omitempty" cbor:"original_url,omitempty"`
	Genre       *[]string   `json:"genre,omitempty" cbor:"genre,omitempty"`
	Title       *string     `json:"title,omitempty" cbor:"title,omitempty"`
	Cover       *string     `json:"cover,omitempty" cbor:"cover,omitempty"`
	Keywords    *[]string   `json:"keywords,omitempty" cbor:"keywords,omitempty"`
	Labels      *[]string   `json:"labels,omitempty" cbor:"labels,omitempty"`
	Authors     *[]string   `json:"authors,omitempty" cbor:"authors,omitempty"`
	Reviewers   *[]util.ID  `json:"reviewers,omitempty" cbor:"reviewers,omitempty"`
	Summary     *string     `json:"summary,omitempty" cbor:"summary,omitempty"`
	Content     *util.Bytes `json:"content,omitempty" cbor:"content,omitempty"`
	License     *string     `json:"license,omitempty" cbor:"license,omitempty"`
}

func CreateCreation(ctx context.Context, input *CreateCreationInput) (*CreationOutput, error) {
	output := SuccessResponse[CreationOutput]{}
	if err := util.RequestCBOR(ctx, util.ExternalHTTPClient, "POST", apiHost+"/v1/creation", input, &output); err != nil {
		return nil, err
	}

	joutput := output.Result
	joutput.Content = nil
	data, err := json.Marshal(joutput)
	if err != nil {
		return nil, err
	}
	fmt.Printf("createCreation: %s, length: %d\n", string(data), len(*output.Result.Content))
	return &output.Result, nil
}

// type QueryCreation struct {
// 	GID    util.ID `json:"gid" cbor:"gid" query:"gid" validate:"required"`
// 	ID     util.ID `json:"id" cbor:"id" query:"id" validate:"required"`
// 	Fields string  `json:"fields" cbor:"fields" query:"fields"`
// }

func ListCreation(ctx context.Context, input *GIDPagination) ([]CreationOutput, error) {
	output := SuccessResponse[[]CreationOutput]{}

	if err := util.RequestCBOR(ctx, util.ExternalHTTPClient, "POST", apiHost+"/v1/creation/list", input, &output); err != nil {
		return nil, err
	}

	joutput := output.Result
	data, err := json.Marshal(joutput)
	if err != nil {
		return nil, err
	}
	fmt.Printf("ListCreation: %s\n", string(data))
	return output.Result, nil
}

func GetCreation(ctx context.Context, gid, cid string) (*CreationOutput, error) {
	output := SuccessResponse[CreationOutput]{}
	query := url.Values{}
	query.Add("gid", gid)
	query.Add("id", cid)

	if err := util.RequestCBOR(ctx, util.ExternalHTTPClient, "GET", apiHost+"/v1/creation?"+query.Encode(), nil, &output); err != nil {
		return nil, err
	}

	joutput := output.Result
	joutput.Content = nil
	data, err := json.Marshal(joutput)
	if err != nil {
		return nil, err
	}
	fmt.Printf("GetCreation: %s, length: %d\n", string(data), len(*output.Result.Content))
	return &output.Result, nil
}

type UpdateCreationInput struct {
	GID       util.ID   `json:"gid" cbor:"gid" validate:"required"`
	ID        util.ID   `json:"id" cbor:"id" validate:"required"`
	UpdatedAt int64     `json:"updated_at" cbor:"updated_at"  validate:"required"`
	Title     *string   `json:"title,omitempty" cbor:"title,omitempty" validate:"required,gte=4,lte=256"`
	Cover     *string   `json:"cover,omitempty" cbor:"cover,omitempty" validate:"omitempty,http_url"`
	Keywords  *[]string `json:"keywords,omitempty" cbor:"keywords,omitempty" validate:"omitempty,gte=0,lte=5"`
	Labels    *[]string `json:"labels,omitempty" cbor:"labels,omitempty" validate:"omitempty,gte=0,lte=5"`
	Authors   *[]string `json:"authors,omitempty" cbor:"authors,omitempty" validate:"omitempty,gte=0,lte=10"`
	Summary   *string   `json:"summary,omitempty" cbor:"summary,omitempty" validate:"omitempty,gte=4,lte=2048"`
	License   *string   `json:"license,omitempty" cbor:"license,omitempty"`
}

func UpdateCreation(ctx context.Context, input *UpdateCreationInput) (*CreationOutput, error) {
	output := SuccessResponse[CreationOutput]{}
	if err := util.RequestCBOR(ctx, util.ExternalHTTPClient, "PATCH", apiHost+"/v1/creation", input, &output); err != nil {
		return nil, err
	}

	joutput := output.Result
	joutput.Content = nil
	data, err := json.Marshal(joutput)
	if err != nil {
		return nil, err
	}
	fmt.Printf("UpdateCreation: %s\n", string(data))
	return &output.Result, nil
}

func ReleaseCreation(ctx context.Context, input *CreatePublicationInput) (*SuccessResponse[*PublicationOutput], error) {
	output := SuccessResponse[*PublicationOutput]{}
	if err := util.RequestCBOR(ctx, util.ExternalHTTPClient, "POST", apiHost+"/v1/creation/release", input, &output); err != nil {
		return nil, err
	}

	joutput := output
	data, err := json.Marshal(joutput)
	if err != nil {
		return nil, err
	}
	fmt.Printf("ReleaseCreation: %s\n", string(data))
	return &output, nil
}
