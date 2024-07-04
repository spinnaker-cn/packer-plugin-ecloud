package basic

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep"
	vpc "gitlab.ecloud.com/ecloud/ecloudsdkvpc"
	"gitlab.ecloud.com/ecloud/ecloudsdkvpc/model"
)

type stepConfigVPC struct {
	VpcId       string
	CidrBlock   string
	VpcName     string
	isCreate    bool
	Region      string
	NetworkName string
}

func (s *stepConfigVPC) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	vpcClient := state.Get("vpc_client").(*vpc.Client)
	if len(s.VpcId) != 0 {
		Say(state, s.VpcId, "Trying to use existing vpc")
		qPath := model.NewGetVpcDetailRespPathBuilder().VpcId(s.VpcId).Build()
		req := model.NewGetVpcDetailRespRequestBuilder().GetVpcDetailRespPath(qPath).Build()
		var resp *model.GetVpcDetailRespResponse
		err := Retry(ctx, func(ctx context.Context) error {
			var e error
			resp, e = vpcClient.GetVpcDetailResp(req)
			return e
		})
		if err != nil {
			return Halt(state, err, "Failed to get vpc info")
		}
		if resp.ErrorMessage != nil {
			return Halt(state, fmt.Errorf(*resp.ErrorMessage), "Failed to get vpc info")
		}
		if resp.Body != nil {
			s.isCreate = false
			state.Put("vpc_id", *resp.Body.Id)
			state.Put("route_id", *resp.Body.RouterId)
			Message(state, *resp.Body.Name, "Vpc found")
			return multistep.ActionContinue
		}
	}
	//bodyBuilder := model.NewVpcOrderBodyBuilder().
	//	Cidr(s.CidrBlock).
	//	Name(s.VpcName).
	//	NetworkName(s.NetworkName).
	//	Region(s.Region).
	//	Specs(model.VpcOrderBodySpecsEnumNormal).
	//	Build()
	//req := model.NewVpcOrderRequestBuilder().
	//	VpcOrderBody(bodyBuilder).
	//	Build()
	//var resp *model.VpcOrderResponse
	//err := Retry(ctx, func(ctx context.Context) error {
	//	var e error
	//	resp, e = vpcClient.VpcOrder(req)
	//	return e
	//})
	//if err != nil {
	//	return Halt(state, err, "Failed to create vpc")
	//}
	//
	//orderId := resp.Body.OrderId
	//s.isCreate = true
	//s.VpcId = *resp.Vpc.VpcId
	//state.Put("vpc_id", s.VpcId)
	//Message(state, s.VpcId, "Vpc created")
	return Halt(state, fmt.Errorf("The specified vpc(%s) does not exist", s.VpcId), "")
}

func (s *stepConfigVPC) Cleanup(state multistep.StateBag) {
	if !s.isCreate {
		return
	}

	ctx := context.TODO()
	vpcClient := state.Get("vpc_client").(*vpc.Client)
	routeId := state.Get("route_id").(string)
	SayClean(state, "vpc")

	bodyBuilder := model.NewCommonMopOrderDeleteVpcBodyBuilder().
		ResourceId(routeId).
		ProductType("router")
	body := bodyBuilder.Build()
	requestBuilder := model.NewCommonMopOrderDeleteVpcRequestBuilder().
		CommonMopOrderDeleteVpcBody(body)
	req := requestBuilder.Build()

	err := Retry(ctx, func(ctx context.Context) error {
		_, e := vpcClient.CommonMopOrderDeleteVpc(req)
		return e
	})
	if err != nil {
		Error(state, err, fmt.Sprintf("Failed to delete vpc(%s), please delete it manually", s.VpcId))
	}
}
