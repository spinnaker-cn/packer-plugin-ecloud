package basic

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep"
	ims "gitlab.ecloud.com/ecloud/ecloudsdkims"
)

type stepPreValidate struct {
}

func (s *stepPreValidate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	client := state.Get("ims_client").(*ims.Client)

	Say(state, config.ImageName, "Trying to check image name")

	image, err := GetImageByName(ctx, client, config.ImageName)
	if err != nil {
		return Halt(state, err, "Failed to get images info")
	}

	if image != nil {
		return Halt(state, fmt.Errorf("Image name %s has exists", config.ImageName), "")
	}

	Message(state, "useable", "Image name")

	return multistep.ActionContinue
}

func (s *stepPreValidate) Cleanup(multistep.StateBag) {}
