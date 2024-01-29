package grpcservice

import (
	"authentication-service/app"
	acProtobuf "authentication-service/proto/v1/pb/account"
	authProtobuf "authentication-service/proto/v1/pb/authentication"
	"context"
	"errors"

	"github.com/golang-jwt/jwt"
	"github.com/sirupsen/logrus"
)

type AuthServer struct {
	authProtobuf.AuthServiceServer
	App *app.App
}

func (s *AuthServer) ValidateUser(ctx context.Context, req *authProtobuf.ValidateUserRequest) (*authProtobuf.ValidateUserReply, error) {

	if req.Token == "" {
		return nil, errors.New("token is not present")
	}

	if req.ProfileID <= 0 {
		return nil, errors.New("profile is not present")
	}

	claims, err := s.App.JwtService.FetchJWTToken(req.Token)
	if err != nil {
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, errors.New("invalid jwt token")
		}

		logrus.Error("Error in jwt token", err)
		return nil, errors.New("invalid jwt token")
	}

	if claims.UserID <= 0 {
		logrus.Error("Error in jwt claims")
		return nil, errors.New("invalid jwt token")
	}

	cr := acProtobuf.AccountDetailRequest{
		AccountId: int32(claims.UserID),
	}

	crRes, err := s.App.Repos.AccountServiceClient.GetAccountDetails(context.TODO(), &cr)
	if err != nil {
		return nil, errors.New("invalid account")
	}

	if crRes.Status != 1 {
		return nil, errors.New("invalid account")
	}

	vr := acProtobuf.ValidateProfileRequest{
		ProfileId: int32(req.ProfileID),
		AccountId: int32(claims.UserID),
	}

	vrs, err := s.App.Repos.AccountServiceClient.ValidateProfile(context.TODO(), &vr)
	if err != nil {
		logrus.Error("Error in validate profile ", err)
		return nil, err
	}

	if vrs.Status != 1 {
		return nil, errors.New("profile is invalid")
	}

	return &authProtobuf.ValidateUserReply{
		Data:    crRes.Data,
		Status:  1,
		Message: "User verified.",
	}, nil
}
