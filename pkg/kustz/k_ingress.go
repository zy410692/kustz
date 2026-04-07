package kustz

import (
	"net/url"
	"strings"

	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeIngress 将 kustz 配置转换为 K8s Ingress 资源
//
// 入口路由设计:
//   - 使用 URL 格式实现语义化路由配置
//   - 自动提取 TLS 配置和后端服务
//   - 简化 Ingress 资源的复杂性
func (kz *Config) KubeIngress() *netv1.Ingress {

	rules, tlss := ParseIngreseRulesFromStrings(kz.Ingress.Rules, kz.Name)
	ing := &netv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        kz.Name,
			Labels:      kz.CommonLabels(),
			Annotations: kz.Ingress.Annotations,
		},
		Spec: netv1.IngressSpec{
			Rules: rules,
			TLS:   tlss,
		},
	}

	return ing
}

// ParseIngreseRulesFromStrings 批量解析 Ingress 规则字符串
//
// 规则格式:
//
//	http://host[:port]/path[?params]
//
//	参数说明:
//
//	  - tls: TLS Secret 名称
//	  - svc: 后端服务名称[:端口]
//
//	示例:
//
//	  - http://api.example.com/ping
//	  - http://api.example.com/api?tls=my-tls&svc=my-svc:8080
func ParseIngreseRulesFromStrings(values []string, defaultService string) ([]netv1.IngressRule, []netv1.IngressTLS) {

	rules := []netv1.IngressRule{}
	tlss := []netv1.IngressTLS{}
	for _, value := range values {
		ing := NewIngressRuleFromString(value)
		if ing == nil {
			continue
		}

		if ing.Service == "" {
			ing.Service = defaultService
		}

		rules = append(rules, ing.KubeIngressRule())
		if tls := ing.KubeIngressTLS(); tls != nil {
			tlss = append(tlss, *tls)
		}
	}
	return rules, tlss
}

// IngressRuleString 表示解析后的 Ingress 规则结构
type IngressRuleString struct {
	Host      string
	Path      string
	PathType  netv1.PathType
	TLSSecret string
	Service   string
}

// NewIngressRuleFromString 从 URL 字符串解析 Ingress 规则
//
//	URL 组件映射:
//
//	  - scheme: 必须是 http 或 https (https 用于 TLS 配置)
//	  - host: Ingress 规则的域名
//	  - path: 路由路径，支持通配符 *
//	  - query tls: TLS Secret 名称
//	  - query svc: 后端服务名和端口
//
//	路径类型:
//
//	  - /api        -> PathType: Exact
//	  - /api/*      -> PathType: Prefix (自动识别)
func NewIngressRuleFromString(value string) *IngressRuleString {

	ur, err := url.Parse(value)
	if err != nil {
		return nil
	}

	// ex: /api/*
	path := ur.Path
	typ := netv1.PathTypeExact
	if strings.HasSuffix(path, "*") {
		path = strings.TrimSuffix(path, "*")
		typ = netv1.PathTypePrefix
	}

	ing := &IngressRuleString{
		Host:      ur.Hostname(),
		Path:      path,
		PathType:  typ,
		TLSSecret: ur.Query().Get("tls"),
		Service:   ur.Query().Get("svc"),
	}

	return ing
}

// KubeIngressTLS 将 Ingress 规则转换为 K8s TLS 配置
func (ir *IngressRuleString) KubeIngressTLS() *netv1.IngressTLS {
	if ir.TLSSecret == "" {
		return nil
	}

	return &netv1.IngressTLS{
		Hosts: []string{
			ir.Host,
		},
		SecretName: ir.TLSSecret,
	}
}

// KubeIngressRule 将 Ingress 规则转换为 K8s IngressRule
func (ir *IngressRuleString) KubeIngressRule() netv1.IngressRule {

	ing := netv1.IngressRule{
		Host: ir.Host,
		IngressRuleValue: netv1.IngressRuleValue{
			HTTP: &netv1.HTTPIngressRuleValue{
				Paths: []netv1.HTTPIngressPath{
					{
						Path:     ir.Path,
						PathType: &ir.PathType,
						Backend:  ir.toKubeIngressBackend(),
					},
				},
			},
		},
	}

	return ing
}

// toKubeIngressBackend 解析服务配置
//
//	服务格式:
//
//	  service-name           -> 服务: default-name, 端口: 80
//	  service-name:8080      -> 服务: service-name, 端口: 8080
func (ir *IngressRuleString) toKubeIngressBackend() netv1.IngressBackend {

	// srv-webapp-demo[:8080]
	svc := ir.Service
	port := int32(80)

	parts := strings.Split(svc, ":")
	if len(parts) == 2 {
		svc = parts[0]
		port = StringToInt32(parts[1])
	}

	return netv1.IngressBackend{
		Service: &netv1.IngressServiceBackend{
			Name: svc,
			Port: netv1.ServiceBackendPort{
				Number: int32(port),
			},
		},
	}
}
