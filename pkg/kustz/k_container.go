package kustz

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/tangx/kustz/pkg/kubeutils"
	"github.com/tangx/kustz/pkg/tokube"
	corev1 "k8s.io/api/core/v1"
)

// KubeContainer 构建 K8s 容器配置
//
// 容器配置组合设计:
//
//	容器规范由多个独立配置组合而成:
//
//	  1. 基础配置 (Name, Image, ImagePullPolicy)
//	  2. 环境变量 (kubeContainerEnv)
//	  3. 环境变量来源 (kubeContainerEnvFrom)
//	  4. 资源限制 (kubeContainerResources)
//	  5. 健康检查 (Probes)
//
//	每个配置模块独立构建，最后组合成完整的容器规范
//
//	envs 配置示例:
//
//	  envs:
//	    pairs:
//	      KEY: value
//	    files:
//	      - config.yaml
//	    secrets:
//	      - my-secret:true
//	    configmaps:
//	      - my-configmap:true
//
//	解析规则:
//	  - "name:true" 表示使用 Secret/ConfigMap 的所有数据
//	  - "name:key" 表示使用 Secret/ConfigMap 中的指定 key
func (kz *Config) KubeContainer() []corev1.Container {
	if kz.Service.Name == "" {
		kz.Service.Name = kz.Name
	}

	probes := kz.Service.Probes

	c := corev1.Container{
		Name:            kz.Service.Name,
		Image:           kz.Service.Image,
		Env:             kz.kubeContainerEnv(),
		EnvFrom:         kz.kubeContainerEnvFrom(),
		Resources:       kz.kubeContainerResources(),
		LivenessProbe:   probes.kubeProbe(probes.Liveness),
		ReadinessProbe:  probes.kubeProbe(probes.Readiness),
		StartupProbe:    probes.kubeProbe(probes.Startup),
		ImagePullPolicy: tokube.ImagePullPolicy(kz.Service.ImagePullPolicy),
	}

	return []corev1.Container{c}
}

// kubeContainerEnvFrom 定义 configmap 或 secret 数据容器变量
// https://kubernetes.io/docs/concepts/configuration/secret/
//
// EnvFrom 模式将 Secret/ConfigMap 的数据作为环境变量注入容器
// 格式: name[:key]
//   - "my-secret" 使用 Secret 所有数据
//   - "my-secret:db-password" 使用 Secret 中的指定 key
func (kz *Config) kubeContainerEnvFrom() []corev1.EnvFromSource {

	sources := []corev1.EnvFromSource{}

	// value = config-name:true 或 config-name:key
	for _, value := range kz.Service.Envs.Secrets {
		sources = append(sources, tokube.ParseEnvFromSource(value, "secret"))
	}

	for _, value := range kz.Service.Envs.ConfigMaps {
		sources = append(sources, tokube.ParseEnvFromSource(value, "configmap"))
	}

	return sources
}

// kubeContainerEnv 构建容器环境变量
//
// 环境变量来源优先级:
//  1. Files 读取的配置文件
//  2. Pairs 直接定义的键值对
//
// 文件格式支持:
//   - YAML: key: value
//   - ENV: KEY=value
func (kz *Config) kubeContainerEnv() []corev1.EnvVar {
	pairs := make(map[string]string, 0)

	for _, file := range kz.Service.Envs.Files {
		b, err := os.ReadFile(file)
		if err != nil {
			logrus.Fatalf("read env file failed: %v", err)
		}
		// err = yaml.Unmarshal(b, &pairs)
		err = kubeutils.YamlPkgUnmarshal(b, &pairs)
		if err != nil {
			logrus.Fatalf("unmarshal env file failed: %v", err)
		}
	}

	for k, v := range kz.Service.Envs.Pairs {
		pairs[k] = v
	}

	return tokube.ContainerEnv(pairs)
}

// kubeContainerResources 返回容器资源申请
// https://kubernetes.io/zh-cn/docs/concepts/configuration/manage-resources-containers/
//
// 资源格式: request/limit
//
//	resources:
//	  cpu: 10m/20m        # request=10m, limit=20m
//	  memory: 10Mi/20Mi   # request=10Mi, limit=20Mi
//	  nvidia.com/gpu: 1/1 # GPU 资源
//
//	单位说明:
//	  - m: milli, 1m = 0.001 CPU
//	  - Mi: Mebibyte, 1Mi = 1024*1024 bytes
func (kz *Config) kubeContainerResources() corev1.ResourceRequirements {
	return tokube.ContainerResources(kz.Service.Resources)
}

// kubeProbe 将 kustz 探针配置转换为 K8s 探针配置
func (cps ContainerProbes) kubeProbe(cp *ContainerProbe) *corev1.Probe {
	if cp == nil {
		return nil
	}
	return cp.kubeProbe()
}

// kubeProbe return K8s Probe without handler
//
// 探针动作类型由 action 前缀决定:
//
//	http:// 或 https://: HTTP GET 请求
//	tcp://: TCP 套接字检查
//	其他: 命令执行检查
func (cp *ContainerProbe) kubeProbe() *corev1.Probe {
	handler := tokube.ProbeHandler(cp.Action, cp.Headers)
	return &corev1.Probe{
		ProbeHandler:                  handler,
		InitialDelaySeconds:           cp.InitialDelaySeconds,
		TimeoutSeconds:                cp.TimeoutSeconds,
		PeriodSeconds:                 cp.PeriodSeconds,
		SuccessThreshold:              cp.SuccessThreshold,
		FailureThreshold:              cp.FailureThreshold,
		TerminationGracePeriodSeconds: cp.TerminationGracePeriodSeconds,
	}
}
