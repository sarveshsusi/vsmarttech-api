package dto

type CreateCustomerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`

	Company string `json:"company" binding:"required"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}
