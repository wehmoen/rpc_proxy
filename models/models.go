package models

type GRPCRequest struct {
	Jsonrpc string        `json:"jsonrpc,omitempty"`
	Method  string        `json:"method,omitempty"`
	Params  []interface{} `json:"params,omitempty"`
	Id      int64         `json:"id"`
}

type RPCResponse struct {
	Jsonrpc string      `json:"jsonrpc,omitempty"`
	Result  interface{} `json:"result,omitempty"`
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
