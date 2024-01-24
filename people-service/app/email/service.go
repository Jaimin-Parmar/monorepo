package email

import "people-service/model"

type Service interface {
	SendEmail(email model.Email) error
}

type service struct {
}

func NewService() Service {
	return &service{}
}

func (s *service) SendEmail(email model.Email) error {
	return sendEmail(email)
}
