package grpcservice

import (
	"authentication-service/app"
	authProtobuf "authentication-service/proto/v1/authentication"
	"context"
)

type AuthServer struct {
	authProtobuf.AuthServiceServer
	App *app.App
}

func (s *AuthServer) ValidateUser(ctx context.Context, req *authProtobuf.ValidateUserRequest) (*authProtobuf.ValidateUserReply, error) {

	// claims, err := s.App.JwtService.FetchJWTToken(req.Token)
	// if err != nil {
	// 	return nil, err
	// }

	// cr := acProtobuf.CachedAccountRequest{
	// 	AccountId: int32(claims.UserID),
	// }

	// crRes, err := s.App.Repos.AccountServiceClient.GetCachedAccount(context.TODO(), &cr)
	// if err != nil {
	// 	return nil, err
	// }

	return nil, nil
}
