package mappers

import (
	"github.com/Rhymond/go-money"
	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/domain"
	bankAccountService "github.com/faozimipa/go-cqrs-es/proto/bank_account"
)

func BalanceFromMoney(money *money.Money) domain.Balance {
	return domain.Balance{
		Amount:   money.AsMajorUnits(),
		Currency: money.Currency().Code,
	}
}

func BalanceToGrpc(balance domain.Balance) *bankAccountService.Balance {
	return &bankAccountService.Balance{
		Amount:   balance.Amount,
		Currency: balance.Currency,
	}
}

func BalanceMoneyToGrpc(money *money.Money) *bankAccountService.Balance {
	return &bankAccountService.Balance{
		Amount:   money.AsMajorUnits(),
		Currency: money.Currency().Code,
	}
}
