package service

import (
	"rbac/models"
	"rbac/repository"
)

type SupportEngineerService struct {
	repo *repository.SupportEngineerRepository
}

func NewSupportEngineerService(
	repo *repository.SupportEngineerRepository,
) *SupportEngineerService {
	return &SupportEngineerService{repo: repo}
}

/*
=========================
 ADMIN: GET ALL SUPPORT ENGINEERS
=========================
*/
func (s *SupportEngineerService) GetAll() ([]models.SupportEngineer, error) {
	return s.repo.GetAll()
}

/*
=========================
 ADMIN: GET ONLY ACTIVE ENGINEERS
(THIS IS WHAT YOU SHOULD USE FOR ASSIGNMENT)
=========================
*/
func (s *SupportEngineerService) GetAllActive() ([]models.SupportEngineer, error) {
	return s.repo.GetAllActive()
}
