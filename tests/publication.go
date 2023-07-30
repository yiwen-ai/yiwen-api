package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/yiwen-ai/yiwen-api/src/util"
)

type CreatePublicationInput struct {
	GID        util.ID  `json:"gid" cbor:"gid" validate:"required"`
	CID        util.ID  `json:"cid" cbor:"cid" validate:"required"`
	Language   string   `json:"language" cbor:"language" validate:"required"`
	Version    uint16   `json:"version" cbor:"version" validate:"required,gte=1,lte=10000"`
	Model      string   `json:"model" cbor:"model" validate:"omitempty,gte=2,lte=16"`
	ToGID      *util.ID `json:"to_gid,omitempty" cbor:"to_gid,omitempty"`
	ToLanguage *string  `json:"to_language,omitempty" cbor:"to_language,omitempty"`
}

// type CreatePublication struct {
// 	GID      util.ID           `json:"gid" cbor:"gid"`
// 	CID      util.ID           `json:"cid" cbor:"cid"`
// 	Language string            `json:"language" cbor:"language"`
// 	Version  uint16            `json:"version" cbor:"version"`
// 	Draft    *PublicationDraft `json:"draft,omitempty" cbor:"draft,omitempty"`
// }

type PublicationDraft struct {
	GID      util.ID    `json:"gid" cbor:"gid"`
	Language string     `json:"language" cbor:"language"`
	Title    string     `json:"title" cbor:"title"`
	Model    string     `json:"model,omitempty" cbor:"model,omitempty"`
	Genre    []string   `json:"genre,omitempty" cbor:"genre,omitempty"`
	Cover    string     `json:"cover,omitempty" cbor:"cover,omitempty"`
	Keywords []string   `json:"keywords,omitempty" cbor:"keywords,omitempty"`
	Summary  string     `json:"summary" cbor:"summary"`
	Content  util.Bytes `json:"content" cbor:"content"`
}

type PublicationOutput struct {
	GID         util.ID     `json:"gid" cbor:"gid"`
	CID         util.ID     `json:"cid" cbor:"cid"`
	Language    string      `json:"language" cbor:"language"`
	Version     uint16      `json:"version" cbor:"version"`
	Rating      *int8       `json:"rating,omitempty" cbor:"rating,omitempty"`
	Status      *int8       `json:"status,omitempty" cbor:"status,omitempty"`
	Creator     *util.ID    `json:"creator,omitempty" cbor:"creator,omitempty"`
	CreatedAt   *int64      `json:"created_at,omitempty" cbor:"created_at,omitempty"`
	UpdatedAt   *int64      `json:"updated_at,omitempty" cbor:"updated_at,omitempty"`
	Model       *string     `json:"model,omitempty" cbor:"model,omitempty"`
	OriginalUrl *string     `json:"original_url,omitempty" cbor:"original_url,omitempty"`
	Genre       *[]string   `json:"genre,omitempty" cbor:"genre,omitempty"`
	Title       *string     `json:"title,omitempty" cbor:"title,omitempty"`
	Cover       *string     `json:"cover,omitempty" cbor:"cover,omitempty"`
	Keywords    *[]string   `json:"keywords,omitempty" cbor:"keywords,omitempty"`
	Authors     *[]string   `json:"authors,omitempty" cbor:"authors,omitempty"`
	Summary     *string     `json:"summary,omitempty" cbor:"summary,omitempty"`
	Content     *util.Bytes `json:"content,omitempty" cbor:"content,omitempty"`
	License     *string     `json:"license,omitempty" cbor:"license,omitempty"`
}

func GetPublicationByJob(ctx context.Context, job string) (*PublicationOutput, error) {
	output := SuccessResponse[PublicationOutput]{}
	query := url.Values{}
	query.Add("job", job)

	if err := util.RequestCBOR(ctx, util.ExternalHTTPClient, "GET", apiHost+"/v1/publication/job?"+query.Encode(), nil, &output); err != nil {
		return nil, err
	}

	joutput := output.Result
	joutput.Content = nil
	data, err := json.Marshal(joutput)
	if err != nil {
		return nil, err
	}
	fmt.Printf("GetPublicationByJob: %s\n", string(data))
	return &output.Result, nil
}

func CreatePublication(ctx context.Context, input *CreatePublicationInput) (*SuccessResponse[*PublicationOutput], error) {
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
