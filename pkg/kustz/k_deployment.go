package kustz

import (
	appv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubeDeployment 将 kustz 配置转换为 K8s Deployment 资源
//
// 构建器模式设计:
//   - Config 作为领域模型
//   - KubeDeployment 作为资源构建方法
//   - 保持配置与 K8s 资源对象的解耦
//
// 资源构建流程:
//
//	Config.Service.Replicas  --> Deployment.Spec.Replicas
//	Config.Service.Image     --> PodTemplate.Spec.Containers[0].Image
//	Config.Name              --> Deployment.Name & PodTemplate.Labels["app"]
//	CommonLabels()           --> Selector.MatchLabels & PodTemplate.Labels
func (kz *Config) KubeDeployment() *appv1.Deployment {
	return &appv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   kz.Name,
			Labels: kz.CommonLabels(),
		},
		Spec: appv1.DeploymentSpec{
			Replicas: &kz.Service.Replicas,
			Template: kz.KubePod(),
			Selector: &metav1.LabelSelector{
				MatchLabels: kz.CommonLabels(),
			},
		},
	}
}
