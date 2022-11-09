package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io"
	"net/http"
	"rpc-proxy/models"
	"rpc-proxy/tools"
	"strings"
)

func main() {
	rpcAllowedPrefix := []string{"eth_"}
	rpcAllowedMethods := []string{"web3_clientVersion", "net_version"}

	hitButUnallowedMethods := map[string]int{}

	e := echo.New()

	e.Use(middleware.Logger())

	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"info":             "Ronin RPC Proxy",
			"allowed_prefixes": rpcAllowedPrefix,
			"allowed_methods":  rpcAllowedMethods,
		})
	})

	e.GET("/method_stats", func(c echo.Context) error {
		return c.JSON(http.StatusOK, hitButUnallowedMethods)
	})

	e.POST("/", func(ctx echo.Context) error {
		var request models.GRPCRequest

		err := ctx.Bind(&request)

		if err != nil {
			return ctx.JSON(http.StatusOK, tools.CreateError(request, -32600, "The JSON sent is not a valid Request object."))
		}

		if strings.HasPrefix(request.Method, "eth_") == true || tools.Contains(rpcAllowedMethods, request.Method) == true {
			jsonValue, _ := json.Marshal(request)
			res, _ := http.Post("http://127.0.0.1:8545", "application/json", bytes.NewBuffer(jsonValue))

			b, _ := io.ReadAll(res.Body)

			var proxyResult map[string]interface{}
			_ = json.Unmarshal(b, &proxyResult)

			return ctx.JSON(http.StatusOK, proxyResult)
		} else {
			hitButUnallowedMethods[request.Method] = hitButUnallowedMethods[request.Method] + 1
			return ctx.JSON(http.StatusOK, tools.CreateError(request, -32601, fmt.Sprintf("the method %s does not exist/is not available", request.Method)))
		}

	})

	err := e.Start("127.0.0.1:9898")
	if err != nil {
		println(err)
	}
}
