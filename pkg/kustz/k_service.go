package kustz

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// KubeService 将 kustz 配置转换为 K8s Service 资源
//
// 服务发现设计:
//   - Service 通过标签选择器关联 Pod
//   - CommonLabels() 确保 Deployment 和 Service 标签一致
//   - 实现了 K8s 服务的自动服务发现机制
func (kz *Config) KubeService() *corev1.Service {

	ports, typ := ParsePortStrings(kz.Service.Ports)

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   kz.Name,
			Labels: kz.CommonLabels(),
			// Namespace: kz.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: kz.CommonLabels(),
			Type:     typ,
			Ports:    ports,
		},
	}

	return svc
}

// ParsePortStrings 解析端口配置字符串
//
// 端口格式设计:
//
//	ClusterIP 格式:
//	  "8080"           -> port:8080, targetPort:8080
//	  "80:8080"        -> port:80, targetPort:8080
//
//	NodePort 格式 (前缀 !):
//	  "!8080"          -> port:8080, targetPort:8080, nodePort:随机
//	  "!80:8080"       -> port:80, targetPort:8080, nodePort:随机
//	  "!10080:80:8080" -> nodePort:10080, port:80, targetPort:8080
//
//	协议前缀 (tcp://, udp://, sctp://):
//	  "udp://!9998:8889" -> UDP 协议, nodePort:9998, port:8889, targetPort:8889
func ParsePortStrings(values []string) ([]corev1.ServicePort, corev1.ServiceType) {

	sps := []corev1.ServicePort{}
	typ := corev1.ServiceTypeClusterIP

	// 如果没有配置端口，默认生成一个端口
	if len(values) == 0 {
		return []corev1.ServicePort{
			{
				Name:       "http",
				Port:       80,
				TargetPort: intstr.FromInt(80),
				Protocol:   corev1.ProtocolTCP,
			},
		}, corev1.ServiceTypeClusterIP
	}

	for _, value := range values {
		port := NewPortFromString(value)
		if port.Type != corev1.ServiceTypeClusterIP {
			typ = port.Type
		}
		sps = append(sps, port.KubeServicePort())

	}

	return sps, typ
}

// PortString 表示解析后的端口配置结构
type PortString struct {
	Port       int32
	TargetPort int32
	NodePort   int32
	Protocol   corev1.Protocol
	Type       corev1.ServiceType
}

// NewPortFromString parse port from string PortString
//
// 解析优先级:
//
//  1. 协议前缀: tcp://, udp://, sctp://
//
//  2. NodePort 前缀: !
//
//  3. 端口映射: containerPort:targetPort
//
//     示例解析:
//
//     "8080"           -> {Port:8080, TargetPort:8080, Protocol:TCP, Type:ClusterIP}
//     "80:8080"        -> {Port:80, TargetPort:8080, Protocol:TCP, Type:ClusterIP}
//     "!8080"          -> {Port:8080, TargetPort:8080, NodePort:随机, Type:NodePort}
//     "udp://!9998:8889" -> {Port:8889, TargetPort:8889, NodePort:9998, Protocol:UDP, Type:NodePort}
func NewPortFromString(value string) PortString {
	port := &PortString{
		Protocol: corev1.ProtocolTCP,
		Type:     corev1.ServiceTypeClusterIP,
	}
	parts := strings.Split(value, "://")
	if len(parts) == 2 {
		value = parts[1]

		proto := parts[0]
		switch strings.ToLower(proto) {
		case "udp":
			port.Protocol = corev1.ProtocolUDP
		case "sctp":
			port.Protocol = corev1.ProtocolSCTP
		default:
			port.Protocol = corev1.ProtocolTCP
		}
	}

	sign := value[0]
	switch sign {
	case '!':
		port.toServiceNodePort(value)
	default:
		port.toServiceClusterIP(value)
	}

	return *port
}

// KubeServicePort 将 PortString 转换为 K8s ServicePort
func (p *PortString) KubeServicePort() corev1.ServicePort {

	sp := &corev1.ServicePort{
		Name:       fmt.Sprintf("%d-%d", p.Port, p.TargetPort),
		Port:       p.Port,
		TargetPort: intstr.FromInt(int(p.TargetPort)),
		Protocol:   p.Protocol,
	}

	// NodePort 为 0 时不设置，让 K8s 自动分配
	if p.NodePort != 0 {
		sp.NodePort = p.NodePort
	}
	return *sp
}

// toServiceClusterIP parse value from for ClusterIP
//
//	解析规则:
//
//	  "8080"     -> Port=8080, TargetPort=8080
//	  "80:8080"  -> Port=80, TargetPort=8080
func (p *PortString) toServiceClusterIP(value string) {

	parts := strings.Split(value, ":")
	switch len(parts) {
	case 1:
		n := StringToInt32(parts[0])
		p.Port = n
		p.TargetPort = n
	case 2:
		p.Port = StringToInt32(parts[0])
		p.TargetPort = StringToInt32(parts[1])
	}

	p.Type = corev1.ServiceTypeClusterIP
}

// toServiceNodePort parse value from for NodePort
//
//	解析规则:
//
//	  "!8080"          -> NodePort=随机, Port=8080, TargetPort=8080
//	  "!80:8080"       -> NodePort=随机, Port=80, TargetPort=8080
//	  "!10080:80:8080" -> NodePort=10080, Port=80, TargetPort=8080
func (p *PortString) toServiceNodePort(value string) {

	value = strings.TrimPrefix(value, "!")
	parts := strings.Split(value, ":")
	switch len(parts) {
	case 1, 2:
		p.toServiceClusterIP(value)
	case 3:
		p.NodePort = StringToInt32(parts[0])
		p.Port = StringToInt32(parts[1])
		p.TargetPort = StringToInt32(parts[2])
	}

	p.Type = corev1.ServiceTypeNodePort
}
