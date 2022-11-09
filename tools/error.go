package tools

import "rpc-proxy/models"

func CreateError(r models.GRPCRequest, c int, m string) models.WhitelistError {
	return models.WhitelistError{
		Jsonrpc: r.Jsonrpc,
		Id:      r.Id,
		Error: models.WhitelistInnerError{
			Code:    c,
			Message: m,
		},
	}
}
