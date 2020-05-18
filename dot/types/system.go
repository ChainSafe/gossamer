package types

// SystemInfo struct to hold system related information
type SystemInfo struct {
	SystemName       string
	SystemVersion    string
	NodeName         string
	SystemProperties systemProperties
}

type systemProperties struct {
	Ss58Format    int
	TokenDecimals int
	TokenSymbol   string
}
