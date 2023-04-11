package events

import "github.com/faozimipa/go-cqrs-es/pkg/es"

const (
	BalanceDepositedEventType es.EventType = "BALANCE_DEPOSITED_V1"
)

type BalanceDepositedEventV1 struct {
	Amount    int64  `json:"amount"`
	PaymentID string `json:"paymentID"`
	Metadata  []byte `json:"-"`
}
