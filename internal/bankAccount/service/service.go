package service

import (
	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/commands"
	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/domain"
	"github.com/faozimipa/go-cqrs-es/internal/bankAccount/queries"
	"github.com/faozimipa/go-cqrs-es/pkg/es"
	"github.com/faozimipa/go-cqrs-es/pkg/logger"
)

type BankAccountService struct {
	Commands *commands.BankAccountCommands
	Queries  *queries.BankAccountQueries
}

func NewBankAccountService(
	log logger.Logger,
	aggregateStore es.AggregateStore,
	mongoRepository domain.MongoRepository,
	elasticSearchRepository domain.ElasticSearchRepository,
) *BankAccountService {

	bankAccountCommands := commands.NewBankAccountCommands(
		commands.NewChangeEmailCmdHandler(log, aggregateStore),
		commands.NewDepositBalanceCmdHandler(log, aggregateStore),
		commands.NewCreateBankAccountCmdHandler(log, aggregateStore),
		commands.NewWithdrawBalanceCommandHandler(log, aggregateStore),
	)

	newBankAccountQueries := queries.NewBankAccountQueries(
		queries.NewGetBankAccountByIDQuery(log, aggregateStore, mongoRepository),
		queries.NewSearchBankAccountsQuery(log, aggregateStore, elasticSearchRepository),
	)

	return &BankAccountService{Commands: bankAccountCommands, Queries: newBankAccountQueries}
}
