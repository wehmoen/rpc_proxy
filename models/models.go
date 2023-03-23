package models

type GRPCRequest struct {
	Jsonrpc string        `json:"jsonrpc,omitempty" validate:"required"`
	Method  string        `json:"method,omitempty" validate:"required"`
	Params  []interface{} `json:"params,omitempty" validate:"required"`
	Id      interface{}   `json:"id" validate:"required"`
}

type RPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Id      interface{} `json:"id"`
}

type WhitelistInnerError struct {
	Code    int64  `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type WhitelistError struct {
	Jsonrpc string              `json:"jsonrpc,omitempty"`
	Id      interface{}         `json:"id,omitempty"`
	Error   WhitelistInnerError `json:"error"`
}
