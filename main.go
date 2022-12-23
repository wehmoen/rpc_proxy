package main

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"os"
	"rpc-proxy/models"
	"rpc-proxy/tools"
	"strings"
)

func main() {
	rpcAllowedPrefix := []string{"eth_"}
	rpcAllowedMethods := []string{"web3_clientVersion", "net_version"}

	hitButUnallowedMethods := map[string]int{}

	RpcKongSecurityKey := os.Getenv("RPC_KONG_SECURITY_KEY")

	if RpcKongSecurityKey != "" {
		fmt.Println(fmt.Sprintf("RPC_KONG_SECURITY_KEY is set to: %s", RpcKongSecurityKey))
	} else {
		fmt.Println("RPC_KONG_SECURITY_KEY is not set")
	}

	e := echo.New()

	client := resty.New()

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
			return ctx.JSON(http.StatusOK, tools.CreateError(request, -32600, "The JSON sent is not a valid RPC Request."))
		}

		security := ctx.Request().Header.Get("x-kong-security") == RpcKongSecurityKey

		if security == false && RpcKongSecurityKey != "" {
			return ctx.JSON(http.StatusUnauthorized, tools.CreateError(request, -0, http.StatusText(http.StatusUnauthorized)))
		}

		if strings.HasPrefix(request.Method, "eth_") == true || tools.Contains(rpcAllowedMethods, request.Method) == true {

			var jsonValue models.RPCResponse

			resp, err := client.R().
				SetBody(request).
				SetResult(&jsonValue).
				Post("http://127.0.0.1:8545")

			if err != nil {
				return ctx.JSON(http.StatusInternalServerError, tools.CreateError(request, -32603, "Upstream error: "+err.Error()))
			}

			if resp.StatusCode() != http.StatusOK {
				return ctx.JSON(http.StatusInternalServerError, tools.CreateError(request, -32603, "Internal error: "+resp.Status()))
			}

			return ctx.JSON(http.StatusOK, jsonValue)
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
