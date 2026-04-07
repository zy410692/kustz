package tokube

import (
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// ContainerEnv 将键值对 map 转换为环境变量切片
//
//	EnvVar 结构:
//
//	  - Name:  环境变量名称
//	  - Value: 环境变量值 (直接值，非引用)
func ContainerEnv(pairs map[string]string) []corev1.EnvVar {
	envs := []corev1.EnvVar{}
	for k, v := range pairs {
		envs = append(envs, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return envs
}

// ParseEnvFromSource 解析 EnvFromSource 配置字符串
//
//	配置格式:
//
//	  name           -> 引用整个 Secret/ConfigMap
//	  name:true      -> 引用整个 Secret/ConfigMap，optional=true
//	  name:false     -> 引用整个 Secret/ConfigMap，optional=false
//
//	示例:
//
//	  - my-secret        -> SecretRef: my-secret, Optional: false
//	  - my-secret:true   -> SecretRef: my-secret, Optional: true
//	  - my-configmap     -> ConfigMapRef: my-configmap, Optional: false
func ParseEnvFromSource(value string, kind string) corev1.EnvFromSource {
	opt := false
	var err error
	parts := strings.Split(value, ":")
	if len(parts) == 2 {
		value = parts[0]
		opt, err = strconv.ParseBool(parts[1])
		if err != nil {
			opt = false
		}
	}

	switch kind {
	case "secret":
		return corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: value,
				},
				Optional: &opt,
			},
		}
	case "configmap":
		fallthrough
	default:
		return corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: value,
				},
				Optional: &opt,
			},
		}
	}
}
