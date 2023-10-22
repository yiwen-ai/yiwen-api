package api

import (
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/bll"
	"github.com/yiwen-ai/yiwen-api/src/middleware"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Collection struct {
	blls *bll.Blls
}

func (a *Collection) Get(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	role, _ := a.checkReadPermission(ctx, input.GID)
	status := int8(2)
	switch role {
	case 2, 1:
		status = -1
	case 0:
		status = 0
	case -1:
		status = 1
	}

	output, err := a.blls.Writing.GetCollection(ctx, input, status)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if output.Subscription != nil {
		subtoken, err := util.EncodeMac0(a.blls.MACer, SubscriptionToken{
			Kind:     2,
			ExpireAt: output.Subscription.ExpireAt,
			UID:      output.Subscription.UID,
			CID:      output.Subscription.CID,
			GID:      output.Subscription.GID,
		}, []byte("SubscriptionToken"))
		if err == nil {
			output.SubToken = &subtoken
		}
	}
	result := bll.CollectionOutputs{*output}
	result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(bll.SuccessResponse[*bll.CollectionOutput]{Result: &result[0]})
}

func (a *Collection) ListByChild(ctx *gear.Context) error {
	input := &bll.QueryGidCid{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}
	role, _ := a.checkReadPermission(ctx, input.GID)
	input.Status = int8(2)
	switch role {
	case 2, 1, 0:
		input.Status = 0
	case -1:
		input.Status = 1
	}
	input.Fields = "gid,status,info"

	output, err := a.blls.Writing.ListCollectionByChild(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	for i := range output.Result {
		if s := output.Result[i].Subscription; s != nil {
			subtoken, err := util.EncodeMac0(a.blls.MACer, SubscriptionToken{
				Kind:     2,
				ExpireAt: s.ExpireAt,
				UID:      s.UID,
				CID:      s.CID,
				GID:      s.GID,
			}, []byte("SubscriptionToken"))
			if err == nil {
				output.Result[i].SubToken = &subtoken
			}
		}
	}

	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Collection) ListChildren(ctx *gear.Context) error {
	input := &bll.IDGIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	role, _ := a.checkReadPermission(ctx, input.GID)
	input.Status = util.Ptr(int8(2))
	switch role {
	case 2, 1, 0:
		input.Status = util.Ptr(int8(0))
	case -1:
		input.Status = util.Ptr(int8(1))
	}

	output, err := a.blls.Writing.ListCollectionChildren(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(output)
}

func (a *Collection) List(ctx *gear.Context) error {
	input := &bll.GIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	role, _ := a.checkReadPermission(ctx, input.GID)
	input.Status = util.Ptr(int8(2))
	switch role {
	case 2, 1, 0:
		input.Status = util.Ptr(int8(0))
	case -1:
		input.Status = util.Ptr(int8(1))
	}

	output, err := a.blls.Writing.ListCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Collection) ListArchived(ctx *gear.Context) error {
	input := &bll.GIDPagination{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	input.Status = util.Ptr(int8(-1))
	output, err := a.blls.Writing.ListCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	output.Result.LoadGroups(func(ids ...util.ID) []bll.GroupInfo {
		return a.blls.Userbase.LoadGroupInfo(ctx, ids...)
	})

	return ctx.OkSend(output)
}

func (a *Collection) Create(ctx *gear.Context) error {
	input := &bll.CreateCollectionInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.CreateCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[*bll.CollectionOutput]{Result: output})
}

func (a *Collection) Update(ctx *gear.Context) error {
	input := &bll.UpdateCollectionInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.UpdateCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[*bll.CollectionOutput]{Result: output})
}

func (a *Collection) Delete(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.DeleteCollection(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Collection) GetInfo(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.GetCollectionInfo(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[*bll.MessageOutput]{Result: output})
}

func (a *Collection) UpdateInfo(ctx *gear.Context) error {
	input := &bll.UpdateMessageInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.UpdateCollectionInfo(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[*bll.MessageOutput]{Result: output})
}

func (a *Collection) UpdateStatus(ctx *gear.Context) error {
	input := &bll.UpdateStatusInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.UpdateCollectionStatus(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	return ctx.OkSend(bll.SuccessResponse[*bll.CollectionOutput]{Result: output})
}

func (a *Collection) AddChildren(ctx *gear.Context) error {
	input := &bll.AddCollectionChildrenInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.AddCollectionChildren(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[[]util.ID]{Result: output})
}

func (a *Collection) UpdateChild(ctx *gear.Context) error {
	input := &bll.UpdateCollectionChildInput{}
	if err := ctx.ParseBody(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.UpdateCollectionChild(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Collection) RemoveChild(ctx *gear.Context) error {
	input := &bll.QueryGidIdCid{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	if err := a.checkWritePermission(ctx, input.GID); err != nil {
		return err
	}

	output, err := a.blls.Writing.RemoveCollectionChild(ctx, input)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return ctx.OkSend(bll.SuccessResponse[bool]{Result: output})
}

func (a *Collection) UploadFile(ctx *gear.Context) error {
	input := &bll.QueryGidID{}
	if err := ctx.ParseURL(input); err != nil {
		return err
	}

	err := a.checkWritePermission(ctx, input.GID)
	if err != nil {
		return err
	}

	input.Fields = "gid,status"
	doc, err := a.blls.Writing.GetCollection(ctx, input, -1)
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}

	if *doc.Status < 0 {
		return gear.ErrBadRequest.WithMsg("collection archived")
	}

	output := a.blls.Writing.SignPostPolicy(doc.GID, doc.ID, "", 0)
	return ctx.OkSend(bll.SuccessResponse[service.PostFilePolicy]{Result: output})
}

func (a *Collection) checkReadPermission(ctx *gear.Context, gid util.ID) (int8, error) {
	sess := gear.CtxValue[middleware.Session](ctx)
	if sess == nil || sess.UserID.Compare(util.MinID) <= 0 {
		return -2, gear.ErrForbidden.WithMsg("no permission")
	}

	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
	if err != nil {
		return -2, gear.ErrNotFound.From(err)
	}
	if role < -1 {
		return role, gear.ErrForbidden.WithMsg("no permission")
	}

	return role, nil
}

func (a *Collection) checkWritePermission(ctx *gear.Context, gid util.ID) error {
	sess := gear.CtxValue[middleware.Session](ctx)
	if sess == nil || sess.UserID.Compare(util.MinID) <= 0 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	role, err := a.blls.Userbase.UserGroupRole(ctx, sess.UserID, gid)
	if err != nil {
		return gear.ErrNotFound.From(err)
	}
	if role <= 0 {
		return gear.ErrForbidden.WithMsg("no permission")
	}

	return nil
}
