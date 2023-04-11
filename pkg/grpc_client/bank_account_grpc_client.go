package grpc_client

import (
	"context"

	"github.com/faozimipa/go-cqrs-es/pkg/interceptors"
	bankAccountService "github.com/faozimipa/go-cqrs-es/proto/bank_account"
)

func NewBankAccountGrpcClient(ctx context.Context, port string, im interceptors.InterceptorManager) (bankAccountService.BankAccountServiceClient, func() error, error) {
	grpcServiceConn, err := NewGrpcServiceConn(ctx, port, im)
	if err != nil {
		return nil, nil, err
	}

	serviceClient := bankAccountService.NewBankAccountServiceClient(grpcServiceConn)

	return serviceClient, grpcServiceConn.Close, nil
}
