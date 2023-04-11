package commands

import (
	"context"

	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/domain"
	bankAccountErrors "github.com/faozimipa/go-cqrs-es/internal/bankAccount/errors"
	"github.com/faozimipa/go-cqrs-es/pkg/es"
	"github.com/faozimipa/go-cqrs-es/pkg/logger"
	"github.com/faozimipa/go-cqrs-es/pkg/tracing"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

type CreateBankAccountCommand struct {
	AggregateID string `json:"id" validate:"required,gte=0"`
	Email       string `json:"email" validate:"required,gte=0,email"`
	Address     string `json:"address" validate:"required,gte=0"`
	FirstName   string `json:"firstName" validate:"required,gte=0"`
	LastName    string `json:"lastName" validate:"required,gte=0"`
	Balance     int64  `json:"balance" validate:"gte=0"`
	Status      string `json:"status"`
}

type CreateBankAccount interface {
	Handle(ctx context.Context, cmd CreateBankAccountCommand) error
}

type createBankAccountCmdHandler struct {
	log            logger.Logger
	aggregateStore es.AggregateStore
}

func NewCreateBankAccountCmdHandler(log logger.Logger, aggregateStore es.AggregateStore) *createBankAccountCmdHandler {
	return &createBankAccountCmdHandler{log: log, aggregateStore: aggregateStore}
}

func (c *createBankAccountCmdHandler) Handle(ctx context.Context, cmd CreateBankAccountCommand) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "createBankAccountCmdHandler.Handle")
	defer span.Finish()
	span.LogFields(log.Object("command", cmd))

	exists, err := c.aggregateStore.Exists(ctx, cmd.AggregateID)
	if err != nil {
		return tracing.TraceWithErr(span, err)
	}
	if exists {
		return tracing.TraceWithErr(span, bankAccountErrors.ErrBankAccountAlreadyExists)
	}

	bankAccountAggregate := domain.NewBankAccountAggregate(cmd.AggregateID)
	err = bankAccountAggregate.CreateNewBankAccount(ctx, cmd.Email, cmd.Address, cmd.FirstName, cmd.LastName, cmd.Status, cmd.Balance)
	if err != nil {
		return tracing.TraceWithErr(span, err)
	}

	return c.aggregateStore.Save(ctx, bankAccountAggregate)
}
