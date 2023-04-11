package events

import (
	"github.com/Rhymond/go-money"
	"github.com/faozimipa/go-cqrs-es/pkg/es"
)

const (
	BankAccountCreatedEventType es.EventType = "BANK_ACCOUNT_CREATED_V1"
)

type BankAccountCreatedEventV1 struct {
	Email     string       `json:"email"`
	Address   string       `json:"address"`
	FirstName string       `json:"firstName"`
	LastName  string       `json:"lastName"`
	Balance   *money.Money `json:"balance"`
	Status    string       `json:"status"`
	Metadata  []byte       `json:"-"`
}
