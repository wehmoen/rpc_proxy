package models

type GRPCRequest struct {
	Jsonrpc string        `json:"jsonrpc,omitempty" validate:"required"`
	Method  string        `json:"method,omitempty" validate:"required"`
	Params  []interface{} `json:"params,omitempty" validate:"required"`
	Id      int64         `json:"id" validate:"required"`
}

type RPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Id      int64       `json:"id"`
}

type WhitelistInnerError struct {
	Code    int64  `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type WhitelistError struct {
	Jsonrpc string              `json:"jsonrpc,omitempty"`
	Id      int64               `json:"id,omitempty"`
	Error   WhitelistInnerError `json:"error"`
}
