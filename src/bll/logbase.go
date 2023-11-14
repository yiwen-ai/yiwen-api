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
	LogActionUserBookmark             = "user.bookmark"
	LogActionUserFollow               = "user.follow"
	LogActionUserSpend                = "user.spend"
	LogActionUserWithdraw             = "user.withdraw"
	LogActionUserTopup                = "user.topup"
	LogActionUserRefund               = "user.refund"
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
	LogActionCreationSubscribe        = "creation.subscribe"
	LogActionPublicationCreate        = "publication.create"
	LogActionPublicationUpdate        = "publication.update"
	LogActionPublicationUpdateContent = "publication.update.content"
	LogActionPublicationPublish       = "publication.publish"
	LogActionPublicationDelete        = "publication.delete"
	LogActionPublicationAssist        = "publication.assist"
	LogActionMessageCreate            = "message.create"
	LogActionMessageUpdate            = "message.update"
	LogActionMessageDelete            = "message.delete"
	LogActionCollectionCreate         = "collection.create"
	LogActionCollectionUpdate         = "collection.update"
	LogActionCollectionUpdateChildren = "collection.update.children"
	LogActionCollectionDelete         = "collection.delete"
	LogActionCollectionSubscribe      = "collection.subscribe"
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

type LogPayload struct {
	GID      util.ID `json:"gid" cbor:"gid"`
	CID      util.ID `json:"cid" cbor:"cid"`
	Version  *uint16 `json:"version,omitempty" cbor:"version,omitempty"`
	Language *string `json:"language,omitempty" cbor:"language,omitempty"`
	Kind     *int8   `json:"kind,omitempty" cbor:"kind,omitempty"`
	Status   *int8   `json:"status,omitempty" cbor:"status,omitempty"`
	Rating   *int8   `json:"rating,omitempty" cbor:"rating,omitempty"`
	Price    *int64  `json:"price,omitempty" cbor:"price,omitempty"`
}

type LogMessage struct {
	ID        util.ID  `json:"id" cbor:"id"`
	AttachTo  util.ID  `json:"attach_to" cbor:"attach_to"`
	Kind      *string  `json:"kind,omitempty" cbor:"kind,omitempty"`
	Language  *string  `json:"language,omitempty" cbor:"language,omitempty"`
	Languages []string `json:"languages,omitempty" cbor:"languages,omitempty"`
	Version   *uint16  `json:"version,omitempty" cbor:"version,omitempty"`
}

func (b *Logbase) Log(ctx *gear.Context, action string, status int8, gid util.ID, payload any) (*LogOutput, error) {
	sess := gear.CtxValue[middleware.Session](ctx)
	if sess == nil {
		return nil, errors.New("no session")
	}

	input := CreateLogInput{
		UID:     sess.UserID,
		GID:     gid,
		Action:  action,
		Status:  status,
		Payload: util.Bytes{0xa0}, // {}
		IP:      ctx.IP().String(),
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

func (b *Logbase) ListRecently(ctx context.Context, input *ListRecentlyLogsInput) ([]LogOutput, error) {
	output := SuccessResponse[[]LogOutput]{}

	if err := b.svc.Post(ctx, "/v1/log/list_recently", input, &output); err != nil {
		return nil, err
	}

	return output.Result, nil
}

type PublicationJob struct {
	Job         string            `json:"job" cbor:"job"`
	Status      int8              `json:"status" cbor:"status"`
	Action      string            `json:"action" cbor:"action"`
	Progress    int8              `json:"progress" cbor:"progress"`
	Tokens      uint32            `json:"tokens" cbor:"tokens"`
	Publication PublicationOutput `json:"publication,omitempty" cbor:"publication,omitempty"`
	Error       *string           `json:"error,omitempty" cbor:"error,omitempty"`
}
