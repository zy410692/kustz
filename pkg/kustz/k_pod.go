package kustz

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubePod 生成 K8s Pod 模板规格
//
// Pod 模板设计:
//
//	Pod 模板包含以下配置:
//
//	  1. 容器规范 (Containers)
//	     - 容器名称、镜像
//	     - 环境变量
//	     - 资源限制
//	     - 健康检查探针
//
//	  2. 镜像拉取凭证 (ImagePullSecrets)
//	     - 私有镜像仓库认证
//
//	  3. DNS 配置 (DNSConfig/DNSPolicy)
//	     - DNS 服务器
//	     - DNS 搜索域
//	     - DNS 选项
func (kz *Config) KubePod() corev1.PodTemplateSpec {
	var affinity *corev1.Affinity
	// 优先使用 service.affinity，其次使用顶层 affinity
	if kz.Service.Affinity != nil {
		affinity = kz.Service.Affinity.ToK8s()
	} else if kz.Affinity != nil {
		affinity = kz.Affinity.ToK8s()
	}

	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: kz.CommonLabels(),
		},
		Spec: corev1.PodSpec{
			Containers:       kz.KubeContainer(),
			ImagePullSecrets: toImagePullSecrets(kz.Service.ImagePullSecrets),
			DNSConfig:        toPodDNSConfig(kz.DNS),
			DNSPolicy:        toDNSPolicy(kz.DNS),
			Affinity:         affinity,
		},
	}
}

// toImagePullSecrets 将字符串列表转换为 K8s 镜像拉取凭证引用
func toImagePullSecrets(secrets []string) []corev1.LocalObjectReference {
	if len(secrets) == 0 {
		return nil
	}

	objs := []corev1.LocalObjectReference{}
	for _, s := range secrets {
		objs = append(objs, corev1.LocalObjectReference{
			Name: s,
		})
	}

	return objs
}

// toPodDNSConfig 转换 Pod DNS 配置
//
//	DNS 配置示例:
//
//	  dns:
//	    config:
//	      nameservers:
//	        - 8.8.8.8
//	      searches:
//	        - default.svc.cluster.local
func toPodDNSConfig(dns *DNS) *corev1.PodDNSConfig {
	if dns == nil {
		return nil
	}
	if dns.Config == nil {
		return nil
	}
	return &corev1.PodDNSConfig{
		Nameservers: dns.Config.Nameservers,
		Searches:    dns.Config.Searches,
		Options:     dns.Config.PodDNSConfigOptions(),
	}
}

// toDNSPolicy 转换 DNS 策略
//
//	可用策略:
//
//	  - ""        : 默认策略 (ClusterFirst)
//	  - "None"    : 不使用任何 DNS 策略
//	  - "ClusterFirst": 使用集群 DNS
//	  - "Default" : 使用节点上配置的 DNS
func toDNSPolicy(dns *DNS) corev1.DNSPolicy {
	if dns == nil {
		// return v1.DNSNone
		return ""
	}

	return dns.DNSPolicy()
}
