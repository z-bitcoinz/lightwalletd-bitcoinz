// BitcoinZ Network Parameters Documentation
// 
// This file contains all the network-specific parameters that need to be
// modified in lightwalletd to support BitcoinZ instead of BitcoinZ.

package common

const (
	// Network Magic Numbers (from bitcore-lib-btcz)
	BitcoinZMainnetMagic = 0x24e92764
	BitcoinZTestnetMagic = 0x2783734e
	
	// Network Ports
	BitcoinZMainnetP2PPort  = 1989
	BitcoinZTestnetP2PPort  = 11989
	BitcoinZMainnetRPCPort  = 1979
	BitcoinZTestnetRPCPort  = 11979
	
	// Address Version Bytes (from bitcore-lib-btcz)
	BitcoinZPubKeyHashMainnet = 0x1cb8  // Produces addresses starting with 't1'
	BitcoinZPubKeyHashTestnet = 0x1d25  // Produces addresses starting with 'tm'
	BitcoinZScriptHashMainnet = 0x1cbd  
	BitcoinZScriptHashTestnet = 0x1cba
	BitcoinZPrivateKey        = 0x80
	
	// Shielded Address Prefixes
	BitcoinZZaddrMainnet = 0x169a  // Produces addresses starting with 'zc'
	BitcoinZZaddrTestnet = 0x16b6  // Produces addresses starting with 'zt'
	BitcoinZZkeyMainnet  = 0xab36
	BitcoinZZkeyTestnet  = 0xac08
	
	// Extended Key Prefixes
	BitcoinZXPubKeyMainnet  = 0x0488b21e
	BitcoinZXPrivKeyMainnet = 0x0488ade4
	BitcoinZXPubKeyTestnet  = 0x043587cf
	BitcoinZXPrivKeyTestnet = 0x04358394
	
	// Protocol Parameters
	// TODO: Verify these values with BitcoinZ team
	BitcoinZSaplingActivationMainnet = 1 // Need actual BitcoinZ sapling height
	BitcoinZSaplingActivationTestnet = 1 // Need actual BitcoinZ testnet sapling height
	
	// Chain Names
	BitcoinZChainMainnet = "bitcoinz"
	BitcoinZChainTestnet = "testnet"
	BitcoinZChainRegtest = "regtest"
	
	// Node Names
	BitcoinZNodeName = "bitcoinzd"
	
	// Address Lengths
	BitcoinZTransparentAddrLen = 35  // 't1' or 'tm' + 33 chars
	BitcoinZShieldedAddrLen    = 95  // 'zc' or 'zt' + 93 chars
)

// BitcoinZ Address Format:
// Transparent Mainnet: t1[33 alphanumeric characters] (35 total)
// Transparent Testnet: tm[33 alphanumeric characters] (35 total)
// Shielded Mainnet:    zc[93 alphanumeric characters] (95 total)
// Shielded Testnet:    zt[93 alphanumeric characters] (95 total)

// Key Differences from BitcoinZ:
// 1. Different network magic numbers
// 2. Different RPC/P2P ports
// 3. Different address prefixes (but same length structure)
// 4. Different chain names ("bitcoinz" vs "main")
// 5. Different sapling activation heights (TODO: verify exact heights)
