package kustz

import (
	"math"
	"strconv"
)

// CommonLabels 生成通用标签映射
//
// 标签设计:
//
//	标签用于资源关联和选择:
//	  - "app": 应用名称，作为资源选择器
//
//	使用场景:
//
//	  - Deployment.spec.selector.matchLabels
//	  - Service.spec.selector
//	  - Pod template.metadata.labels
//	  - 所有资源都会继承这些标签
func (kz *Config) CommonLabels() map[string]string {
	return map[string]string{
		"app": kz.Name,
	}
}

// StringToInt32 将字符串转换为 int32
//
//	转换规则:
//
//	  - 有效数字字符串转换为对应整数值
//	  - 空字符串或无效输入返回 0
//	  - 超出 int32 范围的值返回 0
func StringToInt32(val string) int32 {
	i, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	if i > math.MaxInt32 || i < math.MinInt32 {
		return 0
	}
	return int32(i)
}

// hasConfigMaps 检查是否配置了 ConfigMap
func (kz *Config) hasConfigMaps() bool {
	return len(kz.ConfigMaps.Literals) > 0 || len(kz.ConfigMaps.Envs) > 0 || len(kz.ConfigMaps.Files) > 0
}

// hasSecrets 检查是否配置了 Secret
func (kz *Config) hasSecrets() bool {
	return len(kz.Secrets.Literals) > 0 || len(kz.Secrets.Envs) > 0 || len(kz.Secrets.Files) > 0
}
