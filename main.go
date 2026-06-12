package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type ActionInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Params      map[string]any `json:"params,omitempty"`
}

type Request struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}

type Response struct {
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Result  map[string]any `json:"result"`
}

var pluginMeta = map[string]any{
	"name":        "AutoCDNfly",
	"description": "监控证书状态自动更新部署，此插件适配“失控的防御系统(scdn.io)”的cdnfly系统，不确保cdnfly系统标准接口可用。",
	"version":     "1.0.0",
	"author":      "初春网络",
	"config": map[string]any{
		"base_url":   "基础API接口",
		"api_key":    "api_key",
		"api_secret": "api_secret",
	},
	"actions": []ActionInfo{
		{
			Name:        "monitor",
			Description: "监控证书状态自动更新部署",
			Params:      map[string]any{},
		},
	},
}

func outputJSON(resp *Response) {
	_ = json.NewEncoder(os.Stdout).Encode(resp)
}

func outputError(msg string, err error) {
	outputJSON(&Response{
		Status:  "error",
		Message: fmt.Sprintf("%s: %v", msg, err),
	})
}

func main() {
	var req Request
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		outputError("读取输入失败", err)
		return
	}

	if err := json.Unmarshal(input, &req); err != nil {
		outputError("解析请求失败", err)
		return
	}

	switch req.Action {
	case "get_metadata":
		outputJSON(&Response{
			Status:  "success",
			Message: "插件信息",
			Result:  pluginMeta,
		})
	case "list_actions":
		outputJSON(&Response{
			Status:  "success",
			Message: "支持的动作",
			Result:  map[string]any{"actions": pluginMeta["actions"]},
		})
	case "monitor":
		rep, err := Monitor(req.Params)
		if err != nil {
			outputError("CDN 监控失败", err)
			return
		}
		outputJSON(rep)

	default:
		outputJSON(&Response{
			Status:  "error",
			Message: "未知 action: " + req.Action,
		})
	}
}
