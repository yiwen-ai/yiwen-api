package bll

import (
	"context"

	"github.com/fxamacker/cbor/v2"
	"github.com/yiwen-ai/yiwen-api/src/logging"
	"github.com/yiwen-ai/yiwen-api/src/service"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

type Taskbase struct {
	svc service.APIHost
}

type CreateTaskInput struct {
	UID       util.ID    `json:"uid" cbor:"uid"`
	GID       util.ID    `json:"gid" cbor:"gid"`
	Kind      string     `json:"kind" cbor:"kind"`
	Threshold int8       `json:"threshold" cbor:"threshold"`
	Approvers []util.ID  `json:"approvers" cbor:"approvers"`
	Assignees []util.ID  `json:"assignees" cbor:"assignees"`
	Message   string     `json:"message" cbor:"message"`
	Payload   util.Bytes `json:"payload" cbor:"payload"`
	GroupRole *int8      `json:"group_role,omitempty" cbor:"group_role,omitempty"`
}

func (b *Taskbase) Create(ctx context.Context, input *CreateTaskInput, payload any) {
	var err error
	if payload != nil {
		input.Payload, err = cbor.Marshal(payload)
	}
	if err == nil {
		output := SuccessResponse[any]{}
		err = b.svc.Post(ctx, "/v1/task", input, &output)
	}

	if err != nil {
		logging.Errf("failed to create task: %v", err)
	}
}
