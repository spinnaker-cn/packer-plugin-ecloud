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

type stepCopyImage struct {
	DesinationRegions []string
	SourceRegion      string
}

func (s *stepCopyImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	if len(s.DesinationRegions) == 0 || (len(s.DesinationRegions) == 1 && s.DesinationRegions[0] == s.SourceRegion) {
		return multistep.ActionContinue
	}

	config := state.Get("config").(*Config)
	client := state.Get("ims_client").(*ims.Client)
	imageSrc := state.Get("image").(*model.ListImageRespV2ResponseContent)
	imageId := imageSrc.ImageId
	Say(state, strings.Join(s.DesinationRegions, ","), "Trying to copy image to")
	//todo:复制镜像接口只允许传递单个目标资源池
	req := &model.TransferRequest{}
	transferBody := &model.TransferBody{SrcImageId: imageId, SrcNode: &s.SourceRegion, Description: &config.ImageDescription, DesImageName: &config.ImageName, DesNode: &s.DesinationRegions[0]}
	req.SetTransferBody(transferBody)

	err := Retry(ctx, func(ctx context.Context) error {
		_, e := client.Transfer(req)
		return e
	})
	if err != nil {
		return Halt(state, err, "Failed to copy image")
	}

	Message(state, "Waiting for image ready", "")
	ecloudImages := state.Get("ecloudimages").(map[string]string)
	cf := &EcloudAccessConfig{
		AccessKey:     config.AccessKey,
		SecretKey:     config.SecretKey,
		SecurityToken: config.SecurityToken,
		Region:        s.DesinationRegions[0],
	}
	imsClient := NewImsClient(cf)
	if imsClient == nil {
		return Halt(state, err, "Failed to init client")
	}

	err = WaitForImageReady(ctx, imsClient, config.ImageName, "active", 1800)
	if err != nil {
		return Halt(state, err, "Failed to wait for image ready")
	}

	image, err := GetImageByName(ctx, imsClient, config.ImageName)
	if err != nil {
		return Halt(state, err, "Failed to get image")
	}

	if image == nil {
		return Halt(state, err, "Failed to wait for image ready")
	}

	ecloudImages[s.DesinationRegions[0]] = *image.ImageId
	Message(state, fmt.Sprintf("Copy image from %s(%s) to %s(%s)", s.SourceRegion, *imageId, s.DesinationRegions[0], *image.ImageId), "")

	state.Put("ecloudimages", ecloudImages)
	Message(state, "Image copied", "")

	return multistep.ActionContinue
}

func (s *stepCopyImage) Cleanup(state multistep.StateBag) {}
