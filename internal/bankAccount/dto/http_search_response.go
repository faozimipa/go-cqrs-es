package dto

import (
	"github.com/faozimipa/go-cqrs-es/pkg/utils"
)

type HttpSearchResponse struct {
	List               []*HttpBankAccountResponse `json:"list"`
	PaginationResponse *utils.PaginationResponse  `json:"paginationResponse"`
}
