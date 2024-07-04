//go:generate packer-sdc mapstructure-to-hcl2 -type Config

package basic

import (
	"context"
	"fmt"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer/packer-plugin-sdk/template/interpolate"
)

type Builder struct {
	config Config
	runner multistep.Runner
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) ([]string, []string, error) {
	err := config.Decode(&b.config, &config.DecodeOpts{
		PluginType:         BUILDER_ID,
		Interpolate:        true,
		InterpolateContext: &b.config.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"boot_command",
			},
		},
	}, raws...)
	b.config.ctx.EnableEnv = true
	if err != nil {
		return nil, nil, fmt.Errorf("[ERROR] Failed in decoding JSON->mapstructure")
	}

	errs := &packersdk.MultiError{}
	errs = packersdk.MultiErrorAppend(errs, b.config.EcloudAccessConfig.Prepare(&b.config.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, b.config.EcloudImageConfig.Prepare(&b.config.ctx)...)
	errs = packersdk.MultiErrorAppend(errs, b.config.EcloudRunConfig.Prepare(&b.config.ctx)...)
	if errs != nil && len(errs.Errors) != 0 {
		return nil, nil, errs
	}

	packersdk.LogSecretFilter.Set(b.config.AccessKey, b.config.SecretKey)

	return nil, nil, nil
}
func (b *Builder) Run(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook) (packersdk.Artifact, error) {
	ecsClient, vpcClient, imsClient := b.config.Client()

	state := new(multistep.BasicStateBag)
	state.Put("config", &b.config)
	state.Put("ecs_client", ecsClient)
	state.Put("vpc_client", vpcClient)
	state.Put("ims_client", imsClient)
	state.Put("hook", hook)
	state.Put("ui", ui)

	var steps []multistep.Step
	steps = []multistep.Step{
		&stepPreValidate{},
		&stepCheckSourceImage{
			sourceImageId:   b.config.SourceImageId,
			sourceImageName: b.config.SourceImageName,
		},
		&stepConfigKeyPair{
			Debug:        b.config.PackerDebug,
			Comm:         &b.config.Comm,
			DebugKeyPath: fmt.Sprintf("ecs_%s.pem", b.config.PackerBuildName),
		},
		&stepConfigVPC{
			VpcId:       b.config.VpcId,
			CidrBlock:   b.config.CidrBlock,
			VpcName:     b.config.VpcName,
			NetworkName: b.config.NetworkName,
			Region:      b.config.Zone,
		},
		&stepConfigSubnet{
			NetworkName:     b.config.NetworkName,
			SubnetId:        b.config.SubnetId,
			SubnetCidrBlock: b.config.SubnectCidrBlock,
			SubnetName:      b.config.SubnetName,
			Zone:            b.config.Zone,
		},
		&stepConfigSecurityGroup{
			SecurityGroupId:   b.config.SecurityGroupId,
			SecurityGroupName: b.config.SecurityGroupName,
			Description:       "securitygroup for packer",
		},
		&stepRunInstance{
			SpecsName:       b.config.InstanceType,
			BillingType:     b.config.BillingType,
			UserData:        b.config.UserData,
			ZoneId:          b.config.Zone,
			InstanceName:    b.config.InstanceName,
			VmType:          b.config.VmType,
			Cpu:             b.config.Cpu,
			Ram:             b.config.Ram,
			PublicIpType:    b.config.PublicIpType,
			BandwidthSize:   b.config.BandwidthSize,
			ChargeMode:      b.config.ChargeMode,
			BootVolumnSize:  b.config.BootVolumnSize,
			BootVolumnType:  b.config.BootVolumnType,
			DataVolumns:     b.config.DataVolumns,
			publicIpAddress: b.config.PublicIpAddress,
		},
		&communicator.StepConnect{
			Host:      instanceHost,
			Config:    &b.config.EcloudRunConfig.Comm,
			SSHConfig: b.config.EcloudRunConfig.Comm.SSHConfigFunc(),
		},
		&commonsteps.StepProvision{},
		&commonsteps.StepCleanupTempKeys{
			Comm: &b.config.EcloudRunConfig.Comm,
		},
		&stepCreateImage{},
		&stepShareImage{
			b.config.ImageShareAccounts,
		},
		&stepCopyImage{
			DesinationRegions: b.config.ImageCopyRegions,
			SourceRegion:      b.config.Region,
		},
	}

	b.runner = commonsteps.NewRunner(steps, b.config.PackerConfig, ui)
	b.runner.Run(ctx, state)

	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	if _, ok := state.GetOk("image"); !ok {
		return nil, nil
	}

	artifact := &Artifact{
		EcloudImages:   state.Get("ecloudimages").(map[string]string),
		BuilderIdValue: BUILDER_ID,
		Client:         ecsClient,
		StateData:      map[string]interface{}{"generated_data": state.Get("generated_data")},
	}
	return artifact, nil
}
