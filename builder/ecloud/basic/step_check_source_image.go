package basic

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep"
	ims "gitlab.ecloud.com/ecloud/ecloudsdkims"
	"gitlab.ecloud.com/ecloud/ecloudsdkims/model"
)

type stepCheckSourceImage struct {
	sourceImageId   string
	sourceImageName string
}

func (s *stepCheckSourceImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("ims_client").(*ims.Client)

	query := model.ListImageRespV2Query{}
	if s.sourceImageId != "" {
		Say(state, s.sourceImageId, "Trying to check source image")
		query.ImageId = &s.sourceImageId
	}
	if s.sourceImageName != "" {
		Say(state, s.sourceImageName, "Trying to check source image")
		query.Name = &s.sourceImageName
	}
	request := model.ListImageRespV2Request{
		ListImageRespV2Query: &query,
	}
	response, err := client.ListImageRespV2(&request)
	if err != nil {
		return Halt(state, err, "Failed to get source image info")
	}
	if response == nil || response.Body == nil {
		return Halt(state, fmt.Errorf("received nil response or nil body"), "")
	}
	if response.Body.Total != nil && *response.Body.Total > 0 && response.Body.Content != nil {
		images := *response.Body.Content
		if len(images) > 0 {
			state.Put("source_image", &images[0])
			state.Put("source_image_name", *images[0].Name)
			Message(state, *images[0].Name, "Image found")
			return multistep.ActionContinue
		}
	}
	return Halt(state, fmt.Errorf("No image found under current image_id(%s) restriction", s.sourceImageId), "")
}

func (s *stepCheckSourceImage) Cleanup(bag multistep.StateBag) {}
