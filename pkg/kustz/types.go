package kustz

import (
	"os"

	"github.com/tangx/kustz/pkg/kubeutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// APIVersion 定义 kustz 配置文件的版本标识
// 用于配置文件的版本控制和兼容性检查
const (
	APIVersion = "kustz/v1"
)

// Config 是 kustz 的核心配置结构
// 采用语义化设计，将复杂的 K8s 资源定义简化为直观的 YAML 配置
//
// 设计理念:
//   - 单一配置源: 一个文件定义整个应用的 K8s 资源配置
//   - 语义化字段: 使用易读的字段名，如 name, image, replicas
//   - 内聚性: 将 Deployment, Service, Ingress 等相关配置整合在一起
//
// 示例配置:
//
//	name: nginx
//	image: docker.io/library/nginx:alpine
//	replicas: 2
//	service:
//	  ports:
//	    - "80:8080"
//
//	对应的 K8s 资源:
//	  - Deployment: 管理应用 Pod
//	  - Service: 提供集群内访问
//	  - Ingress: 对外暴露访问入口
type Config struct {
	Metadata `json:",inline" yaml:",inline"`

	// Name 应用名称，同时也是资源的基础标识
	Name string `json:"name" yaml:"name"`

	// Namespace 部署的命名空间
	Namespace string `json:"namespace" yaml:"namespace"`

	// Service 定义应用服务和容器配置
	Service Service `json:"service" yaml:"service"`

	// ConfigMaps 生成 K8s ConfigMap 资源配置
	ConfigMaps Generator `json:"configmaps,omitempty" yaml:"configmaps,omitempty"`

	// Secrets 生成 K8s Secret 资源配置
	Secrets Generator `json:"secrets,omitempty" yaml:"secrets,omitempty"`

	// Ingress 定义应用的对外访问规则
	Ingress Ingress `json:"ingress,omitempty" yaml:"ingress,omitempty"`

	// Outputs 定义需要输出的资源类型
	// 默认输出 Deployment 和 Service
	// 可选: ingress, configmaps, secrets
	Outputs Outputs `json:"outputs,omitempty" yaml:"outputs,omitempty"`

	// DNS 配置 Pod 的 DNS 策略和配置
	DNS *DNS `json:"dns,omitempty" yaml:"dns,omitempty"`

	// Affinity 亲和性配置
	// 支持 podAntiAffinity, podAffinity, nodeAffinity
	Affinity *Affinity `json:"affinity,omitempty" yaml:"affinity,omitempty"`
}

func NewKustzFromConfig(cfg string) *Config {
	b, err := os.ReadFile(cfg)
	if err != nil {
		panic(err)
	}

	kz := &Config{}
	err = kubeutils.YamlSigUnmarshal(b, kz)
	if err != nil {
		panic(err)
	}

	if kz.Metadata.APIVersion == "" {
		kz.Metadata.APIVersion = APIVersion
	}

	return kz
}

// Service 定义应用的服务配置
// 整合了容器镜像、副本数、端口、环境变量等核心配置
//
// 语义化设计示例:
//
//	service:
//	  name: nginx
//	  image: nginx:alpine
//	  replicas: 2
//	  ports:
//	    - "80:8080"
//	  envs:
//	    pairs:
//	      KEY: value
//	  resources:
//	    cpu: 10m/20m
//	    memory: 10Mi/20Mi
type Service struct {
	// Name 容器名称，默认为应用名称
	Name string `json:"name" yaml:"name"`

	// Image 容器镜像地址
	Image string `json:"image" yaml:"image"`

	// Replicas 副本数量，默认 1
	Replicas int32 `json:"replicas" yaml:"replicas"`

	// Ports 服务端口配置
	// 格式说明:
	//   - "80:8080" 表示 containerPort:targetPort
	//   - "!80:80:8080" 表示 NodePort 类型，指定端口
	//   - "udp://!9998:8889" 表示 UDP 协议的随机 NodePort
	Ports []string `json:"ports" yaml:"ports"`

	// Envs 环境变量配置
	// 支持多种来源:
	//   - pairs: 直接键值对
	//   - files: 从 YAML/ENV 文件读取
	//   - secrets: 从 Secret 引用
	//   - configmaps: 从 ConfigMap 引用
	Envs ServiceEnvs `json:"envs,omitempty" yaml:"envs,omitempty"`

	// Resources 资源限制配置
	// 格式: request/limit
	// 示例: cpu: 10m/20m 表示 request=10m, limit=20m
	//       memory: 10Mi/20Mi 表示 request=10Mi, limit=20Mi
	Resources map[string]string `json:"resources,omitempty" yaml:"resources,omitempty"`

	// Probes 容器健康检查配置
	// 支持 liveness, readiness, startup 三种探针
	Probes ContainerProbes `json:"probes,omitempty" yaml:"probes,omitempty"`

	// ImagePullSecrets 镜像拉取凭证
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty" yaml:"imagePullSecrets,omitempty"`

	// ImagePullPolicy 镜像拉取策略
	// 可选值: Always, Never, IfNotPresent
	ImagePullPolicy string `json:"imagePullPolicy,omitempty" yaml:"imagePullPolicy,omitempty"`

	// Affinity 亲和性配置
	// 支持 podAntiAffinity, podAffinity, nodeAffinity
	Affinity *Affinity `json:"affinity,omitempty" yaml:"affinity,omitempty"`
}

// ServiceEnvs 定义环境变量的多种来源
//
// 配置示例:
//
//	envs:
//	  pairs:
//	    KEY1: value1
//	  files:
//	    - config.yaml
//	  secrets:
//	    - my-secret:true
//	  configmaps:
//	    - my-configmap:true
type ServiceEnvs struct {
	// Pairs 直接定义的键值对环境变量
	Pairs map[string]string `json:"pairs,omitempty" yaml:"pairs,omitempty"`

	// Files 从文件读取的环境变量
	// 支持 YAML 和 ENV 格式
	Files []string `json:"files,omitempty" yaml:"files,omitempty"`

	// Secrets 从 Secret 引用环境变量
	Secrets []string `json:"secrets,omitempty" yaml:"secrets,omitempty"`

	// ConfigMaps 从 ConfigMap 引用环境变量
	ConfigMaps []string `json:"configmaps,omitempty" yaml:"configmaps,omitempty"`
}

// Ingress 定义 Ingress 路由规则
//
// 语义化设计示例:
//
//	ingress:
//	  annotations:
//	    kubernetes.io/ingress.class: nginx
//	  rules:
//	    - http://api.example.com/ping?tls=star-example-com&svc=my-svc:8080
//
//	规则解析:
//	  - Host: api.example.com
//	  - Path: /ping
//	  - TLS: star-example-com (Secret 名称)
//	  - Service: my-svc:8080 (服务名:端口)
type Ingress struct {
	// Rules Ingress 路由规则
	// 支持 URL 格式的语义化配置
	Rules []string `json:"rules" yaml:"rules"`

	// Annotations Ingress 注解配置
	Annotations map[string]string `json:"annotations" yaml:"annotations"`
}

// Generator 定义 ConfigMap/Secret 的数据生成器
// 支持多种数据来源模式
//
// 设计模式:
//   - Literals: 字面量数据，适合简单配置
//   - Envs: ENV 格式文件，适合键值对
//   - Files: 完整文件内容，适合复杂配置
//
// 配置示例:
//
//	configmaps:
//	  literals:
//	    - name: my-config
//	      files:
//	        - config.yaml
//	  envs:
//	    - name: my-envs
//	      files:
//	        - app.env
type Generator struct {
	// Literals 字面量数据生成
	Literals []GeneratorArgs `json:"literals,omitempty" yaml:"literals,omitempty"`

	// Envs ENV 格式文件生成
	Envs []GeneratorArgs `json:"envs,omitempty" yaml:"envs,omitempty"`

	// Files 完整文件内容生成
	Files []GeneratorArgs `json:"files,omitempty" yaml:"files,omitempty"`
}

// GeneratorArgs 定义生成器的参数
// 用于指定数据源文件和生成配置
type GeneratorArgs struct {
	// Name 生成的 ConfigMap/Secret 名称
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Files 数据源文件路径
	Files []string `json:"files,omitempty" yaml:"files,omitempty"`

	// Type Secret 类型，如 Opaque, kubernetes.io/tls 等
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
}

// ContainerProbes 定义容器的健康检查探针
//
// 支持三种探针类型:
//   - Liveness: 存活探针，检查容器是否存活
//   - Readiness: 就绪探针，检查容器是否就绪
//   - Startup: 启动探针，检查应用是否启动完成
//
// 探针动作格式:
//
//	liveness:
//	  action: http://:8080/healthy   # HTTP 检查
//	  action: tcp://:3306           # TCP 检查
//	  action: cat /tmp/healthy       # 命令检查
type ContainerProbes struct {
	// Liveness 存活探针
	Liveness *ContainerProbe `json:"liveness,omitempty" yaml:"liveness,omitempty"`

	// Readiness 就绪探针
	Readiness *ContainerProbe `json:"readiness,omitempty" yaml:"readiness,omitempty"`

	// Startup 启动探针
	Startup *ContainerProbe `json:"startup,omitempty" yaml:"startup,omitempty"`
}

// ContainerProbe 定义单个探针的配置
type ContainerProbe struct {
	ProbeHandler `json:",inline" yaml:",inline"`

	// InitialDelaySeconds 容器启动后首次检查的延迟时间（秒）
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty" yaml:"initialDelaySeconds,omitempty"`

	// TimeoutSeconds 检查超时时间（秒）
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`

	// PeriodSeconds 检查间隔时间（秒）
	PeriodSeconds int32 `json:"periodSeconds,omitempty" yaml:"periodSeconds,omitempty"`

	// SuccessThreshold 连续成功次数视为健康
	SuccessThreshold int32 `json:"successThreshold,omitempty" yaml:"successThreshold,omitempty"`

	// FailureThreshold 连续失败次数视为不健康
	FailureThreshold int32 `json:"failureThreshold,omitempty" yaml:"failureThreshold,omitempty"`

	// TerminationGracePeriodSeconds 优雅终止超时时间（秒）
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty" yaml:"terminationGracePeriodSeconds,omitempty"`
}

// ProbeHandler 定义探针的动作处理
//
// 支持三种动作类型:
//
//	HTTP:  http://host:port/path
//	TCP:   tcp://host:port
//	EXEC:  command args...
type ProbeHandler struct {
	// Action 探针动作
	// 格式根据类型不同而不同
	Action string `json:"action,omitempty" yaml:"action,omitempty"`

	// Headers HTTP 请求头
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// Outputs 定义需要输出的 K8s 资源类型
//
// 配置示例:
//
//	outputs:
//	  ingress: true
//	  configmaps: true
//	  secrets: true
//
// 默认输出: Deployment, Service
type Outputs struct {
	// Ingress 是否输出 Ingress 资源
	Ingress bool `json:"ingress,omitempty" yaml:"ingress,omitempty"`

	// ConfigMaps 是否输出 ConfigMap 生成器
	ConfigMaps bool `json:"configmaps,omitempty" yaml:"configmaps,omitempty"`

	// Secrets 是否输出 Secret 生成器
	Secrets bool `json:"secrets,omitempty" yaml:"secrets,omitempty"`
}

// configured 检查是否显式配置了 Outputs
func (o *Outputs) configured() bool {
	return o != nil && (o.Ingress || o.ConfigMaps || o.Secrets)
}

// Affinity 亲和性配置
//
// 支持三种亲和性:
//   - PodAntiAffinity: Pod 反亲和，避免 Pod 调度到同一节点
//   - PodAffinity: Pod 亲和，将 Pod 调度到特定 Pod 所在节点
//   - NodeAffinity: 节点亲和，控制 Pod 调度到特定节点
//
// 示例 - Pod 反亲和（打散 Pod 到不同节点）:
//
//	affinity:
//	  podAntiAffinity:
//	    requiredDuringSchedulingIgnoredDuringExecution:
//	      - labelSelector:
//	          matchLabels:
//	            app: my-app
//	        topologyKey: kubernetes.io/hostname
type Affinity struct {
	// PodAntiAffinity Pod 反亲和
	// 用于将 Pod 打散到不同节点
	PodAntiAffinity *PodAntiAffinity `json:"podAntiAffinity,omitempty" yaml:"podAntiAffinity,omitempty"`

	// PodAffinity Pod 亲和
	PodAffinity *PodAffinity `json:"podAffinity,omitempty" yaml:"podAffinity,omitempty"`

	// NodeAffinity 节点亲和
	NodeAffinity *NodeAffinity `json:"nodeAffinity,omitempty" yaml:"nodeAffinity,omitempty"`
}

// PodAntiAffinity Pod 反亲和配置
type PodAntiAffinity struct {
	// RequiredDuringSchedulingIgnoredDuringExecution 硬限制
	// 必须满足的条件，不满足则不调度
	RequiredDuringSchedulingIgnoredDuringExecution []PodAffinityTerm `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`

	// PreferredDuringSchedulingIgnoredDuringExecution 软限制
	// 优先满足的条件，不满足仍可调度
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// PodAffinity Pod 亲和配置
type PodAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []PodAffinityTerm         `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// NodeAffinity 节点亲和配置
type NodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  *NodeSelector             `json:"requiredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"requiredDuringSchedulingIgnoredDuringExecution,omitempty"`
	PreferredDuringSchedulingIgnoredDuringExecution []PreferredSchedulingTerm `json:"preferredDuringSchedulingIgnoredDuringExecution,omitempty" yaml:"preferredDuringSchedulingIgnoredDuringExecution,omitempty"`
}

// PodAffinityTerm Pod 亲和条件
type PodAffinityTerm struct {
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
	TopologyKey   string                `json:"topologyKey,omitempty" yaml:"topologyKey,omitempty"`
	Namespaces    []string              `json:"namespaces,omitempty" yaml:"namespaces,omitempty"`
}

// WeightedPodAffinityTerm 加权 Pod 亲和条件
type WeightedPodAffinityTerm struct {
	Weight          int32           `json:"weight,omitempty" yaml:"weight,omitempty"`
	PodAffinityTerm PodAffinityTerm `json:"podAffinityTerm,omitempty" yaml:"podAffinityTerm,omitempty"`
}

// NodeSelector 节点选择器
type NodeSelector struct {
	NodeSelectorTerms []NodeSelectorTerm `json:"nodeSelectorTerms,omitempty" yaml:"nodeSelectorTerms,omitempty"`
}

// NodeSelectorTerm 节点选择条件
type NodeSelectorTerm struct {
	MatchExpressions []NodeSelectorRequirement `json:"matchExpressions,omitempty" yaml:"matchExpressions,omitempty"`
	MatchFields      []NodeSelectorRequirement `json:"matchFields,omitempty" yaml:"matchFields,omitempty"`
}

// NodeSelectorRequirement 节点选择要求
type NodeSelectorRequirement struct {
	Key      string   `json:"key" yaml:"key"`
	Operator string   `json:"operator" yaml:"operator"`
	Values   []string `json:"values,omitempty" yaml:"values,omitempty"`
}

// PreferredSchedulingTerm 优先调度条件
type PreferredSchedulingTerm struct {
	Weight     int32            `json:"weight,omitempty" yaml:"weight,omitempty"`
	Preference NodeSelectorTerm `json:"preference,omitempty" yaml:"preference,omitempty"`
}

// ToK8s 将 kustz Affinity 转换为 K8s corev1.Affinity
func (a *Affinity) ToK8s() *corev1.Affinity {
	if a == nil {
		return nil
	}

	aff := &corev1.Affinity{}

	if a.PodAntiAffinity != nil {
		aff.PodAntiAffinity = a.PodAntiAffinity.ToK8s()
	}

	if a.PodAffinity != nil {
		aff.PodAffinity = a.PodAffinity.ToK8s()
	}

	if a.NodeAffinity != nil {
		aff.NodeAffinity = a.NodeAffinity.ToK8s()
	}

	return aff
}

// ToK8s 将 PodAntiAffinity 转换为 K8s 结构
func (pa *PodAntiAffinity) ToK8s() *corev1.PodAntiAffinity {
	if pa == nil {
		return nil
	}

	return &corev1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution:  podAffinityTermsToK8s(pa.RequiredDuringSchedulingIgnoredDuringExecution),
		PreferredDuringSchedulingIgnoredDuringExecution: weightedPodAffinityTermsToK8s(pa.PreferredDuringSchedulingIgnoredDuringExecution),
	}
}

// ToK8s 将 PodAffinity 转换为 K8s 结构
func (pa *PodAffinity) ToK8s() *corev1.PodAffinity {
	if pa == nil {
		return nil
	}

	return &corev1.PodAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution:  podAffinityTermsToK8s(pa.RequiredDuringSchedulingIgnoredDuringExecution),
		PreferredDuringSchedulingIgnoredDuringExecution: weightedPodAffinityTermsToK8s(pa.PreferredDuringSchedulingIgnoredDuringExecution),
	}
}

// ToK8s 将 NodeAffinity 转换为 K8s 结构
func (na *NodeAffinity) ToK8s() *corev1.NodeAffinity {
	if na == nil {
		return nil
	}

	aff := &corev1.NodeAffinity{}

	if na.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		aff.RequiredDuringSchedulingIgnoredDuringExecution = nodeSelectorToK8s(na.RequiredDuringSchedulingIgnoredDuringExecution)
	}

	for _, p := range na.PreferredDuringSchedulingIgnoredDuringExecution {
		aff.PreferredDuringSchedulingIgnoredDuringExecution = append(aff.PreferredDuringSchedulingIgnoredDuringExecution, corev1.PreferredSchedulingTerm{
			Weight:     p.Weight,
			Preference: nodeSelectorTermToK8s(p.Preference),
		})
	}

	return aff
}

func podAffinityTermsToK8s(terms []PodAffinityTerm) []corev1.PodAffinityTerm {
	var result []corev1.PodAffinityTerm
	for _, t := range terms {
		result = append(result, corev1.PodAffinityTerm{
			LabelSelector: t.LabelSelector,
			TopologyKey:   t.TopologyKey,
			Namespaces:    t.Namespaces,
		})
	}
	return result
}

func weightedPodAffinityTermsToK8s(terms []WeightedPodAffinityTerm) []corev1.WeightedPodAffinityTerm {
	var result []corev1.WeightedPodAffinityTerm
	for _, t := range terms {
		result = append(result, corev1.WeightedPodAffinityTerm{
			Weight: t.Weight,
			PodAffinityTerm: corev1.PodAffinityTerm{
				LabelSelector: t.PodAffinityTerm.LabelSelector,
				TopologyKey:   t.PodAffinityTerm.TopologyKey,
				Namespaces:    t.PodAffinityTerm.Namespaces,
			},
		})
	}
	return result
}

func nodeSelectorToK8s(ns *NodeSelector) *corev1.NodeSelector {
	if ns == nil {
		return nil
	}

	selector := &corev1.NodeSelector{
		NodeSelectorTerms: []corev1.NodeSelectorTerm{},
	}

	for _, term := range ns.NodeSelectorTerms {
		selector.NodeSelectorTerms = append(selector.NodeSelectorTerms, nodeSelectorTermToK8s(term))
	}

	return selector
}

func nodeSelectorTermToK8s(term NodeSelectorTerm) corev1.NodeSelectorTerm {
	return corev1.NodeSelectorTerm{
		MatchExpressions: nodeSelectorRequirementsToK8s(term.MatchExpressions),
		MatchFields:      nodeSelectorRequirementsToK8s(term.MatchFields),
	}
}

func nodeSelectorRequirementsToK8s(reqs []NodeSelectorRequirement) []corev1.NodeSelectorRequirement {
	var result []corev1.NodeSelectorRequirement
	for _, r := range reqs {
		result = append(result, corev1.NodeSelectorRequirement{
			Key:      r.Key,
			Operator: corev1.NodeSelectorOperator(r.Operator),
			Values:   r.Values,
		})
	}
	return result
}
