// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package basic

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep"
	ims "gitlab.ecloud.com/ecloud/ecloudsdkims"
	"gitlab.ecloud.com/ecloud/ecloudsdkims/model"
	"strings"
)

type stepShareImage struct {
	ShareAccounts []string
}

func (s *stepShareImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if len(s.ShareAccounts) == 0 {
		return multistep.ActionContinue
	}
	client := state.Get("ims_client").(*ims.Client)
	imageId := state.Get("image").(*model.ListImageRespV2ResponseContent).ImageId
	Say(state, strings.Join(s.ShareAccounts, ","), "Trying to share image to")

	request := &model.ShareImageV2Request{}
	shareImageBody := &model.ShareImageV2Body{ImageId: imageId}
	request.SetShareImageV2Body(shareImageBody)
	for i := 0; i < len(s.ShareAccounts); i++ {
		shareImageBody.SetUserName(s.ShareAccounts[i])
		err := Retry(ctx, func(ctx context.Context) error {
			_, e := client.ShareImageV2(request)
			return e
		})
		if err != nil {
			return Halt(state, err, "Failed to share image")
		}
	}
	Message(state, "Image shared", "")
	return multistep.ActionContinue
}

func (s *stepShareImage) Cleanup(state multistep.StateBag) {
	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	if !cancelled && !halted {
		return
	}
	ctx := context.TODO()
	client := state.Get("ims_client").(*ims.Client)
	imageId := state.Get("image").(*model.ListImageRespV2ResponseContent).ImageId

	SayClean(state, "image share")
	request := &model.DeleteShareImageV2Request{
		DeleteShareImageV2Path:  &model.DeleteShareImageV2Path{ImageId: imageId},
		DeleteShareImageV2Query: &model.DeleteShareImageV2Query{SharedUser: s.ShareAccounts},
	}
	err := Retry(ctx, func(ctx context.Context) error {
		_, e := client.DeleteShareImageV2(request)
		return e
	})
	if err != nil {
		Error(state, err, fmt.Sprintf("Failed to cancel share image(%s), please delete it manually", *imageId))
	}

}
