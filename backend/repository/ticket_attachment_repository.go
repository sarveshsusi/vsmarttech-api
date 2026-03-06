package repository

import (
	"rbac/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TicketAttachmentRepository struct {
	db *gorm.DB
}

func NewTicketAttachmentRepository(db *gorm.DB) *TicketAttachmentRepository {
	return &TicketAttachmentRepository{db: db}
}

func (r *TicketAttachmentRepository) Create(
	ticketID uuid.UUID,
	fileURL string,
	fileName string,
	fileType string,
	uploadedBy uuid.UUID,
) error {

	return r.db.Create(&models.TicketAttachment{
		TicketID:   ticketID,
		FileURL:    fileURL,
		FileName:   fileName,
		FileType:   fileType,
		UploadedBy: uploadedBy,
	}).Error
}
