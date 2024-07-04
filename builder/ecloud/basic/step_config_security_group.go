package basic

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep"
	vpc "gitlab.ecloud.com/ecloud/ecloudsdkvpc"
	"gitlab.ecloud.com/ecloud/ecloudsdkvpc/model"
)

type stepConfigSecurityGroup struct {
	SecurityGroupId   string
	SecurityGroupName string
	Description       string
	isCreate          bool
}

func (s *stepConfigSecurityGroup) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("vpc_client").(*vpc.Client)
	config := state.Get("config").(*Config)

	if len(s.SecurityGroupId) != 0 {
		Say(state, s.SecurityGroupId, "Trying to use existing securitygroup")
		request := model.ListSecurityGroupRespRequest{
			ListSecurityGroupRespQuery: &model.ListSecurityGroupRespQuery{
				SecurityGroupIds: []string{config.SecurityGroupId},
			},
		}
		var response *model.ListSecurityGroupRespResponse
		err := Retry(ctx, func(ctx context.Context) error {
			var e error
			response, e = client.ListSecurityGroupResp(&request)
			return e
		})
		if err != nil {
			return Halt(state, err, "Failed to get securitygroup info")
		}
		if response == nil || response.Body == nil {
			return Halt(state, fmt.Errorf("received nil response or nil body"), "")
		}
		if response.Body.Total != nil && *response.Body.Total > 0 && response.Body.Content != nil {
			s.isCreate = false
			state.Put("security_group_id", s.SecurityGroupId)
			Message(state, *(*response.Body.Content)[0].Name, "Securitygroup found")
			return multistep.ActionContinue
		}

	}

	return Halt(state, fmt.Errorf("The specified securitygroup(%s) does not exists", s.SecurityGroupId), "")

	//Say(state, "Trying to create a new securitygroup", "")

	//req := model.NewCreateSecurityGroupRequestBuilder().Build()
	//req.CreateSecurityGroupBody = &model.CreateSecurityGroupBody{Name: &s.SecurityGroupName, Description: &s.Description}
	//
	//var response *model.CreateSecurityGroupResponse
	//err := Retry(ctx, func(ctx context.Context) error {
	//	var e error
	//	response, e = client.CreateSecurityGroup(req)
	//	return e
	//})
	//if err != nil {
	//	return Halt(state, err, "Failed to create securitygroup")
	//}
	//
	//s.isCreate = true
	//s.SecurityGroupId = *response.OpenApiReturnValue
	//state.Put("security_group_id", s.SecurityGroupId)
	//Message(state, s.SecurityGroupId, "Securitygroup created")
	//
	//// bind securitygroup ingress police
	//Say(state, "Trying to create securitygroup polices", "")
	//pReq := model.CreateSecurityGroupRuleRequest{}
	//securityGroupRuleBody := model.CreateSecurityGroupRuleBody{}
	//securityGroupRuleBody.SetDescription("ingress").SetProtocol("protocol").SetRemoteType(model.CreateSecurityGroupRuleBodyRemoteTypeEnumCidr)
	//securityGroupRuleBody.SetRemoteIpPrefix("0.0.0.0/0")
	//securityGroupRuleBody.SecurityGroupId = &s.SecurityGroupId
	//pReq.SetCreateSecurityGroupRuleBody(&securityGroupRuleBody)
	//perr := Retry(ctx, func(ctx context.Context) error {
	//	_, e := client.CreateSecurityGroupRule(&pReq)
	//	return e
	//})
	//if perr != nil {
	//	return Halt(state, perr, "Failed to create securitygroup ingress polices")
	//}
	//
	//// bind securitygroup engress police
	//oReq := model.CreateSecurityGroupRuleRequest{}
	//securityGroupRuleBody2 := &model.CreateSecurityGroupRuleBody{}
	//securityGroupRuleBody2.SetDescription("egress").SetProtocol("protocol").SetRemoteType(model.CreateSecurityGroupRuleBodyRemoteTypeEnumCidr)
	//securityGroupRuleBody2.SetRemoteIpPrefix("0.0.0.0/0")
	//securityGroupRuleBody2.SecurityGroupId = &s.SecurityGroupId
	//oReq.SetCreateSecurityGroupRuleBody(securityGroupRuleBody2)
	//oErr := Retry(ctx, func(ctx context.Context) error {
	//	_, e := client.CreateSecurityGroupRule(&oReq)
	//	return e
	//})
	//if oErr != nil {
	//	return Halt(state, oErr, "Failed to create securitygroup egress polices")
	//}
	//
	//Message(state, "Securitygroup polices created", "")
	//
	//return multistep.ActionContinue
}

func (s *stepConfigSecurityGroup) Cleanup(state multistep.StateBag) {
	if !s.isCreate {
		return
	}
	ctx := context.TODO()
	vpcClient := state.Get("vpc_client").(*vpc.Client)
	SayClean(state, "securitygroup")
	req := model.DeleteSecurityGroupRequest{}
	deletePath := model.DeleteSecurityGroupPath{SecurityGroupId: &s.SecurityGroupId}
	req.SetDeleteSecurityGroupPath(&deletePath)

	err := Retry(ctx, func(ctx context.Context) error {
		var e error
		_, e = vpcClient.DeleteSecurityGroup(&req)
		return e
	})

	if err != nil {
		Error(state, err, fmt.Sprintf("Failed to delete securitygroup(%s), please delete it manually", s.SecurityGroupId))
	}
}
