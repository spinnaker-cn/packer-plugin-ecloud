package basic

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep"
	vpc "gitlab.ecloud.com/ecloud/ecloudsdkvpc"
	"gitlab.ecloud.com/ecloud/ecloudsdkvpc/model"
)

type stepConfigSubnet struct {
	NetworkName     string
	NetworkId       string
	SubnetId        string
	SubnetCidrBlock string
	SubnetName      string
	Zone            string
	Region          string
	isCreate        bool
}

func (s *stepConfigSubnet) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	vpcClient := state.Get("vpc_client").(*vpc.Client)

	if len(s.SubnetId) != 0 {
		Say(state, s.SubnetId, "Trying to use existing subnet")

		qPath := model.NewGetSubnetDetailRespPathBuilder().SubnetId(s.SubnetId).Build()
		req := model.NewGetSubnetDetailRespRequestBuilder().GetSubnetDetailRespPath(qPath).Build()

		var resp *model.GetSubnetDetailRespResponse
		err := Retry(ctx, func(ctx context.Context) error {
			var e error
			resp, e = vpcClient.GetSubnetDetailResp(req)
			return e
		})
		if err != nil {
			return Halt(state, err, "Failed to get subnet info")
		}
		if resp.ErrorMessage != nil {
			return Halt(state, fmt.Errorf(*resp.ErrorMessage), "Failed to get subnet info")
		}
		if resp.Body != nil {
			s.isCreate = false
			state.Put("subnet_id", *resp.Body.Id)
			state.Put("network_id", *resp.Body.NetworkId)
			state.Put("subnet_name", *resp.Body.Name)
			state.Put("networkType", *resp.Body.NetworkTypeEnum)
			Message(state, *resp.Body.Name, "Subnet found")
			return multistep.ActionContinue
		}
		return Halt(state, fmt.Errorf("The specified subnet(%s) does not exist", s.SubnetId), "")
	}

	//Say(state, "Trying to create a new subnet", "")
	//
	//routeId := state.Get("route_id").(string)
	//networkBodyBuilder := model.NewCreateNetworkBodyBuilder()
	//networkBody := networkBodyBuilder.
	//	AvailabilityZoneHints(s.Zone).
	//	RouterId(routeId).
	//	NetworkName(s.NetworkName).
	//	NetworkTypeEnum(model.CreateNetworkBodyNetworkTypeEnumEnumVm).
	//	Build()
	//
	//requestBuilder := model.NewCreateNetworkRequestBuilder()
	//req := requestBuilder.
	//	CreateNetworkBody(networkBody).
	//	Build()
	//var resp *model.CreateNetworkResponse
	//err := Retry(ctx, func(ctx context.Context) error {
	//	var e error
	//	resp, e = vpcClient.CreateNetwork(req)
	//	return e
	//})
	//if err != nil {
	//	return Halt(state, err, "Failed to create subnet")
	//}
	//
	//s.isCreate = true
	////s.SubnetId = *resp..Subnet.SubnetId
	////state.Put("subnet_id", s.SubnetId)
	//Message(state, s.SubnetId, "Subnet created")

	return multistep.ActionContinue
}

func (s *stepConfigSubnet) Cleanup(state multistep.StateBag) {
	if !s.isCreate {
		return
	}

	ctx := context.TODO()
	vpcClient := state.Get("vpc_client").(*vpc.Client)

	SayClean(state, "subnet")

	requestBuilder := model.NewDeleteNetworkRequestBuilder()
	pathBuilder := model.NewDeleteNetworkPathBuilder()
	networkPath := pathBuilder.NetworkId(s.NetworkId).Build()
	request := requestBuilder.DeleteNetworkPath(networkPath).Build()

	err := Retry(ctx, func(ctx context.Context) error {
		_, e := vpcClient.DeleteNetwork(request)
		return e
	})
	if err != nil {
		Error(state, err, fmt.Sprintf("Failed to delete subnet(%s), please delete it manually", s.SubnetId))
	}
}
