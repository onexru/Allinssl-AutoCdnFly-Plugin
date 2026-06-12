package domain

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
)

// MatchDomain 检查证书域名列表是否匹配目标域名（支持泛域名）
func MatchDomain(certDomains []string, targetDomain string) bool {
	targetDomain = strings.ToLower(strings.TrimSpace(targetDomain))

	for _, certDomain := range certDomains {
		certDomain = strings.ToLower(strings.TrimSpace(certDomain))
		if certDomain == targetDomain {
			return true
		}
		if strings.HasPrefix(certDomain, "*.") {
			suffix := strings.TrimPrefix(certDomain, "*")
			if strings.HasSuffix(targetDomain, suffix) && targetDomain != suffix {
				return true
			}
		}
	}
	return false
}

// ParseCertInfo 解析PEM证书，返回证书详情和所有域名
func ParseCertInfo(certStr string) (*x509.Certificate, []string, error) {
	if certStr == "" {
		return nil, nil, fmt.Errorf("证书内容不能为空")
	}

	var cert *x509.Certificate
	remaining := []byte(certStr)

	for {
		block, rest := pem.Decode(remaining)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			parsedCert, err := x509.ParseCertificate(block.Bytes)
			if err == nil {
				cert = parsedCert
				break
			}
		}
		remaining = rest
	}

	if cert == nil {
		return nil, nil, fmt.Errorf("未找到有效的PEM证书")
	}

	var domains []string
	if cert.Subject.CommonName != "" {
		domains = append(domains, cert.Subject.CommonName)
	}
	domains = append(domains, cert.DNSNames...)

	// 去重
	uniqueDomains := make([]string, 0)
	exist := make(map[string]bool)
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d != "" && !exist[d] {
			exist[d] = true
			uniqueDomains = append(uniqueDomains, d)
		}
	}

	return cert, uniqueDomains, nil
}

// CheckCertDomains 过滤出证书域名匹配的站点
func CheckCertDomains(certDomains []string, expectedDomains []map[string]any) []map[string]any {
	var result []map[string]any
	for _, site := range expectedDomains {
		siteDomain, ok := site["site_domain"].(string)
		if !ok {
			continue
		}
		if MatchDomain(certDomains, siteDomain) {
			result = append(result, site)
		}
	}
	return result
}
