package etherscan

import (
	"context"
	"errors"
	"net/http"
)

const (
	accountModule = "account"
)

var errNoTxFound = errors.New("No transactions found")

type AccountService Service

func (s *AccountService) ListInternalTxs(
	ctx context.Context,
	query TransactionQueryParams) ([]Transaction, error) {
	query.Action = "txlistinternal"
	var res []Transaction
	if err := s.req.request(
		ctx,
		http.MethodGet,
		nil,
		accountModule,
		query,
		nil,
		&res); nil != err {
		if err.Error() == errNoTxFound.Error() {
			return []Transaction{}, nil
		}
		return nil, err
	}

	return res, nil
}
