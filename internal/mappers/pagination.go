package mappers

import (
	"github.com/faozimipa/go-cqrs-es/pkg/utils"
	bankAccountService "github.com/faozimipa/go-cqrs-es/proto/bank_account"
)

func PaginationFromProto(pagination *bankAccountService.Pagination) *utils.Pagination {
	return &utils.Pagination{
		Size: int(pagination.GetSize()),
		Page: int(pagination.GetPage()),
	}
}

func PaginationResponseToProto(response *utils.PaginationResponse) *bankAccountService.Pagination {
	return &bankAccountService.Pagination{
		TotalCount: response.TotalCount,
		TotalPages: response.TotalPages,
		Page:       response.Page,
		Size:       response.Size,
		HasMore:    response.HasMore,
	}
}
