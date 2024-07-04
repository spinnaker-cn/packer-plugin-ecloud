package basic

import (
	"context"
	"fmt"
	"github.com/hashicorp/packer/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer/packer-plugin-sdk/multistep"
	ecs "gitlab.ecloud.com/ecloud/ecloudsdkecs"
	"gitlab.ecloud.com/ecloud/ecloudsdkecs/model"
	"os"
	"runtime"
)

type stepConfigKeyPair struct {
	Debug        bool
	Comm         *communicator.Config
	DebugKeyPath string
	keyID        string
}

func (s *stepConfigKeyPair) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	client := state.Get("ecs_client").(*ecs.Client)

	if s.Comm.SSHAgentAuth {
		if s.Comm.SSHKeyPairName == "" {
			Say(state, "Using SSH agent with key pair in source image", "")
			return multistep.ActionContinue
		}
	}
	if s.Comm.SSHKeyPairName != "" {
		queryReq := &model.VmGetKeyPairDetailRequest{
			VmGetKeyPairDetailPath: &model.VmGetKeyPairDetailPath{
				KeypairName: &s.Comm.SSHKeyPairName,
			},
		}
		var queryRsp *model.VmGetKeyPairDetailResponse
		err := Retry(ctx, func(ctx context.Context) error {
			var e error
			queryRsp, e = client.VmGetKeyPairDetail(queryReq)
			return e
		})
		if err != nil {
			return Halt(state, err, "Invalid KeyPairName")
		}
		if queryRsp.ErrorMessage != nil {
			return Halt(state, fmt.Errorf(*queryRsp.ErrorMessage), "Invalid KeyPairName")
		}
		Say(state, fmt.Sprintf("Using SSH agent with exists key pair(%s)", s.Comm.SSHKeyPairName), "")
		return multistep.ActionContinue
	}
	if s.Comm.SSHTemporaryKeyPairName == "" {
		Say(state, "Not to use temporary keypair", "")
		s.Comm.SSHKeyPairName = ""
		return multistep.ActionContinue
	}

	Say(state, s.Comm.SSHTemporaryKeyPairName, "Trying to create a new keypair")

	apiVersion := "v2.1"
	req := &model.VmCreateKeypairRequest{
		VmCreateKeypairBody: &model.VmCreateKeypairBody{
			Name:       &s.Comm.SSHTemporaryKeyPairName,
			ApiVersion: &apiVersion,
		},
	}
	var resp *model.VmCreateKeypairResponse
	err := Retry(ctx, func(ctx context.Context) error {
		var e error
		resp, e = client.VmCreateKeypair(req)
		return e
	})
	if err != nil {
		return Halt(state, err, "Failed to create keypair")
	}
	if resp.ErrorMessage != nil {
		return Halt(state, fmt.Errorf(*resp.ErrorMessage), "Failed to create keypair")
	}

	bodyMap, ok := resp.Body.(map[string]interface{})
	if ok {
		if keyId, ok := bodyMap["id"]; ok {
			s.keyID = keyId.(string)
		}
		if publicKey, ok := bodyMap["publicKey"]; ok {
			s.Comm.SSHPublicKey = []byte(publicKey.(string))
		}
		if privateKey, ok := bodyMap["privateKey"]; ok {
			s.Comm.SSHPrivateKey = []byte(privateKey.(string))
		}
	}
	state.Put("temporary_key_pair_name", s.Comm.SSHTemporaryKeyPairName)

	if s.Debug {
		Message(state, fmt.Sprintf("Saving temporary key to %s for debug purposes", s.DebugKeyPath), "")
		f, err := os.Create(s.DebugKeyPath)
		if err != nil {
			return Halt(state, err, "Failed to saving debug key file")
		}
		defer f.Close()
		if _, err := f.Write(s.Comm.SSHPrivateKey); err != nil {
			return Halt(state, err, "Failed to writing debug key file")
		}
		if runtime.GOOS != "windows" {
			if err := f.Chmod(0600); err != nil {
				return Halt(state, err, "Failed to chmod debug key file")
			}
		}
	}

	Message(state, s.Comm.SSHTemporaryKeyPairName, "Keypair created")

	return multistep.ActionContinue
}

func (s *stepConfigKeyPair) Cleanup(state multistep.StateBag) {
	if s.Comm.SSHPrivateKeyFile != "" || (s.Comm.SSHKeyPairName == "" && s.keyID == "") {
		return
	}

	ctx := context.TODO()
	client := state.Get("ecs_client").(*ecs.Client)

	SayClean(state, "keypair")

	req := &model.VmDeleteKeyPairRequest{
		VmDeleteKeyPairPath: &model.VmDeleteKeyPairPath{
			Id: &s.keyID,
		},
	}
	err := Retry(ctx, func(ctx context.Context) error {
		_, e := client.VmDeleteKeyPair(req)
		return e
	})
	if err != nil {
		Error(state, err, fmt.Sprintf("Failed to delete keypair(%s), please delete it manually", s.keyID))
	}

	if s.Debug {
		if err := os.Remove(s.DebugKeyPath); err != nil {
			Error(state, err, fmt.Sprintf("Failed to delete debug key file(%s), please delete it manually", s.DebugKeyPath))
		}
	}
	Message(state, s.keyID, "Key Pair Cleaned")

}
