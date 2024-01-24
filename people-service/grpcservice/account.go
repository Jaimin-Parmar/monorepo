package grpcservice

import (
	"context"
	"fmt"
	"people-service/app"
	"people-service/model"
	"people-service/protobuf"
	"time"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
)

type AccountServer struct {
	protobuf.AccountServiceServer
	App *app.App
}

func (s *AccountServer) AuthAccount(ctx context.Context, reqdata *protobuf.CredentialsRequest) (*protobuf.AccountReply, error) {
	fmt.Println("reqdata", reqdata)
	req := model.Credentials{}
	req.Email = reqdata.Email
	req.Password = reqdata.Password
	req.UserName = reqdata.UserName

	accountData, err := s.App.AccountService.AuthAccount(&req)
	if err != nil {
		return nil, err
	}

	thumbs := protobuf.Thumbnails{}
	thumbs.Small = accountData.Thumbs.Small
	thumbs.Medium = accountData.Thumbs.Medium
	thumbs.Large = accountData.Thumbs.Large
	thumbs.Icon = accountData.Thumbs.Icon
	thumbs.Original = accountData.Thumbs.Original

	return &protobuf.AccountReply{
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

func converTimeToTimestamp(timedata time.Time) *timestamp.Timestamp {
	return &timestamp.Timestamp{
		Seconds: timedata.Unix(),
		Nanos:   int32(timedata.Nanosecond()),
	}
}

func ConvertProtobufAccounts(aps []*model.AccountPermimssion) []*protobuf.AccountPermission {
	rap := make([]*protobuf.AccountPermission, 0)
	for _, obj := range aps {
		tmp := protobuf.AccountPermission{}
		tmp.AccountId = int32(obj.AccountID)
		tmp.Company = obj.Company
		tmp.HasApiAccess = obj.HasAPIAccess
		tmp.IsOwner = obj.IsOwner
		rap = append(rap, &tmp)
	}
	return rap
}
