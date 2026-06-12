package cdnfly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	BaseURL   string
	ApiKey    string
	ApiSecret string
}

func GetAllSites(config *Config) ([]map[string]any, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	url := fmt.Sprintf("%s/sites", config.BaseURL)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("api-key", config.ApiKey)
	req.Header.Set("api-secret", config.ApiSecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求接口失败: %w", err)
	}
	defer resp.Body.Close()

	var respMap map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&respMap); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	code, _ := respMap["code"].(float64)
	if code != 0 {
		msg, _ := respMap["message"].(string)
		return nil, fmt.Errorf("接口返回错误: %s", msg)
	}

	dataArr, ok := respMap["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("data 数据格式异常")
	}

	var result []map[string]any
	for _, item := range dataArr {
		siteMap, ok := item.(map[string]any)
		if !ok {
			continue
		}

		hlStr, _ := siteMap["https_listen"].(string)
		hlStr = strings.TrimSpace(hlStr)
		var certID float64 = 0
		if hlStr != "" {
			var hlMap map[string]any
			_ = json.Unmarshal([]byte(hlStr), &hlMap)
			certID, _ = hlMap["cert"].(float64)
		}

		newMap := make(map[string]any)
		newMap["id"] = siteMap["id"]
		newMap["domain"] = siteMap["domain"]
		newMap["name"] = siteMap["name"]
		newMap["cert_id"] = certID

		result = append(result, newMap)
	}

	return result, nil
}

func GetAllCertificates(config *Config) ([]map[string]any, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	url := fmt.Sprintf("%s/certs", config.BaseURL)
	postBody := []byte(`{"page":1,"limit":1000}`)
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(postBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", config.ApiKey)
	req.Header.Set("api-secret", config.ApiSecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求接口失败: %w", err)
	}
	defer resp.Body.Close()

	var respMap map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&respMap); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	code, _ := respMap["code"].(float64)
	if code != 0 {
		msg, _ := respMap["message"].(string)
		return nil, fmt.Errorf("接口返回错误: %s", msg)
	}

	dataArr, ok := respMap["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("data 数据格式异常")
	}

	var result []map[string]any
	for _, item := range dataArr {
		rawMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		newMap := make(map[string]any)
		newMap["id"] = rawMap["id"]
		newMap["domain"] = rawMap["domain"]
		newMap["name"] = rawMap["name"]
		newMap["expire_time2"] = rawMap["expire_time2"]

		result = append(result, newMap)
	}

	return result, nil
}

func GetNeedUpdateCerts(config *Config) ([]map[string]any, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}
	certs, err := GetAllCertificates(config)
	if err != nil {
		return nil, fmt.Errorf("获取证书列表失败: %w", err)
	}
	domains, err := GetAllSites(config)
	if err != nil {
		return nil, fmt.Errorf("获取网站列表失败: %w", err)
	}

	var needList []map[string]any
	certMap := make(map[float64]map[string]any)
	for _, c := range certs {
		cid, ok := c["id"].(float64)
		if !ok {
			continue
		}
		certMap[cid] = c
	}

	now := time.Now()
	threshold := 30 * 24 * time.Hour

	for _, site := range domains {
		siteID, _ := site["id"].(float64)
		siteDomainStr, _ := site["domain"].(string)
		siteName, _ := site["name"].(string)
		siteCertID, hasCert := site["cert_id"].(float64)

		siteDomains := strings.Fields(siteDomainStr)

		if !hasCert || siteCertID <= 0 {
			needList = append(needList, map[string]any{
				"site_id":     siteID,
				"site_name":   siteName,
				"site_domain": siteDomainStr,
				"operate":     "create",
				"cert_id":     0,
			})
			continue
		}

		cert, exist := certMap[siteCertID]
		if !exist {
			needList = append(needList, map[string]any{
				"site_id":     siteID,
				"site_name":   siteName,
				"site_domain": siteDomainStr,
				"operate":     "create",
				"cert_id":     siteCertID,
			})
			continue
		}

		expireStr, ok := cert["expire_time2"].(string)
		if !ok {
			return nil, fmt.Errorf("证书[%.0f] 过期时间格式错误", siteCertID)
		}
		expireTime, err := time.Parse("2006-01-02 15:04:05", expireStr)
		if err != nil {
			return nil, fmt.Errorf("解析证书[%.0f]过期时间失败: %w", siteCertID, err)
		}

		remain := expireTime.Sub(now)
		certDomainStr, _ := cert["domain"].(string)
		certDomains := strings.Fields(certDomainStr)

		match := false
		for _, sDom := range siteDomains {
			for _, cDom := range certDomains {
				if domainMatch(cDom, sDom) {
					match = true
					break
				}
			}
			if match {
				break
			}
		}

		if remain < threshold {
			needList = append(needList, map[string]any{
				"site_id":     siteID,
				"site_name":   siteName,
				"site_domain": siteDomainStr,
				"operate":     "update",
				"cert_id":     siteCertID,
				"expire_time": expireStr,
				"remain_days": remain.Hours() / 24,
			})
			continue
		}

		if match {
			continue
		}

		needList = append(needList, map[string]any{
			"site_id":     siteID,
			"site_name":   siteName,
			"site_domain": siteDomainStr,
			"operate":     "replace",
			"cert_id":     siteCertID,
		})
	}

	return needList, nil
}

// domainMatch 泛域名/精确域名匹配
func domainMatch(certDomain, target string) bool {
	certDomain = strings.TrimSpace(strings.ToLower(certDomain))
	target = strings.TrimSpace(strings.ToLower(target))

	if certDomain == target {
		return true
	}
	if strings.HasPrefix(certDomain, "*.") {
		suffix := strings.TrimPrefix(certDomain, "*")
		if strings.HasSuffix(target, suffix) && target != suffix {
			return true
		}
	}
	return false
}

func UpdateCert(config *Config, certInfo map[string]any, cert string, key string) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	requestData := map[string]any{
		"cert": cert,
		"des":  "AllInSSL自动更新",
		"key":  key,
		"id":   certInfo["id"],
		"name": certInfo["name"],
		"type": "custom",
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %w", err)
	}

	url := fmt.Sprintf("%s/certs", config.BaseURL)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("创建PUT请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", config.ApiKey)
	req.Header.Set("api-secret", config.ApiSecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("更新证书失败，HTTP状态码: %d", resp.StatusCode)
	}

	var respMap map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&respMap); err != nil {
		return fmt.Errorf("解析响应数据失败: %w", err)
	}

	code, _ := respMap["code"].(float64)
	if code != 0 {
		msg, _ := respMap["message"].(string)
		return fmt.Errorf("接口业务失败: %s", msg)
	}

	return nil
}

// CreateCert 新建自定义证书，返回新建证书ID
func CreateCert(config *Config, name string, cert string, key string) (int64, error) {
	if config == nil {
		return 0, fmt.Errorf("配置不能为空")
	}

	requestData := map[string]any{
		"name": name,
		"cert": cert,
		"des":  "AllInSSL自动创建",
		"key":  key,
		"type": "custom",
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return 0, fmt.Errorf("序列化请求数据失败: %w", err)
	}

	url := fmt.Sprintf("%s/certs", config.BaseURL)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", config.ApiKey)
	req.Header.Set("api-secret", config.ApiSecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("请求接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("新建证书失败，HTTP状态码: %d", resp.StatusCode)
	}

	var respMap map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&respMap); err != nil {
		return 0, fmt.Errorf("解析响应数据失败: %w", err)
	}

	code, _ := respMap["code"].(float64)
	if code != 0 {
		msg, _ := respMap["message"].(string)
		return 0, fmt.Errorf("接口业务异常: %s", msg)
	}

	certID, ok := respMap["data"].(float64)
	if !ok {
		return 0, fmt.Errorf("获取新建证书ID失败")
	}
	return int64(certID), nil
}

// SetSiteCert 为站点绑定证书
func SetSiteCert(config *Config, siteID int64, certID int64) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}
	if siteID <= 0 {
		return fmt.Errorf("网站ID不能小于等于0")
	}
	if certID <= 0 {
		return fmt.Errorf("证书ID不能小于等于0")
	}

	requestData := map[string]any{
		"id": siteID,
		"https_listen": map[string]any{
			"cert": certID,
		},
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %w", err)
	}

	url := fmt.Sprintf("%s/sites/%d", config.BaseURL, siteID)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("创建PUT请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", config.ApiKey)
	req.Header.Set("api-secret", config.ApiSecret)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("设置网站证书失败，HTTP状态码: %d", resp.StatusCode)
	}

	var respMap map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&respMap); err != nil {
		return fmt.Errorf("解析响应数据失败: %w", err)
	}

	code, _ := respMap["code"].(float64)
	if code != 0 {
		msg, _ := respMap["message"].(string)
		return fmt.Errorf("接口业务异常: %s", msg)
	}

	return nil
}

// ExecuteAction 批量执行证书操作：create / update / replace
func ExecuteAction(config *Config, cert string, key string, actions []map[string]any) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}
	if len(actions) == 0 {
		return fmt.Errorf("操作列表为空")
	}

	// 预加载全量证书，用于update场景
	certList, err := GetAllCertificates(config)
	if err != nil {
		return fmt.Errorf("预加载证书列表失败: %w", err)
	}
	certMap := make(map[float64]map[string]any)
	for _, c := range certList {
		cid, ok := c["id"].(float64)
		if ok {
			certMap[cid] = c
		}
	}

	for _, action := range actions {
		operate, ok := action["operate"].(string)
		if !ok {
			return fmt.Errorf("未知操作类型")
		}

		siteIDFloat, ok := action["site_id"].(float64)
		if !ok {
			return fmt.Errorf("站点ID格式错误")
		}
		siteID := int64(siteIDFloat)

		siteName, _ := action["site_name"].(string)
		certIDFloat, _ := action["cert_id"].(float64)

		switch operate {
		case "create":
			// 新建证书 + 绑定站点
			newCertID, err := CreateCert(config, siteName, cert, key)
			if err != nil {
				return fmt.Errorf("新建证书失败: %w", err)
			}
			if err := SetSiteCert(config, siteID, newCertID); err != nil {
				return fmt.Errorf("绑定站点证书失败: %w", err)
			}

		case "update":
			// 更新已有证书
			oldCert, exist := certMap[certIDFloat]
			if !exist {
				return fmt.Errorf("证书ID %.0f 不存在", certIDFloat)
			}
			if err := UpdateCert(config, oldCert, cert, key); err != nil {
				return fmt.Errorf("更新证书失败: %w", err)
			}

		case "replace":
			// 新建证书 + 替换绑定
			newCertID, err := CreateCert(config, siteName, cert, key)
			if err != nil {
				return fmt.Errorf("新建替换证书失败: %w", err)
			}
			if err := SetSiteCert(config, siteID, newCertID); err != nil {
				return fmt.Errorf("替换站点证书失败: %w", err)
			}

		default:
			return fmt.Errorf("不支持的操作类型: %s", operate)
		}
	}
	return nil
}
