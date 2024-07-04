package basic

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep"
	ims "gitlab.ecloud.com/ecloud/ecloudsdkims"
	"gitlab.ecloud.com/ecloud/ecloudsdkims/model"
)

type stepCreateImage struct {
	imageId string
}

func (s *stepCreateImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	client := state.Get("ims_client").(*ims.Client)
	instanceId := state.Get("instance_id").(string)

	Say(state, fmt.Sprintf("Trying to create a new image(%s) from instance(%s)", config.ImageName, instanceId), "")

	req := model.CreateImageAclRequest{
		CreateImageAclBody: &model.CreateImageAclBody{
			ImageName: &config.ImageName,
			Note:      &config.ImageDescription,
			ServerId:  &instanceId,
		},
	}

	//todo: 镜像绑定标签单独接口，暂无用到标签
	/*var tags []*ims.Tag
	for k, v := range config.ImageTags {
		k := k
		v := v
		tags = append(tags, &cvm.Tag{
			Key:   &k,
			Value: &v,
		})
	}

	resourceType := "image"
	if len(tags) > 0 {
		req.TagSpecification = []*cvm.TagSpecification{
			{
				ResourceType: &resourceType,
				Tags:         tags,
			},
		}
	}*/

	var resp *model.CreateImageAclResponse
	err := Retry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = client.CreateImageAcl(&req)
		return e
	})
	if err != nil {
		return Halt(state, err, "Failed to create image")
	}
	Message(state, resp.ToJsonString(), "Create Image Response")
	if resp.ErrorMessage != nil {
		return Halt(state, fmt.Errorf(*resp.ErrorMessage), "Failed to create image")
	}
	Message(state, "Waiting for image ready", "")
	err = WaitForImageReady(ctx, client, config.ImageName, "active", 3600)
	if err != nil {
		return Halt(state, err, "Failed to wait for image ready")
	}

	image, err := GetImageByName(ctx, client, config.ImageName)
	if err != nil {
		return Halt(state, err, "Failed to get image")
	}

	if image == nil {
		return Halt(state, fmt.Errorf("No image return"), "Failed to crate image")
	}

	s.imageId = *image.ImageId
	state.Put("image", image)
	Message(state, s.imageId, "Image created")

	eCloudImages := make(map[string]string)
	eCloudImages[config.Region] = s.imageId
	state.Put("ecloudimages", eCloudImages)

	return multistep.ActionContinue
}

func (s *stepCreateImage) Cleanup(state multistep.StateBag) {
	if s.imageId == "" {
		return
	}

	_, cancelled := state.GetOk(multistep.StateCancelled)
	_, halted := state.GetOk(multistep.StateHalted)
	if !cancelled && !halted {
		return
	}
	ctx := context.TODO()
	client := state.Get("ims_client").(*ims.Client)
	SayClean(state, "image")
	request := &model.DeleteImageV2Request{}
	imageV2Path := &model.DeleteImageV2Path{ImageId: &s.imageId}
	request.SetDeleteImageV2Path(imageV2Path)
	err := Retry(ctx, func(ctx context.Context) error {
		_, e := client.DeleteImageV2(request)
		return e
	})
	if err != nil {
		Error(state, err, fmt.Sprintf("Failed to delete image(%s), please delete it manually", s.imageId))
	}
}
