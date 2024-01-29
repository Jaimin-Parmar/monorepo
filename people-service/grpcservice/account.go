package grpcservice

import (
	"context"
	"encoding/json"
	"fmt"
	"people-service/app"
	"people-service/model"
	acProtobuf "people-service/proto/v1/account"
	"time"

	"github.com/golang/protobuf/ptypes/any"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
)

type AccountServer struct {
	acProtobuf.AccountServiceServer
	App *app.App
}

func (s *AccountServer) AuthAccount(ctx context.Context, reqdata *acProtobuf.CredentialsRequest) (*acProtobuf.AccountReply, error) {
	fmt.Println("reqdata", reqdata)
	req := model.Credentials{}
	req.Email = reqdata.Email
	req.Password = reqdata.Password
	req.UserName = reqdata.UserName

	accountData, err := s.App.AccountService.AuthAccount(&req)
	if err != nil {
		return nil, err
	}

	thumbs := acProtobuf.Thumbnails{}
	thumbs.Small = accountData.Thumbs.Small
	thumbs.Medium = accountData.Thumbs.Medium
	thumbs.Large = accountData.Thumbs.Large
	thumbs.Icon = accountData.Thumbs.Icon
	thumbs.Original = accountData.Thumbs.Original

	return &acProtobuf.AccountReply{
		Id:               int32(accountData.ID),
		AccountType:      int32(accountData.AccountType),
		UserName:         accountData.UserName,
		FirstName:        accountData.FirstName,
		LastName:         accountData.LastName,
		Photo:            accountData.Photo,
		Thumbs:           &thumbs,
		Email:            accountData.Email,
		RecoveryEmail:    accountData.RecoveryEmail,
		Phone:            accountData.Phone,
		Password:         accountData.Password,
		CreateDate:       converTimeToTimestamp(accountData.CreateDate),
		LastModifiedDate: converTimeToTimestamp(accountData.LastModifiedDate),
		Token:            accountData.Token,
		Accounts:         ConvertProtobufAccounts(accountData.Accounts),
		IsActive:         accountData.IsActive,
		ResetStatus:      accountData.ResetStatus,
		ResetTime:        converTimeToTimestamp(accountData.ResetTime),
	}, nil
}

func (s *AccountServer) GetCachedAccount(ctx context.Context, reqdata *acProtobuf.CachedAccountRequest) (*acProtobuf.GenericReply, error) {
	accountData, err := s.App.AccountService.FetchCachedAccount(int(reqdata.AccountId))
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

func converTimeToTimestamp(timedata time.Time) *timestamp.Timestamp {
	return &timestamp.Timestamp{
		Seconds: timedata.Unix(),
		Nanos:   int32(timedata.Nanosecond()),
	}
}

func ConvertProtobufAccounts(aps []*model.AccountPermimssion) []*acProtobuf.AccountPermission {
	rap := make([]*acProtobuf.AccountPermission, 0)
	for _, obj := range aps {
		tmp := acProtobuf.AccountPermission{}
		tmp.AccountId = int32(obj.AccountID)
		tmp.Company = obj.Company
		tmp.HasApiAccess = obj.HasAPIAccess
		tmp.IsOwner = obj.IsOwner
		rap = append(rap, &tmp)
	}
	return rap
}

func convertDataToBytes(data interface{}) ([]byte, error) {
	// Implement the logic to serialize YourModelData to bytes
	// For example, you can use encoding/json, encoding/gob, or other serialization methods
	// Here, we'll use a simple JSON encoding for demonstration purposes
	return json.Marshal(data)
}
