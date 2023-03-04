package main

import (
	"flag"
	"fmt"
	"github.com/go-playground/validator"
	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"rpc-proxy/config"
	"rpc-proxy/models"
	"rpc-proxy/tools"
	"rpc-proxy/ws"
)

type Validator struct {
	Validator *validator.Validate
}

func (v *Validator) Validate(i interface{}) error {
	if err := v.Validator.Struct(i); err != nil {
		return err
	}
	return nil
}

func InitValidator() *Validator {
	return &Validator{Validator: validator.New()}
}

func main() {

	hasGraphql := flag.Bool("graphql", false, "Enable GraphQL")
	hasWebsocket := flag.Bool("websocket", false, "Enable Websocket")
	upstreamRPC := flag.String("upstream", "http://127.0.0.1:8545", "Upstream RPC Host")
	upstreamWebsocket := flag.String("upstream-ws", "ws://127.0.0.1:8546", "Upstream Websocket Host")
	httpListen := flag.String("http-listen", "127.0.0.1:9898", "HTTP Port")
	flag.Parse()

	cfg := config.Get(*hasGraphql, *hasWebsocket)

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

	e.Validator = InitValidator()

	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"info":             "Ronin RPC Proxy",
			"allowed_prefixes": cfg.RpcAllowedPrefix,
			"allowed_methods":  cfg.RpcAllowedMethods,
		})
	})

	if cfg.HasWebsocket {

		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		e.GET("/ws", ws.Setup(upgrader, cfg, *upstreamWebsocket))
	}

	e.GET("/method_stats", func(c echo.Context) error {
		return c.JSON(http.StatusOK, hitButUnallowedMethods)
	})

	if cfg.HasGraphQL {

		graphQlUIUrl, _ := url.Parse(*upstreamRPC)
		uiProxy := httputil.NewSingleHostReverseProxy(graphQlUIUrl)

		uiProxy.Director = func(req *http.Request) {
			req.URL.Scheme = graphQlUIUrl.Scheme
			req.URL.Host = graphQlUIUrl.Host
			req.URL.Path = "/graphql/ui"
		}

		e.GET("/graphql/ui", func(c echo.Context) error {
			uiProxy.ServeHTTP(c.Response(), c.Request())
			return nil
		})

		graphProxy := httputil.NewSingleHostReverseProxy(graphQlUIUrl)

		graphProxy.Director = func(req *http.Request) {
			req.URL.Scheme = graphQlUIUrl.Scheme
			req.URL.Host = graphQlUIUrl.Host
			req.URL.Path = "/graphql"
		}

		e.POST("/graphql", func(c echo.Context) error {
			graphProxy.ServeHTTP(c.Response(), c.Request())
			return nil
		})

	}

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

		if cfg.IsAllowedMethod(request.Method) {

			var jsonValue models.RPCResponse

			resp, err := client.R().
				SetBody(request).
				SetResult(&jsonValue).
				Post(*upstreamRPC)

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

	err := e.Start(*httpListen)
	if err != nil {
		println(err)
	}
}
