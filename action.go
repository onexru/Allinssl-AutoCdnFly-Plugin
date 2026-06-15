package main

import (
	"fmt"
	"scdn_push_ssl/utils/cdnfly"
	"scdn_push_ssl/utils/domain"
)

func Monitor(cfg map[string]any) (*Response, error) {
	// 校验参数
	if cfg == nil {
		return nil, fmt.Errorf("config不能为空")
	}
	certStr, ok := cfg["cert"].(string)
	if !ok || certStr == "" {
		return nil, fmt.Errorf("cert不能为空")
	}
	keyStr, ok := cfg["key"].(string)
	if !ok || keyStr == "" {
		return nil, fmt.Errorf("key不能为空")
	}
	baseUrlStr, ok := cfg["base_url"].(string)
	if !ok || baseUrlStr == "" {
		return nil, fmt.Errorf("base_url不能为空")
	}
	apiKeyStr, ok := cfg["api_key"].(string)
	if !ok || apiKeyStr == "" {
		return nil, fmt.Errorf("api_key不能为空")
	}
	apiSecretStr, ok := cfg["api_secret"].(string)
	if !ok || apiSecretStr == "" {
		return nil, fmt.Errorf("api_secret不能为空")
	}

	// 解析证书
	certInfo, domains, err := domain.ParseCertInfo(certStr)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}
	_ = certInfo

	// 初始化CDN配置
	config := &cdnfly.Config{
		BaseURL:   baseUrlStr,
		ApiKey:    apiKeyStr,
		ApiSecret: apiSecretStr,
	}

	// 获取待操作站点列表
	needList, err := cdnfly.GetNeedUpdateCerts(config)
	if err != nil {
		return nil, fmt.Errorf("获取需要更新的证书列表失败: %w", err)
	}

	if len(needList) == 0 {
		return &Response{
			Status:  "success",
			Message: "没有需要更新的证书",
		}, nil
	}

	// 过滤出域名匹配的站点
	matchList := domain.CheckCertDomains(domains, needList)
	if len(matchList) == 0 {
		return &Response{
			Status:  "success",
			Message: "没有匹配的证书和站点",
		}, nil
	}

	// 执行证书操作
	err = cdnfly.ExecuteAction(config, certStr, keyStr, matchList)
	if err != nil {
		return nil, fmt.Errorf("执行操作失败: %w", err)
	}

	return &Response{
		Status:  "success",
		Message: "操作成功",
	}, nil
}
