package ws

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"rpc-proxy/config"
	"rpc-proxy/models"
	"rpc-proxy/tools"
)

func Setup(upgrader websocket.Upgrader, cfg *config.Config, upstreamWebsocket string, stats tools.Request) echo.HandlerFunc {

	return func(ctx echo.Context) error {

		// Upgrade initial GET request to a websocket
		ws, err := upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
		if err != nil {
			return err
		}
		// Make sure we close the connection when the function returns
		defer func(ws *websocket.Conn) {
			err := ws.Close()
			if err != nil {
				fmt.Println(err)
			}
		}(ws)

		for {

			// Create a new WebSocket connection to the upstream server
			upstreamConn, _, err := websocket.DefaultDialer.Dial(upstreamWebsocket, nil)
			if err != nil {
				fmt.Println(err)
				return err
			}
			defer func(upstreamConn *websocket.Conn) {
				err := upstreamConn.Close()
				if err != nil {
					fmt.Println(err)
				}
			}(upstreamConn)

			// Listen for new messages from the client and forward them to the upstream server
			go func() {
				for {
					msgType, request, err := ws.ReadMessage()

					var rpcRequest models.GRPCRequest
					err = json.Unmarshal(request, &rpcRequest)
					if err != nil {
						errMsg, _ := json.Marshal(tools.CreateError(rpcRequest, -32600, "The JSON sent is not a valid RPC Request."))
						_ = ws.WriteMessage(websocket.TextMessage, errMsg)
						return
					}

					if cfg.IsAllowedMethod(rpcRequest.Method) {
						stats.Add(rpcRequest.Method)
						if err != nil {
							fmt.Println(err)
							return
						}
						err = upstreamConn.WriteMessage(msgType, request)
						if err != nil {
							fmt.Println(err)
							return
						}
					} else {
						errMsg, _ := json.Marshal(tools.CreateError(rpcRequest, -32601, fmt.Sprintf("the method %s does not exist/is not available", rpcRequest.Method)))
						err := ws.WriteMessage(websocket.TextMessage, errMsg)
						if err != nil {
							return
						}
					}

				}
			}()

			for {
				_, msg, err := upstreamConn.ReadMessage()
				if err != nil {
					fmt.Println(err)
					return err
				}
				err = ws.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					fmt.Println(err)
					return err
				}
			}

		}
	}
}
