package basic

import (
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/template/interpolate"
	"gitlab.ecloud.com/ecloud/ecloudsdkcore/config"
	ecs "gitlab.ecloud.com/ecloud/ecloudsdkecs"
	ims "gitlab.ecloud.com/ecloud/ecloudsdkims"
	vpc "gitlab.ecloud.com/ecloud/ecloudsdkvpc"
	"os"
)

const (
	PACKER_ACCESS_KEY     = "ECLOUD_ACCESS_KEY"
	PACKER_SECRET_KEY     = "ECLOUD_SECRET_KEY"
	PACKER_SECURITY_TOKEN = "ECLOUD_SECURITY_TOKEN"
	PACKER_REGION         = "ECLOUD_REGION"
)

type EcloudAccessConfig struct {
	AccessKey     string `mapstructure:"access_key" required:"true"`
	SecretKey     string `mapstructure:"secret_key" required:"true"`
	Region        string `mapstructure:"region" required:"true"`
	SecurityToken string `mapstructure:"security_token" required:"false"`
}

func (ct *EcloudAccessConfig) Client() (*ecs.Client, *vpc.Client, *ims.Client) {
	credential := config.NewConfigBuilder().AccessKey(ct.AccessKey).SecretKey(ct.SecretKey).PoolId(ct.Region).Build()
	ecs_client := ecs.NewClient(credential)
	vpc_client := vpc.NewClient(credential)
	ims_client := ims.NewClient(credential)
	return ecs_client, vpc_client, ims_client
}
func NewImsClient(ct *EcloudAccessConfig) *ims.Client {
	credential := config.NewConfigBuilder().AccessKey(ct.AccessKey).SecretKey(ct.SecretKey).PoolId(ct.Region).Build()
	ims_client := ims.NewClient(credential)
	return ims_client
}
func (ct *EcloudAccessConfig) Prepare(ctx *interpolate.Context) []error {
	errorArray := []error{}

	if ct == nil {
		return append(errorArray, fmt.Errorf("[PRE-FLIGHT] Empty EcloudAccessConfig detected"))
	}

	if err := ct.ValidateKeyPair(); err != nil {
		errorArray = append(errorArray, err)
	}

	if len(errorArray) != 0 {
		return errorArray
	}
	return nil
}

func (ct *EcloudAccessConfig) ValidateKeyPair() error {
	if ct.AccessKey == "" {
		ct.AccessKey = os.Getenv("ECLOUD_ACCESS_KEY")
	}

	if ct.SecretKey == "" {
		ct.SecretKey = os.Getenv("ECLOUD_SECRET_KEY")
	}

	if ct.AccessKey == "" || ct.SecretKey == "" {
		return fmt.Errorf("[PRE-FLIGHT] We can't find your key pairs," +
			"write them here {access_key=xxx , secret_key=xxx} ")
	}
	return nil
}
