package bll

import (
	"context"
	"fmt"

	"github.com/teambition/gear"
	"github.com/yiwen-ai/yiwen-api/src/logging"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Userbase struct {
	svc service.APIHost
}

func (b *Userbase) UserGroupRole(ctx context.Context, uid, gid util.ID) (int8, error) {
	if uid == gid {
		return 2, nil
	}

	output := SuccessResponse[GroupInfo]{}
	api := fmt.Sprintf("/v1/user/get_group?gid=%s&fields=cn,status", gid.String())
	err := b.svc.Get(ctx, api, &output)
	if err == nil && output.Result.Status > -2 && output.Result.Role != nil {
		role := *output.Result.Role
		if role > -2 {
			return role, nil
		}

		return -2, gear.ErrForbidden.WithMsg("no permission")
	}

	return -2, gear.ErrForbidden.WithMsg("user not in group")
}

type IDs struct {
	IDs []util.ID `json:"ids" cbor:"ids"`
}

func (b *Userbase) LoadUserInfo(ctx context.Context, ids ...util.ID) []UserInfo {
	output := SuccessResponse[[]UserInfo]{}
	if len(ids) == 0 {
		return []UserInfo{}
	}

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

	err := b.svc.Post(ctx, "/v1/group/batch_get_info", IDs{ids}, &output)
	if err != nil {
		logging.Warningf("Userbase.LoadGroupInfo error: %v", err)
		return []GroupInfo{}
	}

	return output.Result
}
