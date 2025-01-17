package statementdistribution

// //nolint
func createStatementDistribution() (*StatementDistribution, chan any) {

	subSystemToOverseer := make(chan any, 10)

	return &StatementDistribution{
		SubSystemToOverseer: subSystemToOverseer,
	}, subSystemToOverseer
}
