package common

import (
	"strings"
)

// TransformAddressForValidation converts BitcoinZ addresses to BitcoinZ format for validation
// This is a compatibility layer until we update the regex
func TransformAddressForValidation(addr string) string {
	// BitcoinZ t1 addresses -> t addresses (remove the '1')
	if strings.HasPrefix(addr, "t1") && len(addr) == 35 {
		return "t" + addr[2:]
	}
	// BitcoinZ tm addresses -> t addresses (remove the 'm')  
	if strings.HasPrefix(addr, "tm") && len(addr) == 35 {
		return "t" + addr[2:]
	}
	return addr
}

// GetChainName returns BitcoinZ-specific chain names
func GetChainName(chain string) string {
	switch chain {
	case "main":
		return "bitcoinz"
	case "test":
		return "testnet"
	default:
		return chain
	}
}
