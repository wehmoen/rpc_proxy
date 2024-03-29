package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-playground/validator"
	"github.com/go-resty/resty/v2"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	cmap "github.com/orcaman/concurrent-map"
	"io"
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
	trackingApiKey := flag.String("tracking-api-key", "5b5dca2d-76ee-4d76-8c44-406ce059371f", "Tracking API Key")
	flag.Parse()

	httpStats := cmap.New()
	wsStats := cmap.New()
	disallowedStats := cmap.New()

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

	var tracking *tools.SkyMavisTracking

	if *trackingApiKey != "" {
		tracking = tools.NewSkyMavisTracking(*trackingApiKey)
	}

	e.Use(middleware.Logger())

	e.Validator = InitValidator()

	e.GET("/", func(c echo.Context) error {

		var jsonValue models.RPCResponse

		request := models.GRPCRequest{
			Jsonrpc: "2.0",
			Method:  "web3_clientVersion",
			Params:  []interface{}{},
		}

		httpStats.Upsert("web3_clientVersion", 1, func(exist bool, valueInMap interface{}, newValue interface{}) interface{} {
			if exist {
				return valueInMap.(int) + 1
			}
			return newValue
		})

		_, _ = client.R().
			SetBody(request).
			SetResult(&jsonValue).
			Post(*upstreamRPC)

		_, _ = tracking.TrackAPIRequest(c.RealIP(), "/", nil)

		return c.JSON(http.StatusOK, map[string]interface{}{
			"allowed_prefixes": cfg.RpcAllowedPrefix,
			"allowed_methods":  cfg.RpcAllowedMethods,
			"websocketEnabled": cfg.HasWebsocket,
			"graphqlEnabled":   cfg.HasGraphQL,
			"nodeVersion":      jsonValue.Result,
		})
	})

	if cfg.HasWebsocket {

		var upgrader = websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		e.GET("/ws", ws.Setup(upgrader, cfg, *upstreamWebsocket, wsStats, tracking))
	}

	e.GET("/method_stats", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"http":       httpStats,
			"ws":         wsStats,
			"disallowed": disallowedStats,
		})
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
		isBatchRequest := false
		var request models.GRPCRequest
		var batchRequest []models.GRPCRequest
		var jsonValue models.RPCResponse
		var jsonBatchValue []models.RPCResponse

		var bodyBytes []byte
		if ctx.Request().Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(ctx.Request().Body)
			if err != nil {
				return ctx.JSON(http.StatusInternalServerError, tools.CreateError(request, -32600, "Failed to read request body."))
			}
		}

		err := json.Unmarshal(bodyBytes, &request)

		if err != nil {
			spew.Dump(err)
			err = json.Unmarshal(bodyBytes, &batchRequest)

			if err != nil {
				return ctx.JSON(http.StatusBadRequest, tools.CreateError(request, -32600, "The JSON sent is not a valid RPC Request."))
			}

			isBatchRequest = true
		}

		security := ctx.Request().Header.Get("x-kong-security") == RpcKongSecurityKey

		if !security && RpcKongSecurityKey != "" {
			return ctx.JSON(http.StatusUnauthorized, tools.CreateError(request, -0, http.StatusText(http.StatusUnauthorized)))
		}

		if isBatchRequest {
			if len(batchRequest) > 100 {
				return ctx.JSON(http.StatusBadRequest, tools.CreateError(request, -32600, "Batch request too long > 100."))
			}

			isValid := true
			for _, req := range batchRequest {
				if !cfg.IsAllowedMethod(req.Method) {
					isValid = false
					break
				}
				httpStats.Upsert(req.Method, 1, func(exist bool, valueInMap interface{}, newValue interface{}) interface{} {
					if exist {
						return valueInMap.(int) + 1
					}
					return newValue
				})
			}

			if !isValid {
				return ctx.JSON(http.StatusBadRequest, tools.CreateError(request, -32601, fmt.Sprintf("The batch request contains an invalid method.")))
			}
		}

		if !isBatchRequest && !cfg.IsAllowedMethod(request.Method) {
			hitButUnallowedMethods[request.Method]++
			disallowedStats.Upsert(request.Method, 1, func(exist bool, valueInMap interface{}, newValue interface{}) interface{} {
				if exist {
					return valueInMap.(int) + 1
				}
				return newValue
			})
			return ctx.JSON(http.StatusBadRequest, tools.CreateError(request, -32601, fmt.Sprintf("The method %s does not exist or is not available.", request.Method)))
		}

		if isBatchRequest {

			for _, req := range batchRequest {
				properties := map[string]string{
					"method":   req.Method,
					"rpc_type": "http",
				}

				_, _ = tracking.TrackAPIRequest(ctx.RealIP(), req.Method, properties)
			}

			_, err = client.R().
				SetBody(batchRequest).
				SetResult(&jsonBatchValue).
				Post(*upstreamRPC)
		} else {
			httpStats.Upsert(request.Method, 1, func(exist bool, valueInMap interface{}, newValue interface{}) interface{} {
				if exist {
					return valueInMap.(int) + 1
				}
				return newValue
			})
			_, err = client.R().
				SetBody(request).
				SetResult(&jsonValue).
				Post(*upstreamRPC)

			properties := map[string]string{
				"method":   request.Method,
				"rpc_type": "http",
			}

			_, _ = tracking.TrackAPIRequest(ctx.RealIP(), request.Method, properties)
		}

		if err != nil {
			return ctx.JSON(http.StatusInternalServerError, tools.CreateError(request, -32603, "Upstream error: "+err.Error()))
		}

		if isBatchRequest {
			return ctx.JSON(http.StatusOK, jsonBatchValue)
		} else {
			return ctx.JSON(http.StatusOK, jsonValue)
		}
	})

	err := e.Start(*httpListen)
	if err != nil {
		println(err)
	}
}
