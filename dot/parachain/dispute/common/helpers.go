package common

// GetByzantineThreshold returns the byzantine threshold for the given number of validators.
func GetByzantineThreshold(n int) int {
	if n < 1 {
		return 0
	}
	return (n - 1) / 3
}

// GetSuperMajorityThreshold returns the super majority threshold for the given number of validators.
func GetSuperMajorityThreshold(n int) int {
	return n - GetByzantineThreshold(n)
}
