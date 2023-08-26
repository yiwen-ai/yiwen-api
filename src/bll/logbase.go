package bll

import (
	"context"
	"errors"
	"net/url"

	"github.com/fxamacker/cbor/v2"
	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

const (
	LogActionSysCreateUser            = "sys.create.user"
	LogActionSysUpdateUser            = "sys.update.user"
	LogActionSysUpdateGroup           = "sys.update.group"
	LogActionSysUpdateCreation        = "sys.update.creation"
	LogActionUserLogin                = "user.login"
	LogActionUserAuthz                = "user.authz"
	LogActionUserUpdate               = "user.update"
	LogActionUserUpdateCN             = "user.update.cn"
	LogActionUserLogout               = "user.logout"
	LogActionUserCollect              = "user.collect"
	LogActionUserFollow               = "user.follow"
	LogActionUserSubscribe            = "user.subscribe"
	LogActionUserSponsor              = "user.sponsor"
	LogActionGroupCreate              = "group.create"
	LogActionGroupUpdate              = "group.update"
	LogActionGroupUpdateCN            = "group.update.cn"
	LogActionGroupTransfer            = "group.transfer"
	LogActionGroupDelete              = "group.delete"
	LogActionGroupCreateUser          = "group.create.user"
	LogActionGroupUpdateUser          = "group.update.user"
	LogActionGroupAddMember           = "group.add.member"
	LogActionGroupUpdateMember        = "group.update.member"
	LogActionGroupRemoveMember        = "group.remove.member"
	LogActionCreationCreate           = "creation.create"
	LogActionCreationCreateConverting = "creation.create.converting"
	LogActionCreationCreateScraping   = "creation.create.scraping"
	LogActionCreationUpdate           = "creation.update"
	LogActionCreationUpdateContent    = "creation.update.content"
	LogActionCreationRelease          = "creation.release"
	LogActionCreationDelete           = "creation.delete"
	LogActionCreationAssist           = "creation.assist"
	LogActionCreationTransfer         = "creation.transfer"
	LogActionPublicationCreate        = "publication.create"
	LogActionPublicationUpdate        = "publication.update"
	LogActionPublicationUpdateContent = "publication.update.content"
	LogActionPublicationPublish       = "publication.publish"
	LogActionPublicationDelete        = "publication.delete"
	LogActionPublicationAssist        = "publication.assist"
)

type Logbase struct {
	svc service.APIHost
}

type LogOutput struct {
	UID     util.ID     `json:"uid" cbor:"uid"`
	ID      util.ID     `json:"id" cbor:"id"`
	Status  int8        `json:"status" cbor:"status"`
	Action  string      `json:"action" cbor:"action"`
	GID     *util.ID    `json:"gid,omitempty" cbor:"gid,omitempty"`
	IP      *string     `json:"ip,omitempty" cbor:"ip,omitempty"`
	Payload *util.Bytes `json:"payload,omitempty" cbor:"payload,omitempty"`
	Tokens  *uint32     `json:"tokens,omitempty" cbor:"tokens,omitempty"`
	Error   *string     `json:"error,omitempty" cbor:"error,omitempty"`
}

func (b *Logbase) Get(ctx context.Context, uid, id util.ID, fields string) (*LogOutput, error) {
	output := SuccessResponse[LogOutput]{}

	query := url.Values{}
	query.Add("uid", uid.String())
	query.Add("id", id.String())
	if fields != "" {
		query.Add("fields", fields)
	}
	// ignore error
	if err := b.svc.Get(ctx, "/v1/log?"+query.Encode(), &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type CreateLogInput struct {
	UID     util.ID    `json:"uid" cbor:"uid"`
	GID     util.ID    `json:"gid" cbor:"gid"`
	Action  string     `json:"action" cbor:"action"`
	Status  int8       `json:"status" cbor:"status"`
	IP      string     `json:"ip" cbor:"ip"`
	Payload util.Bytes `json:"payload" cbor:"payload"`
	Tokens  uint32     `json:"tokens" cbor:"tokens"`
}

func (b *Logbase) Log(ctx *gear.Context, action string, status int8, gid util.ID, payload any) (*LogOutput, error) {
	sess := gear.CtxValue[middleware.Session](ctx)
	if sess == nil {
		return nil, errors.New("no session")
	}

	input := CreateLogInput{
		UID:    sess.UserID,
		GID:    gid,
		Action: action,
		Status: status,
		IP:     ctx.IP().String(),
	}

	if payload != nil {
		data, err := cbor.Marshal(payload)
		if err != nil {
			return nil, err
		}
		input.Payload = data
	}

	output := SuccessResponse[LogOutput]{}
	if err := b.svc.Post(ctx, "/v1/log", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type UpdateLog struct {
	UID     util.ID     `json:"uid" cbor:"uid"`
	ID      util.ID     `json:"id" cbor:"id"`
	Status  int8        `json:"status" cbor:"status"`
	Payload *util.Bytes `json:"payload,omitempty" cbor:"payload,omitempty"`
	Tokens  *uint32     `json:"tokens,omitempty" cbor:"tokens,omitempty"`
	Error   *string     `json:"error,omitempty" cbor:"error,omitempty"`
}

func (b *Logbase) Update(ctx context.Context, input *UpdateLog) (*LogOutput, error) {
	output := SuccessResponse[LogOutput]{}

	if err := b.svc.Patch(ctx, "/v1/log", input, &output); err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type ListRecentlyLogsInput struct {
	UID     util.ID  `json:"uid" cbor:"uid"`
	Actions []string `json:"actions" cbor:"actions"`
	Fields  []string `json:"fields" cbor:"fields"`
}

func (b *Logbase) ListRecently(ctx context.Context, input *ListRecentlyLogsInput) (*SuccessResponse[[]*LogOutput], error) {
	output := SuccessResponse[[]*LogOutput]{}

	if err := b.svc.Post(ctx, "/v1/log/list_recently", input, &output); err != nil {
		return nil, err
	}

	return &output, nil
}

type PublicationJob struct {
	Job    string  `json:"job" cbor:"job"`
	GID    util.ID `json:"gid" cbor:"gid"`
	Status int8    `json:"status" cbor:"status"`
	Action string  `json:"action" cbor:"action"`
	Error  *string `json:"error,omitempty" cbor:"error,omitempty"`
}

func PublicationJobsFrom(input []*LogOutput) []*PublicationJob {
	output := make([]*PublicationJob, 0, len(input))
	for _, v := range input {
		output = append(output, &PublicationJob{
			Job:    v.ID.String(),
			GID:    *v.GID,
			Status: v.Status,
			Action: v.Action,
			Error:  v.Error,
		})
	}
	return output
}
