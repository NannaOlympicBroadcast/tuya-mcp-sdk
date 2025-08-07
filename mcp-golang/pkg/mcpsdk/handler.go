package mcpsdk

import (
	"encoding/json"
	"io"
	"mcp-sdk/pkg/entity"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

type MCPSdkHandle interface {
	HandleError() func(session *Session, err error)
	HandleConnect() func(session *Session) error
	HandleMessageBinary(sdk *MCPSdk) func(session *Session, message []byte)
	HandlePong() func(session *Session) error
	HandleDisconnect(sdk *MCPSdk) func(session *Session) error
	HandleClose() func(session *Session, code int, text string) error
}

type MCPSdkHandler struct {
}

func NewMCPSdkHandler() *MCPSdkHandler {
	return &MCPSdkHandler{}
}

func (h *MCPSdkHandler) HandleError() func(session *Session, err error) {
	return func(session *Session, err error) {
		println("------------ Debug Handle Error ---------------")
		if err == io.EOF {
			println("[Warn::HandleError] connection is closed")
			return
		}
		println(err.Error())
	}
}

func (h *MCPSdkHandler) HandleConnect() func(session *Session) error {
	return func(session *Session) error {
		println("----------- Debug Handle Connect --------------")
		return nil
	}
}

func (h *MCPSdkHandler) HandleMessageBinary(sdk *MCPSdk) func(session *Session, message []byte) {
	return func(session *Session, message []byte) {
		println("----------- Debug Receive Message -------------")
		println(string(message))

		req := entity.MCPSdkRequest{}
		if err := json.Unmarshal(message, &req); err != nil {
			println("[Error::HandleMessageBinary] failed to unmarshal message: %v", err)
			return
		}

		ok, err := req.DoVerify(sdk.GetAuthToken())
		if err != nil {
			println("[Error::HandleMessageBinary] failed to verify message: %v", err)
			return
		}
		if !ok {
			println("[Error::HandleMessageBinary] sign failed, invalid message")
			return
		}

		replyMessage := ""
		switch mcpgo.MCPMethod(req.Method) {
		case mcpgo.MethodToolsList:
			listToolsReq := mcpgo.ListToolsRequest{}
			if err := json.Unmarshal([]byte(req.Request), &listToolsReq); err != nil {
				println("[Error::HandleMessageBinary] failed to unmarshal list tools request: %v", err)
				return
			}

			tools, err := sdk.GetMCPClient().ListTools(listToolsReq)
			if err != nil {
				println("[Error::HandleMessageBinary] failed to list tools: %v", err)
				return
			}

			listToolsResp, err := json.Marshal(tools)
			if err != nil {
				println("[Error::HandleMessageBinary] failed to marshal list tools response: %v", err)
				return
			}

			mcpSdkResp := entity.MCPSdkResponse{
				MCPSdkBaseMsg: req.MCPSdkBaseMsg,
				Response:      string(listToolsResp),
			}

			if err := mcpSdkResp.DoSign(sdk.GetAuthToken()); err != nil {
				println("[Error::HandleMessageBinary] failed to sign list tools response: %v", err)
				return
			}

			replyMessage = mcpSdkResp.String()

		case mcpgo.MethodToolsCall:
			callToolReq := mcpgo.CallToolRequest{}
			if err := json.Unmarshal([]byte(req.Request), &callToolReq); err != nil {
				println("[Error::HandleMessageBinary] failed to unmarshal call tool request: %v", err)
				return
			}

			callToolResp, err := sdk.GetMCPClient().CallTool(callToolReq)
			if err != nil {
				println("[Error::HandleMessageBinary] failed to call tool: %v", err.Error())
				replyError(&req, session, err.Error(), sdk.GetAuthToken())
				return
			}

			callToolRespJson, err := json.Marshal(callToolResp)
			if err != nil {
				println("[Error::HandleMessageBinary] failed to marshal call tool response: %v", err)
				replyError(&req, session, err.Error(), sdk.GetAuthToken())
				return
			}

			mcpSdkResp := entity.MCPSdkResponse{
				MCPSdkBaseMsg: req.MCPSdkBaseMsg,
				Response:      string(callToolRespJson),
			}

			if err := mcpSdkResp.DoSign(sdk.GetAuthToken()); err != nil {
				println("[Error::HandleMessageBinary] failed to sign call tool response: %v", err)
				replyError(&req, session, err.Error(), sdk.GetAuthToken())
				return
			}
			replyMessage = mcpSdkResp.String()

		case mcpgo.MCPMethod("root/kickout"):
			sdk.sendEvent(EventTypeKickout)
			println("------- Debug HandleMessageBinary Kickout --------")
			return

		case mcpgo.MCPMethod("root/migrate"):
			sdk.sendEvent(EventTypeMigrate)
			println("------- Debug HandleMessageBinary Migrate --------")
			return

		default:
			println("unknown method: ", req.Method)
			return
		}
		session.WriteBinary([]byte(replyMessage))
	}
}

func replyError(req *entity.MCPSdkRequest, session *Session, text string, token string) {
	callToolResp := mcpgo.CallToolResult{
		IsError: true,
		Content: []mcpgo.Content{
			mcpgo.TextContent{
				Type: "text",
				Text: text,
			},
		},
	}

	callToolRespJson, err := json.Marshal(callToolResp)
	if err != nil {
		println("[Error::HandleMessageBinary] failed to marshal call tool response: %v", err)
		return
	}
	mcpSdkResp := entity.MCPSdkResponse{
		MCPSdkBaseMsg: req.MCPSdkBaseMsg,
		Response:      string(callToolRespJson),
	}
	mcpSdkResp.DoSign(token)
	session.WriteBinary([]byte(mcpSdkResp.String()))
}

func (h *MCPSdkHandler) HandlePong() func(session *Session) error {
	return func(session *Session) error {
		println("------------- Debug HandlePong ----------------")
		return nil
	}
}

func (h *MCPSdkHandler) HandleDisconnect(sdk *MCPSdk) func(session *Session) error {
	return func(session *Session) error {
		sdk.sendEvent(EventTypeDisconnect)
		println("---------- Debug HandleDisconnect -------------")
		return nil
	}
}

func (h *MCPSdkHandler) HandleClose() func(session *Session, code int, text string) error {
	return func(session *Session, code int, text string) error {
		println("------------- Debug HandleClose ---------------")
		println("code: ", code)
		println("text: ", text)
		return nil
	}
}
