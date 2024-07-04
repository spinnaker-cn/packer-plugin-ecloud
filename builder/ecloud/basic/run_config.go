//go:generate packer-sdc mapstructure-to-hcl2 -type ecloudDataVolumn

package basic

import (
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer/packer-plugin-sdk/template/interpolate"
	"github.com/hashicorp/packer/packer-plugin-sdk/uuid"
	"github.com/pkg/errors"
)

type ecloudDataVolumn struct {
	IsShare      bool   `mapstructure:"is_share"`
	ResourceType string `mapstructure:"resource_type"`
	Size         int32  `mapstructure:"size"`
}
type EcloudRunConfig struct {
	// The base image id of Image you want to create
	// your customized image from.
	SourceImageId string `mapstructure:"source_image_id" required:"false"`
	// The base image name of Image you want topub create your
	// customized image from.Conflict with SourceImageId.
	SourceImageName string `mapstructure:"source_image_name" required:"false"`
	// The zone of the instance
	Zone string `mapstructure:"zone" required:"true"`
	// Charge type of cvm, values can be `HOUR` (default) `SPOTPAID`
	BillingType string `mapstructure:"billing_type" required:"false"`
	// The instance type your cvm will be launched by.
	// You should reference [Instance Type](https://intl.cloud.tencent.com/document/product/213/11518)
	// for parameter taking.
	InstanceType string `mapstructure:"instance_type" required:"true"`
	// Instance name.
	InstanceName string `mapstructure:"instance_name" required:"false"`
	VmType       string `mapstructure:"vm_type" required:"false"`
	Cpu          int32  `mapstructure:"cpu" required:"false"`
	Ram          int32  `mapstructure:"ram" required:"false"`
	// system disk config
	BootVolumnSize int32  `mapstructure:"boot_volumn_size" required:"false"`
	BootVolumnType string `mapstructure:"boot_volumn_type" required:"false"`
	// Add one or more data disks to the instance before creating the image.
	// Note that if the source image has data disk snapshots, this argument
	// will be ignored, and the running instance will use source image data
	// disk settings, in such case, `disk_type` argument will be used as disk
	// type for all data disks, and each data disk size will use the origin
	// value in source image.
	DataVolumns []ecloudDataVolumn `mapstructure:"data_volumns"`
	// Specify vpc your cvm will be launched by.
	VpcId string `mapstructure:"vpc_id" required:"true"`
	// Specify vpc name you will create. if `vpc_id` is not set, Packer will
	// create a vpc for you named this parameter.
	VpcName string `mapstructure:"vpc_name" required:"false"`
	// Specify subnet your cvm will be launched by.
	SubnetId string `mapstructure:"subnet_id" required:"true"`
	// Specify subnet name you will create. if `subnet_id` is not set, Packer will
	// create a subnet for you named this parameter.
	SubnetName  string `mapstructure:"subnet_name" required:"false"`
	NetworkName string `mapstructure:"network_name" required:"false"`
	// Specify cider block of the vpc you will create if vpc_id not set
	CidrBlock string `mapstructure:"cidr_block" required:"false"` // 10.0.0.0/16(default), 172.16.0.0/12, 192.168.0.0/16
	// Specify cider block of the subnet you will create if
	// subnet_id not set
	SubnectCidrBlock string `mapstructure:"subnect_cidr_block" required:"false"`
	//  bind existed public ip to your ecs
	PublicIpAddress string `mapstructure:"public_ip_address" required:"false"`
	// new publicIp type
	PublicIpType string `mapstructure:"public_ip_type" required:"false"`
	// Internet charge type of publicIp, values can be trafficCharge, bandwidthCharge
	ChargeMode string `mapstructure:"charge_mode" required:"false"`
	// Max bandwidth out your cvm will be launched by(in MB).
	// values can be set between 1 ~ 500.
	BandwidthSize int32 `mapstructure:"bandwidth_size" required:"false"`
	// Specify securitygroup your cvm will be launched by.
	SecurityGroupId string `mapstructure:"security_group_id" required:"false"`
	// Specify security name you will create if security_group_id not set.
	SecurityGroupName string `mapstructure:"security_group_name" required:"false"`
	// userdata.
	UserData string `mapstructure:"user_data" required:"false"`

	// Communicator settings
	Comm         communicator.Config `mapstructure:",squash"`
	SSHPrivateIp bool                `mapstructure:"ssh_private_ip"`
}

func (cf *EcloudRunConfig) Prepare(ctx *interpolate.Context) []error {
	packerId := fmt.Sprintf("packer%s", uuid.TimeOrderedUUID()[:8])
	if cf.Comm.SSHKeyPairName == "" && cf.Comm.SSHTemporaryKeyPairName == "" &&
		cf.Comm.SSHPrivateKeyFile == "" && cf.Comm.SSHPassword == "" && cf.Comm.WinRMPassword == "" {
		//ecloud support key pair name length 5~128 with only alphabets and digits
		cf.Comm.SSHTemporaryKeyPairName = packerId
	}

	errs := cf.Comm.Prepare(ctx)
	if cf.SourceImageId == "" && cf.SourceImageName == "" {
		errs = append(errs, errors.New("source_image_id or source_image_name must be specified"))
	}

	if cf.InstanceType == "" {
		errs = append(errs, errors.New("instance_type must be specified"))
	}

	if (cf.VpcId != "" || cf.CidrBlock != "") && cf.SubnetId == "" && cf.SubnectCidrBlock == "" {
		errs = append(errs, errors.New("if vpc cidr_block is specified, then "+
			"subnet_cidr_block must also be specified."))
	}

	if cf.VpcId == "" {
		if cf.VpcName == "" {
			cf.VpcName = packerId
		}
		if cf.CidrBlock == "" {
			cf.CidrBlock = "10.0.0.0/16"
		}
		if cf.SubnetId != "" {
			errs = append(errs, errors.New("can't set subnet_id without set vpc_id"))
		}
	}

	if cf.SubnetId == "" {
		if cf.SubnetName == "" {
			cf.SubnetName = packerId
		}
		if cf.SubnectCidrBlock == "" {
			cf.SubnectCidrBlock = "10.0.8.0/24"
		}
	}

	if cf.SecurityGroupId == "" && cf.SecurityGroupName == "" {
		cf.SecurityGroupName = packerId
	}

	if cf.VmType == "" {
		cf.VmType = "common"
	}
	if cf.Cpu == 0 {
		cf.Cpu = 1
	}
	if cf.Ram == 0 {
		cf.Ram = 2
	}
	validBootVolumnType := []string{
		"highPerformance", "highPerformanceyc", "performanceOptimization", "performanceOptimizationyc",
	}

	var isValidBootVolumn bool
	if cf.BootVolumnType != "" {
		for _, valid := range validBootVolumnType {
			if valid == cf.BootVolumnType {
				isValidBootVolumn = true
			}
		}
	}
	if isValidBootVolumn == false {
		errs = append(errs, errors.New(fmt.Sprintf("specified boot_volumn_type(%s) is invalid", cf.BootVolumnType)))
	} else if cf.BootVolumnType == "" {
		cf.BootVolumnType = "highPerformance"
	}

	if cf.BootVolumnSize < 20 || cf.BootVolumnSize > 1024 {
		cf.BootVolumnSize = 40
	}

	if cf.PublicIpType != "" && cf.PublicIpAddress != "" {
		errs = append(errs, errors.New("can not create a new publicIp when binding an existed publicIp"))
	}

	if cf.PublicIpType != "" {
		if cf.PublicIpType != "MOBILE" && cf.PublicIpType != "MULTI_LINE" {
			errs = append(errs, errors.New(fmt.Sprintf("specified public_ip_type(%s) is invalid", cf.PublicIpType)))
		}
		if cf.ChargeMode != "trafficCharge" || cf.ChargeMode != "bandwidthCharge" {
			cf.ChargeMode = "trafficCharge"
		}
		if cf.BandwidthSize < 1 {
			cf.BandwidthSize = 1
		}

	}

	if cf.InstanceName == "" {
		cf.InstanceName = packerId
	}

	validDataVolumnType := []string{
		"capebs", "ssd", "ssdebs", "capebsyc", "ssdyc", "ssdebsyc",
	}
	if cf.DataVolumns != nil && len(cf.DataVolumns) > 0 {
		for _, volumn := range cf.DataVolumns {
			var isValidDataVolumnType bool
			if volumn.ResourceType != "" {
				for _, valid := range validDataVolumnType {
					if valid == cf.BootVolumnType {
						isValidDataVolumnType = true
					}
				}
			}
			if isValidDataVolumnType == false {
				volumn.ResourceType = "capebs"
			}
			if volumn.Size < 10 || volumn.Size > 32768 {
				volumn.Size = 10
			}
		}
	}

	return errs
}
