package bll

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/logging"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Userbase struct {
	svc service.APIHost
	oss *service.OSS
}

func (b *Userbase) UserGroupRole(ctx context.Context, uid, gid util.ID) (int8, error) {
	if uid == gid {
		return 2, nil
	}
	if gid.Compare(util.MinID) <= 0 {
		return -2, gear.ErrBadRequest.WithMsg("invalid group id")
	}

	output := SuccessResponse[GroupInfo]{}
	api := fmt.Sprintf("/v1/group/get_by_user?id=%s&fields=cn,status", gid.String())
	err := b.svc.Get(ctx, api, &output)
	if err == nil && output.Result.Status > -2 && output.Result.MyRole != nil {
		role := *output.Result.MyRole
		if role > -2 {
			return role, nil
		}

		return -2, gear.ErrForbidden.WithMsg("no permission")
	}

	return -2, gear.ErrForbidden.WithMsg("user not in group")
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

type Groups []Group

func (list *Groups) LoadUsers(loader func(ids ...util.ID) []UserInfo) {
	if len(*list) == 0 {
		return
	}

	ids := make([]util.ID, 0, len(*list))
	for _, g := range *list {
		if g.UID != nil {
			ids = append(ids, *g.UID)
		}
	}

	users := loader(ids...)
	if len(users) == 0 {
		return
	}

	infoMap := make(map[util.ID]*UserInfo, len(users))
	for i := range users {
		infoMap[*users[i].ID] = &users[i]
		infoMap[*users[i].ID].ID = nil
	}

	for i := range *list {
		(*list)[i].Owner = infoMap[*(*list)[i].UID]
		(*list)[i].UID = nil
	}
}

func (b *Userbase) MyGroups(ctx context.Context) (Groups, error) {
	input := Pagination{
		PageSize: util.Ptr(uint16(100)),
		Fields:   &[]string{},
	}

	output := SuccessResponse[Groups]{}
	if err := b.svc.Post(ctx, "/v1/group/list_by_user", input, &output); err != nil {
		return nil, err
	}

	return output.Result, nil
}

type IDs struct {
	IDs []util.ID `json:"ids" cbor:"ids"`
}

func (b *Userbase) LoadUserInfo(ctx context.Context, ids ...util.ID) []UserInfo {
	output := SuccessResponse[[]UserInfo]{}
	if len(ids) == 0 {
		return []UserInfo{}
	}

	ids = util.RemoveDuplicates(ids)
	err := b.svc.Post(ctx, "/v1/user/batch_get_info", IDs{ids}, &output)
	if err != nil {
		logging.Warningf("Userbase.LoadUserInfo error: %v", err)
		return []UserInfo{}
	}

	return output.Result
}

func (b *Userbase) LoadGroupInfo(ctx context.Context, ids ...util.ID) []GroupInfo {
	output := SuccessResponse[[]GroupInfo]{}
	if len(ids) == 0 {
		return []GroupInfo{}
	}

	ids = util.RemoveDuplicates(ids)
	err := b.svc.Post(ctx, "/v1/group/batch_get_info", IDs{ids}, &output)
	if err != nil {
		logging.Warningf("Userbase.LoadGroupInfo error: %v", err)
		return []GroupInfo{}
	}

	return output.Result
}

func (b *Userbase) FollowGroup(ctx context.Context, input *QueryIdCn) (bool, error) {
	output := SuccessResponse[bool]{}
	if err := b.svc.Patch(ctx, "/v1/group/follow", input, &output); err != nil {
		return false, err
	}

	return output.Result, nil
}

func (b *Userbase) UnFollowGroup(ctx context.Context, input *QueryIdCn) (bool, error) {
	output := SuccessResponse[bool]{}
	if err := b.svc.Patch(ctx, "/v1/group/unfollow", input, &output); err != nil {
		return false, err
	}

	return output.Result, nil
}

func (b *Userbase) ListFollowing(ctx context.Context, input *Pagination) (Groups, error) {
	output := SuccessResponse[Groups]{}
	input.Fields = &[]string{"status", "kind", "name", "logo"}
	if err := b.svc.Post(ctx, "/v1/group/list_following", input, &output); err != nil {
		return nil, err
	}

	return output.Result, nil
}

func (b *Userbase) FollowingGids(ctx context.Context) ([]util.ID, error) {
	output := SuccessResponse[[]util.ID]{}
	if err := b.svc.Get(ctx, "/v1/group/following_ids", &output); err != nil {
		return nil, err
	}

	return output.Result, nil
}

func (b *Userbase) GroupInfo(ctx context.Context, input *QueryIdCn) (*GroupInfo, error) {
	output := SuccessResponse[GroupInfo]{}

	query := url.Values{}
	if input.ID != nil {
		query.Add("id", input.ID.String())
	}
	if input.CN != nil {
		query.Add("cn", *input.CN)
	}
	query.Add("fields", "cn,name,logo,status,slogan")

	err := b.svc.Get(ctx, "/v1/group?"+query.Encode(), &output)
	if err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Userbase) GetGroup(ctx context.Context, id util.ID, fields string) (*Group, error) {
	output := SuccessResponse[Group]{}

	query := url.Values{}
	query.Add("id", id.String())
	query.Add("fields", fields)

	err := b.svc.Get(ctx, "/v1/group?"+query.Encode(), &output)
	if err != nil {
		return nil, err
	}

	return &output.Result, nil
}

type UpdateGroupInfoInput struct {
	ID      util.ID `json:"id" cbor:"id" validate:"required"`
	Name    *string `json:"name,omitempty" cbor:"name,omitempty" validate:"omitempty,gte=2,lte=16"`
	Logo    *string `json:"logo,omitempty" cbor:"logo,omitempty" validate:"omitempty,http_url"`
	Slogan  *string `json:"slogan,omitempty" cbor:"slogan,omitempty" validate:"omitempty,gte=0,lte=256"`
	Website *string `json:"website,omitempty" cbor:"website,omitempty" validate:"omitempty,http_url"`
}

func (i *UpdateGroupInfoInput) Validate() error {
	if err := util.Validator.Struct(i); err != nil {
		return gear.ErrBadRequest.From(err)
	}
	if i.Logo != nil && !strings.HasPrefix(*i.Logo, conf.Config.OSSPic.BaseUrl) {
		return gear.ErrBadRequest.WithMsgf("logo must be a url from %s", conf.Config.OSSPic.BaseUrl)
	}
	return nil
}

func (b *Userbase) UpdateGroupInfo(ctx context.Context, input *UpdateGroupInfoInput) (*GroupInfo, error) {
	output := SuccessResponse[GroupInfo]{}

	err := b.svc.Patch(ctx, "/v1/group", input, &output)
	if err != nil {
		return nil, err
	}

	return &output.Result, nil
}

func (b *Userbase) SignPicturePolicy(id util.ID) service.PostFilePolicy {
	return b.oss.SignPicturePolicy(id.String())
}
