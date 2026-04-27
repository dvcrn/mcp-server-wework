package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

const protocolVersion = "2025-06-18"

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     func(context.Context, json.RawMessage) (any, error)
}

type Server struct {
	name    string
	version string
	tools   map[string]Tool
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

func NewServer(name, version string) *Server {
	return &Server{name: name, version: version, tools: map[string]Tool{}}
}

func (s *Server) AddTool(tool Tool) {
	s.tools[tool.Name] = tool
}

func (s *Server) Handle(ctx context.Context, line []byte) ([][]byte, error) {
	trimmed := json.RawMessage(line)
	if len(trimmed) == 0 {
		return nil, nil
	}

	var batch []json.RawMessage
	if err := json.Unmarshal(trimmed, &batch); err == nil {
		responses := make([]response, 0, len(batch))
		for _, item := range batch {
			resp, ok := s.handleOne(ctx, item)
			if ok {
				responses = append(responses, resp)
			}
		}
		if len(responses) == 0 {
			return nil, nil
		}
		encoded, err := json.Marshal(responses)
		if err != nil {
			return nil, err
		}
		return [][]byte{encoded}, nil
	}

	resp, ok := s.handleOne(ctx, trimmed)
	if !ok {
		return nil, nil
	}
	encoded, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return [][]byte{encoded}, nil
}

func (s *Server) handleOne(ctx context.Context, raw json.RawMessage) (response, bool) {
	var req request
	if err := json.Unmarshal(raw, &req); err != nil {
		return response{JSONRPC: "2.0", Error: &responseError{Code: -32700, Message: "parse error"}}, true
	}

	if len(req.ID) == 0 {
		s.handleNotification(ctx, req)
		return response{}, false
	}

	switch req.Method {
	case "initialize":
		return response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": protocolVersion,
				"capabilities": map[string]any{
					"tools": map[string]any{
						"listChanged": false,
					},
				},
				"serverInfo": map[string]any{
					"name":    s.name,
					"version": s.version,
				},
			},
		}, true
	case "ping":
		return response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}, true
	case "tools/list":
		tools := make([]map[string]any, 0, len(s.tools))
		for _, tool := range s.tools {
			tools = append(tools, map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
			})
		}
		return response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": tools}}, true
	case "tools/call":
		var params toolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return response{JSONRPC: "2.0", ID: req.ID, Error: &responseError{Code: -32602, Message: "invalid tool params"}}, true
		}
		tool, ok := s.tools[params.Name]
		if !ok {
			return response{JSONRPC: "2.0", ID: req.ID, Error: &responseError{Code: -32601, Message: "unknown tool: " + params.Name}}, true
		}
		args := params.Arguments
		if len(args) == 0 {
			args = json.RawMessage(`{}`)
		}
		result, err := tool.Handler(ctx, args)
		if err != nil {
			return response{JSONRPC: "2.0", ID: req.ID, Result: toolErrorResult(err)}, true
		}
		return response{JSONRPC: "2.0", ID: req.ID, Result: toolSuccessResult(result)}, true
	default:
		return response{JSONRPC: "2.0", ID: req.ID, Error: &responseError{Code: -32601, Message: "method not found"}}, true
	}
}

func (s *Server) handleNotification(ctx context.Context, req request) {
	_ = ctx
	_ = req
}

func toolSuccessResult(result any) map[string]any {
	text := marshalPretty(result)
	return map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": text,
		}},
		"structuredContent": result,
		"isError":           false,
	}
}

func toolErrorResult(err error) map[string]any {
	return map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": err.Error(),
		}},
		"isError": true,
	}
}

func marshalPretty(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
