package config

type Config struct {
	RpcAllowedPrefix  []string
	RpcAllowedMethods []string
}

func Get() *Config {
	RpcAllowedPrefix := []string{"eth_"}
	RpcAllowedMethods := []string{"web3_clientVersion", "net_version"}

	return &Config{
		RpcAllowedPrefix:  RpcAllowedPrefix,
		RpcAllowedMethods: RpcAllowedMethods,
	}

}
