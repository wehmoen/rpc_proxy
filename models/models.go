package models

type GRPCRequest struct {
	Jsonrpc string        `json:"jsonrpc,omitempty"`
	Method  string        `json:"method,omitempty"`
	Params  []interface{} `json:"params,omitempty"`
	Id      int           `json:"id,omitempty"`
}

type WhitelistInnerError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type WhitelistError struct {
	Jsonrpc string              `json:"jsonrpc,omitempty"`
	Id      int                 `json:"id,omitempty"`
	Error   WhitelistInnerError `json:"error"`
}
