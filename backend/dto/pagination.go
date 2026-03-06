package dto

type PaginationQuery struct {
	Page int `form:"page"`
	Limit int `form:"limit"`
}
