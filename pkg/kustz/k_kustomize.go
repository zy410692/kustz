package kustz

import (
	"github.com/tangx/kustz/pkg/tokust"
	"sigs.k8s.io/kustomize/v3/pkg/types"
)

// Kustomization 生成 Kustomize 配置
//
// Kustomize 设计:
//
//	通过 Kustomize 组合多个 K8s 资源文件:
//
//	  1. deployment.yml  - Deployment 资源
//	  2. service.yml     - Service 资源
//	  3. ingress.yml     - Ingress 资源
//
//	同时生成 ConfigMap 和 Secret:
//
//	  - ConfigMapGenerator: 从配置文件生成 ConfigMap
//	  - SecretGenerator:    从配置文件生成 Secret
//
//	GeneratorOptions:
//
//	  - DisableNameSuffixHash: 禁用名称后缀哈希
//	    使生成的配置名称可预测
func (kz *Config) Kustomization() types.Kustomization {
	resources := []string{
		"deployment.yml",
		"service.yml",
	}

	// Ingress: 默认输出(如果有rules)，除非显式配置 outputs.ingress: false
	ingressOutput := kz.Outputs.Ingress || (!kz.Outputs.configured() && len(kz.Ingress.Rules) > 0)
	if ingressOutput {
		resources = append(resources, "ingress.yml")
	}

	k := types.Kustomization{
		TypeMeta: types.TypeMeta{
			Kind:       types.KustomizationKind,
			APIVersion: types.KustomizationVersion,
		},
		Namespace: kz.Namespace,
		Resources: resources,
		GeneratorOptions: &types.GeneratorOptions{
			DisableNameSuffixHash: true,
		},
	}

	// ConfigMaps: 默认输出(如果有配置)，除非显式配置 outputs.configmaps: false
	if kz.Outputs.ConfigMaps || (!kz.Outputs.configured() && kz.hasConfigMaps()) {
		k.ConfigMapGenerator = kz.ConfigMaps.toConfigMapArgs()
	}

	// Secrets: 默认输出(如果有配置)，除非显式配置 outputs.secrets: false
	if kz.Outputs.Secrets || (!kz.Outputs.configured() && kz.hasSecrets()) {
		k.SecretGenerator = kz.Secrets.toSecretArgs()
	}

	return k
}

// toConfigMapArgs 返回 ConfigMap 参数
//
//	生成器模式:
//
//	  literals:
//	    - name: my-config
//	      files:
//	        - config.yaml
//	  envs:
//	    - name: my-envs
//	      files:
//	        - app.env
//	  files:
//	    - name: my-files
//	      files:
//	        - application.properties
func (genor *Generator) toConfigMapArgs() []types.ConfigMapArgs {

	args := []types.ConfigMapArgs{}

	for _, data := range genor.datas() {
		for _, garg := range data.gargs {
			arg := tokust.ConfigMapArgs(garg.Name, garg.Files, data.mode)
			args = append(args, arg)
		}
	}

	return args
}

// toSecretArgs 返回 Secret 参数
//
//	生成器模式:
//
//	  literals:
//	    - name: my-secret
//	      files:
//	        - secret.yaml
//	  type: Opaque  # 默认类型
func (genor *Generator) toSecretArgs() []types.SecretArgs {

	args := []types.SecretArgs{}

	for _, data := range genor.datas() {
		for _, garg := range data.gargs {
			arg := tokust.SecretArgs(garg.Name, garg.Files, garg.Type, data.mode)
			args = append(args, arg)
		}
	}

	return args
}

// GeneratorArgsData 整合生成器数据的内部结构
type GeneratorArgsData struct {
	mode  tokust.GeneratorMode
	gargs []GeneratorArgs
}

// datas 整合生成器数据
//
//	数据来源优先级:
//
//	  1. envs     - ENV 格式文件
//	  2. files    - 完整文件内容
//	  3. literals - 字面量数据
func (genor *Generator) datas() []GeneratorArgsData {
	return []GeneratorArgsData{
		{mode: tokust.GeneratorMode_Envs, gargs: genor.Envs},
		{mode: tokust.GeneratorMode_Files, gargs: genor.Files},
		{mode: tokust.GeneratorMode_Literals, gargs: genor.Literals},
	}
}
