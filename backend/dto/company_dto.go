package dto

type CreateCompanyRequest struct {
	Name string `json:"name" binding:"required"`
}

type CompanyResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
