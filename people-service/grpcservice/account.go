package grpcservice

import (
	"context"
	"encoding/json"
	"people-service/app"
	"people-service/model"
	acProtobuf "people-service/proto/v1/pb/account"

	"github.com/golang/protobuf/ptypes/any"
)

type AccountServer struct {
	acProtobuf.AccountServiceServer
	App *app.App
}

// AuthAccount mehtod is verify account credentials.
func (s *AccountServer) AuthAccount(ctx context.Context, reqdata *acProtobuf.CredentialsRequest) (*acProtobuf.AccountReply, error) {
	req := model.Credentials{}
	req.Email = reqdata.Email
	req.Password = reqdata.Password
	req.UserName = reqdata.UserName

	accountData, err := s.App.AccountService.AuthAccount(&req)
	if err != nil {
		return nil, err
	}

	return &acProtobuf.AccountReply{
		Id:    int32(accountData.ID),
		Email: accountData.Email,
	}, nil
}

// GetAccountDetails is get account details based on accountID
func (s *AccountServer) GetAccountDetails(ctx context.Context, reqdata *acProtobuf.AccountDetailRequest) (*acProtobuf.GenericReply, error) {
	accountData, err := s.App.AccountService.FetchAccount(int(reqdata.AccountId), true)
	if err != nil {
		return nil, err
	}

	dataBytes, err := convertDataToBytes(accountData)
	if err != nil {
		return nil, err
	}

	dataAny := &any.Any{
		Value: dataBytes,
	}

	return &acProtobuf.GenericReply{
		Data:    dataAny,
		Status:  1,
		Message: "Account details",
	}, nil
}

// ValidateProfile is validate profile with account.
func (s *AccountServer) ValidateProfile(ctx context.Context, reqdata *acProtobuf.ValidateProfileRequest) (*acProtobuf.GenericReply, error) {
	err := s.App.ProfileService.ValidateProfile(int(reqdata.ProfileId), int(reqdata.AccountId))
	if err != nil {
		return nil, err
	}

	return &acProtobuf.GenericReply{
		Data:    nil,
		Status:  1,
		Message: "Profile details are verified",
	}, nil
}

func convertDataToBytes(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}
