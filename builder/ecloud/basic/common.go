package basic

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/common"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer/packer-plugin-sdk/retry"
	"github.com/hashicorp/packer/packer-plugin-sdk/template/interpolate"
	"github.com/pkg/errors"
	ecsModel "gitlab.ecloud.com/ecloud/ecloudsdkecs/model"
	ims "gitlab.ecloud.com/ecloud/ecloudsdkims"
	imsModel "gitlab.ecloud.com/ecloud/ecloudsdkims/model"
	"strings"
	"time"
)

const (
	BUILDER_ID = "mgtv.ecloud"
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	EcloudAccessConfig  `mapstructure:",squash"`
	EcloudImageConfig   `mapstructure:",squash"`
	EcloudRunConfig     `mapstructure:",squash"`
	ctx                 interpolate.Context
}

const DefaultWaitForInterval = 5

func Halt(state multistep.StateBag, err error, prefix string) multistep.StepAction {
	Error(state, err, prefix)
	state.Put("error", err)

	return multistep.ActionHalt
}

func Error(state multistep.StateBag, err error, prefix string) {
	if prefix != "" {
		err = fmt.Errorf("%s: %s", prefix, err)
	}

	ui := state.Get("ui").(packersdk.Ui)
	ui.Error(err.Error())
}

func Message(state multistep.StateBag, message, prefix string) {
	if prefix != "" {
		message = fmt.Sprintf("%s: %s", prefix, message)
	}

	ui := state.Get("ui").(packersdk.Ui)
	ui.Message(message)
}

func Say(state multistep.StateBag, message, prefix string) {
	if prefix != "" {
		message = fmt.Sprintf("%s: %s", prefix, message)
	}

	if strings.HasPrefix(message, "Trying to") {
		message += "..."
	}

	ui := state.Get("ui").(packersdk.Ui)
	ui.Say(message)
}

// WaitForImageReady wait for image reaches statue
func WaitForImageReady(ctx context.Context, client *ims.Client, imageName string, status string, timeout int) error {
	for {
		image, err := GetImageByName(ctx, client, imageName)
		if err != nil {
			return err
		}

		if image != nil && *image.Status == imsModel.ListImageRespV2ResponseContentStatusEnum(status) {
			return nil
		}

		time.Sleep(DefaultWaitForInterval * time.Second)
		timeout = timeout - DefaultWaitForInterval
		if timeout <= 0 {
			return fmt.Errorf("wait image(%s) status(%s) timeout", imageName, status)
		}
	}
}

func GetImageByName(ctx context.Context, client *ims.Client, imageName string) (*imsModel.ListImageRespV2ResponseContent, error) {
	query := imsModel.ListImageRespV2Query{
		Name: &imageName,
	}
	request := imsModel.ListImageRespV2Request{
		ListImageRespV2Query: &query,
	}

	response, err := client.ListImageRespV2(&request)

	if err != nil {
		return nil, err
	}
	if response == nil || response.Body == nil {
		return nil, errors.New("received nil response or nil body")
	}
	if response.Body.Total != nil && *response.Body.Total > 0 {
		for _, image := range *response.Body.Content {
			if image.Name != nil && *image.Name == imageName {
				return &image, nil
			}
		}
	}

	return nil, nil
}
func SayClean(state multistep.StateBag, module string) {
	_, halted := state.GetOk(multistep.StateHalted)
	_, cancelled := state.GetOk(multistep.StateCancelled)
	if halted {
		Say(state, fmt.Sprintf("Deleting %s because of error...", module), "")
	} else if cancelled {
		Say(state, fmt.Sprintf("Deleting %s because of cancellation...", module), "")
	} else {
		Say(state, fmt.Sprintf("Cleaning up %s...", module), "")
	}
}

func instanceHost(state multistep.StateBag) (string, error) {
	instance := state.Get("instance").(*ecsModel.VmGetServerDetailResponseBody)
	port := (*instance.Ports)[0]
	if len(port.PublicIp) > 0 {
		return port.PublicIp[0], nil
	} else {
		return port.PrivateIp[0], nil
	}
}

func Retry(ctx context.Context, fn func(context.Context) error) error {
	return retry.Config{
		Tries: 60,
		ShouldRetry: func(err error) bool {
			e, ok := err.(*EcloudSDKError)
			if !ok {
				return false
			}
			if e.Code == "ClientError.NetworkError" || e.Code == "ClientError.HttpStatusCodeError" ||
				e.Code == "InvalidKeyPair.NotSupported" || e.Code == "InvalidParameterValue.KeyPairNotSupported" ||
				e.Code == "InvalidInstance.NotSupported" || e.Code == "OperationDenied.InstanceOperationInProgress" ||
				strings.Contains(e.Code, "RequestLimitExceeded") || strings.Contains(e.Code, "InternalError") ||
				strings.Contains(e.Code, "ResourceInUse") || strings.Contains(e.Code, "ResourceBusy") {
				return true
			}
			return false
		},
		RetryDelay: (&retry.Backoff{
			InitialBackoff: 1 * time.Second,
			MaxBackoff:     5 * time.Second,
			Multiplier:     2,
		}).Linear,
	}.Run(ctx, fn)
}
