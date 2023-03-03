package config

import "strings"

type Config struct {
	RpcAllowedPrefix      []string
	RpcAllowedMethods     []string
	RpcBlacklistedMethods []string
	HasGraphQL            bool
}

func (c *Config) IsAllowedMethod(method string) bool {

	for _, blacklistedMethod := range c.RpcBlacklistedMethods {
		if method == blacklistedMethod {
			return false
		}
	}

	for _, allowedMethod := range c.RpcAllowedMethods {
		if method == allowedMethod {
			return true
		}
	}

	for _, allowedPrefix := range c.RpcAllowedPrefix {
		if strings.HasPrefix(method, allowedPrefix) {
			return true
		}
	}

	return false
}

func Get(hasGraphQL bool) *Config {
	RpcAllowedPrefix := []string{"eth_"}
	RpcAllowedMethods := []string{"web3_clientVersion", "net_version", "debug_traceTransaction", "debug_dumpBlock", "debug_traceBlock"}
	RpcBlacklistedMethods := []string{"eth_sendTransaction", "eth_accounts", "eth_sign", "eth_signTransaction", "eth_getWork", "eth_submitWork", "eth_submitHashrate"}

	return &Config{
		RpcAllowedPrefix:      RpcAllowedPrefix,
		RpcAllowedMethods:     RpcAllowedMethods,
		RpcBlacklistedMethods: RpcBlacklistedMethods,
		HasGraphQL:            hasGraphQL,
	}

}
