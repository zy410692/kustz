package tokube

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ContainerResources 将资源配置 map 转换为 K8s ResourceRequirements
//
// 资源格式:
//
//	resources:
//	  cpu: 10m/20m              # request=10m, limit=20m
//	  memory: 10Mi/20Mi         # request=10Mi, limit=20Mi
//	  nvidia.com/gpu: 1/1       # GPU 资源
//
//	格式说明:
//
//	  - request/limit 格式，用 / 分隔
//	  - 只有 request 时，limit 等于 request
//	  - 单位: m (milli CPU), Mi/Gi/Ti (内存), 无单位 (GPU)
func ContainerResources(res map[string]string) corev1.ResourceRequirements {
	// 如果资源为空， 直接返回
	if len(res) == 0 {
		return corev1.ResourceRequirements{}
	}

	limits := corev1.ResourceList{}
	requests := corev1.ResourceList{}

	for k, v := range res {
		name := corev1.ResourceName(k)
		req, limit := toResourceQuantity(v)
		limits[name] = limit
		requests[name] = req
	}
	return corev1.ResourceRequirements{
		Limits:   limits,
		Requests: requests,
	}
}

// toResourceQuantity 解析资源值为 request 和 limit
//
//	输入格式: request/limit
//
//	解析规则:
//
//	  "10m"       -> request=10m, limit=10m
//	  "10m/20m"   -> request=10m, limit=20m
//	  "1Gi/2Gi"   -> request=1Gi, limit=2Gi
//
//	支持的资源类型:
//
//	  - cpu: 单位 m (millicore), 1 CPU = 1000m
//	  - memory: 单位 Ei, Pi, Ti, Gi, Mi, Ki
//	  - nvidia.com/gpu: 整数数量
func toResourceQuantity(value string) (request resource.Quantity, limit resource.Quantity) {

	re, li := "", ""
	parts := strings.Split(value, "/")
	if len(parts) == 1 {
		re = value
		li = value
	}
	if len(parts) == 2 {
		re = parts[0]
		li = parts[1]
	}

	request = resource.MustParse(re)
	limit = resource.MustParse(li)

	return
}
