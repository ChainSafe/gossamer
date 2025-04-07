package runtime

type TransactionOutcome[R any] interface {
	isTransactionOutcome()
}

type (
	TransactionOutcomeCommit[R any] struct {
		Result R
	}
	TransactionOutcomeRollback[R any] struct {
		Result R
	}
)

func (TransactionOutcomeCommit[R]) isTransactionOutcome()   {}
func (TransactionOutcomeRollback[R]) isTransactionOutcome() {}
