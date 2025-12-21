package service

type AuditService struct {
}

func NewAuditService() *AuditService {
	return &AuditService{}
}

func (s *AuditService) CreateAudit() {}

func (s *AuditService) GetOneAudit() {}

func (s *AuditService) GetAllAudits() {}
