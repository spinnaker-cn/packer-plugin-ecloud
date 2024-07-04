package basic

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep"
	ecs "gitlab.ecloud.com/ecloud/ecloudsdkecs"
	"gitlab.ecloud.com/ecloud/ecloudsdkecs/model"
	"time"
)

type stepRunInstance struct {
	SpecsName       string
	BillingType     string
	UserData        string
	ZoneId          string
	InstanceName    string
	VmType          string
	Cpu             int32
	Ram             int32
	PublicIpType    string
	BandwidthSize   int32
	ChargeMode      string
	BootVolumnSize  int32
	BootVolumnType  string
	DataVolumns     []ecloudDataVolumn
	publicIpAddress string
	instanceId      string
}

func (s *stepRunInstance) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {

	Say(state, "Trying to create a new instance", "")

	client := state.Get("ecs_client").(*ecs.Client)
	config := state.Get("config").(*Config)
	network_id := state.Get("network_id").(string)
	image_name := state.Get("source_image_name").(string)
	security_group_id := state.Get("security_group_id").(string)
	temp_key_pair := state.Get("temporary_key_pair_name").(string)

	password := config.Comm.SSHPassword
	if password == "" && config.Comm.WinRMPassword != "" {
		password = config.Comm.WinRMPassword
	}

	billingType := s.BillingType
	if billingType == "" {
		billingType = "HOUR"
	}

	network := &model.VmCreateRequestNetworks{
		NetworkId: &network_id,
	}

	bootVolumnType := model.VmCreateRequestBootVolumeVolumeTypeEnum(s.BootVolumnType)
	bootVolumn := &model.VmCreateRequestBootVolume{
		Size:       &s.BootVolumnSize,
		VolumeType: &bootVolumnType,
	}

	// config RunInstances parameters
	createReq := &model.VmCreateRequest{}
	reqBody := &model.VmCreateBody{}
	reqBody.Name = &s.InstanceName
	reqBody.Region = &s.ZoneId
	reqBody.BillingType = (*model.VmCreateBodyBillingTypeEnum)(&billingType)
	reqBody.VmType = (*model.VmCreateBodyVmTypeEnum)(&s.VmType)
	reqBody.Cpu = &s.Cpu
	reqBody.Ram = &s.Ram
	reqBody.SpecsName = &s.SpecsName
	reqBody.ImageName = &image_name
	quantity := int32(1)
	reqBody.Quantity = &quantity
	reqBody.UserData = &s.UserData
	reqBody.Networks = network
	reqBody.BootVolume = bootVolumn
	if len(s.DataVolumns) > 0 {
		dataVolume := make([]model.VmCreateRequestDataVolume, 0)
		for _, disk := range s.DataVolumns {
			var dataDisk model.VmCreateRequestDataVolume
			dataDisk.Size = &disk.Size
			dataDisk.IsShare = &disk.IsShare
			dataDisk.ResourceType = (*model.VmCreateRequestDataVolumeResourceTypeEnum)(&disk.ResourceType)
			dataVolume = append(dataVolume, dataDisk)
		}
		reqBody.DataVolume = &dataVolume
	}
	reqBody.SecurityGroupIds = []string{security_group_id}
	if password != "" {
		reqBody.Password = &password
	}
	if config.Comm.SSHKeyPairName != "" {
		reqBody.KeypairName = &config.Comm.SSHKeyPairName
	}
	if reqBody.KeypairName == nil {
		reqBody.KeypairName = &temp_key_pair
	}
	if s.publicIpAddress != "" {
		bind := &model.VmCreateRequestBind{
			PublicIp: &model.VmCreateRequestPublicIp{
				Address: &s.publicIpAddress,
			},
		}
		reqBody.Bind = bind
	}
	if s.PublicIpType != "" {
		ipType := model.VmCreateRequestIpIpTypeEnum(s.PublicIpType)
		ip := &model.VmCreateRequestIp{
			IpType: &ipType,
		}
		chargeMode := model.VmCreateRequestBandwidthChargeModeEnum(s.ChargeMode)
		bandwidth := &model.VmCreateRequestBandwidth{
			BandwidthSize: &s.BandwidthSize,
			ChargeMode:    &chargeMode,
		}
		reqBody.Ip = ip
		reqBody.Bandwidth = bandwidth
	}

	createReq.VmCreateBody = reqBody
	var createRsp *model.VmCreateResponse
	err := Retry(ctx, func(ctx context.Context) error {
		var e error
		createRsp, e = client.VmCreate(createReq)
		return e
	})
	if err != nil {
		return Halt(state, err, "Failed to run instance")
	}
	if createRsp.ErrorMessage != nil {
		return Halt(state, fmt.Errorf(*createRsp.ErrorMessage), "Failed to run instance")
	}
	orderId := createRsp.Body.OrderId
	if orderId == nil {
		return Halt(state, fmt.Errorf("No orderId return"), "Failed to run instance")
	}
	Message(state, *orderId, "Waiting Instance ready at Order")

	instanceBody, ecsErr := WaitForActiveInstanceByOrder(ctx, client, *orderId, 1800)
	if ecsErr != nil {
		return Halt(state, ecsErr, "Failed to wait for instance ready")
	}
	s.instanceId = *instanceBody.Id
	state.Put("instance", instanceBody)
	state.Put("instance_id", s.instanceId)
	Message(state, s.instanceId, "Instance created")

	return multistep.ActionContinue
}

func (s *stepRunInstance) Cleanup(state multistep.StateBag) {
	if s.instanceId == "" {
		return
	}

	ctx := context.TODO()
	client := state.Get("ecs_client").(*ecs.Client)

	if _, ok := state.GetOk("temporary_key_pair_name"); ok {
		temp_key_pair := state.Get("temporary_key_pair_name").(string)
		Say(state, temp_key_pair, "Trying to detach key pair")
		req := &model.VmUnbindKeypairRequest{
			VmUnbindKeypairBody: &model.VmUnbindKeypairBody{
				ServerId: &s.instanceId,
				KeyName:  &temp_key_pair,
			},
		}
		var resp *model.VmUnbindKeypairResponse
		err := Retry(ctx, func(ctx context.Context) error {
			var e error
			resp, e = client.VmUnbindKeypair(req)
			return e
		})
		if err != nil || resp.ErrorMessage != nil {
			Say(state, "Fail to detach keypair from instance, Key Pair may not be able to be cleaned later", "")
		} else {
			Say(state, temp_key_pair, "Detach Key Pair")
			// wait for 5 second to complete detach
			time.Sleep(5 * time.Second)
		}
	}

	SayClean(state, "instance:"+s.instanceId)
	trueValue := true
	req := &model.VmDeleteRequest{
		VmDeletePath: &model.VmDeletePath{
			ServerId: &s.instanceId,
		},
		VmDeleteQuery: &model.VmDeleteQuery{
			DataVolumeDelete: &trueValue,
		},
	}
	var deleteRsp *model.VmDeleteResponse
	err := Retry(ctx, func(ctx context.Context) error {
		var e error
		deleteRsp, e = client.VmDelete(req)
		return e
	})
	if err != nil {
		Error(state, err, fmt.Sprintf("Failed to terminate instance(%s), please delete it manually", s.instanceId))
	}
	if deleteRsp.ErrorMessage != nil {
		Error(state, fmt.Errorf(*deleteRsp.ErrorMessage), fmt.Sprintf("Failed to terminate instance(%s), please delete it manually", s.instanceId))
	}
}

func WaitForActiveInstanceByOrder(ctx context.Context, client *ecs.Client, orderId string, timeout int) (*model.VmGetServerDetailResponseBody, error) {
	req := &model.VmgetOrderInfoByOrderIdRequest{
		VmgetOrderInfoByOrderIdQuery: &model.VmgetOrderInfoByOrderIdQuery{
			OrderId: &orderId,
		},
	}
	needDetailed := true
	for {
		var resp *model.VmgetOrderInfoByOrderIdResponse
		err := Retry(ctx, func(ctx context.Context) error {
			var e error
			resp, e = client.VmgetOrderInfoByOrderId(req)
			return e
		})
		if err != nil {
			return nil, fmt.Errorf("Failed to get order info")
		}
		if resp.ErrorMessage != nil {
			return nil, fmt.Errorf(*resp.ErrorMessage)
		}
		if resp.Body != nil && len(*resp.Body) > 0 {
			// it is possible that response body is empty or instanceId is empty for the first few queries
			for _, one := range *resp.Body {
				if *one.InstanceId != "" {
					serverReq := &model.VmGetServerDetailRequest{
						VmGetServerDetailPath: &model.VmGetServerDetailPath{
							ServerId: one.InstanceId,
						},
						VmGetServerDetailQuery: &model.VmGetServerDetailQuery{
							Detail: &needDetailed,
						},
					}
					serverResp, e := client.VmGetServerDetail(serverReq)
					if e == nil && serverResp != nil && serverResp.Body != nil && serverResp.Body.Id != nil {
						// the right instanceId
						for {
							e = Retry(ctx, func(ctx context.Context) error {
								serverResp, e = client.VmGetServerDetail(serverReq)
								return e
							})
							if e != nil {
								return nil, e
							}
							if resp.ErrorMessage != nil {
								return nil, fmt.Errorf(*resp.ErrorMessage)
							}
							instanceStatus := *serverResp.Body.Status
							if instanceStatus == 1 {
								return serverResp.Body, nil
							}
							time.Sleep(DefaultWaitForInterval * time.Second)
							timeout = timeout - DefaultWaitForInterval
							if timeout <= 0 {
								return nil, fmt.Errorf("wait instance ready(%s) timeout", *one.InstanceId)
							}
						}
					}
				}
			}
		}
		time.Sleep(DefaultWaitForInterval * time.Second)
		timeout = timeout - DefaultWaitForInterval
		if timeout <= 0 {
			return nil, fmt.Errorf("wait order ready(%s) timeout", orderId)
		}
	}
}
