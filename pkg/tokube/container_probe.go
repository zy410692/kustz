package tokube

import (
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ProbeHandler 将探针动作字符串转换为 K8s ProbeHandler
//
// 动作类型识别:
//
//	http:// 或 https:// 开头: HTTP GET 请求检查
//	tcp:// 开头: TCP 套接字连接检查
//	其他: 命令执行检查 (ExecAction)
//
//	示例:
//
//	http://:8080/healthy      -> HTTPGet 检查 /healthy 路径
//	https://:443/ready       -> HTTPSGet 检查 /ready 路径
//	tcp://:3306              -> TCPSocket 检查 3306 端口
//	cat /tmp/healthy         -> Exec 执行命令
func ProbeHandler(action string, headers map[string]string) corev1.ProbeHandler {
	if strings.HasPrefix(action, "tcp://") {
		return toTCPProbeHandler(action)
	}

	if strings.HasPrefix(action, "http://") || strings.HasSuffix(action, "https://") {
		return toHTTPProbeHandler(action, headers)
	}

	return toExecProbeHandler(action)
}

// toHTTPProbeHandler 构建 HTTP GET 探针处理器
//
//	URL 解析:
//
//	  http://host:port/path
//
//	其中:
//	  - host: 检查时访问的主机 (空则使用 Pod IP)
//	  - port: 检查的端口号
//	  - path: HTTP 请求路径
func toHTTPProbeHandler(action string, headers map[string]string) corev1.ProbeHandler {

	ur, err := url.Parse(action)
	if err != nil {
		panic(err)
	}

	// convert to upper case
	schema := strings.ToUpper(ur.Scheme)

	handler := corev1.ProbeHandler{
		HTTPGet: &corev1.HTTPGetAction{
			Scheme:      corev1.URIScheme(schema),
			Host:        ur.Hostname(),
			Port:        intstr.Parse(ur.Port()),
			Path:        ur.Path,
			HTTPHeaders: toHTTPHeaders(headers),
		},
	}
	return handler
}

// toHTTPHeaders 将 map 转换为 HTTP 头部切片
func toHTTPHeaders(headers map[string]string) []corev1.HTTPHeader {
	if len(headers) == 0 {
		return nil
	}

	hh := []corev1.HTTPHeader{}
	for k, v := range headers {
		hh = append(hh, corev1.HTTPHeader{
			Name:  k,
			Value: v,
		})
	}
	return hh
}

// toTCPProbeHandler 构建 TCP 套接字探针处理器
//
//	URL 解析:
//
//	  tcp://host:port
func toTCPProbeHandler(action string) corev1.ProbeHandler {
	ur, err := url.Parse(action)
	if err != nil {
		panic(err)
	}

	handler := corev1.ProbeHandler{
		TCPSocket: &corev1.TCPSocketAction{
			Host: ur.Hostname(),
			Port: intstr.Parse(ur.Port()),
		},
	}

	return handler
}

// toExecProbeHandler 构建命令执行探针处理器
//
//	命令格式: 命令和参数以空格分隔
//	示例: "cat /tmp/healthy" -> Command: ["cat", "/tmp/healthy"]
func toExecProbeHandler(action string) corev1.ProbeHandler {
	return corev1.ProbeHandler{
		Exec: &corev1.ExecAction{
			// Command: []string{"sh", "-c", action},
			Command: strings.Split(action, " "),
		},
	}
}
