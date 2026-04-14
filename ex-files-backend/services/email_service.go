package services

type ResendEmailService struct {
}

func NewResendEmailService() *ResendEmailService {
	return &ResendEmailService{}
}

func (r *ResendEmailService) Send(to, subject, body string) error {
	return nil
}
