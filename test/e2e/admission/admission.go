package admission

import (
	"context"
	"fmt"
	"strings"

	"github.com/gjkim42/default-imagepullsecrets/test/e2e/framework"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var _ = ginkgo.Describe("Admission controller", func() {
	f := framework.NewDefaultFramework("admission")

	var ns string
	var client kubernetes.Interface
	var imagePullSecrets []string

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		client = f.ClientSet
		imagePullSecrets = strings.Split(framework.TestContext.ImagePullSecrets, ",")
	})

	ginkgo.AfterEach(func() {
	})

	ginkgo.It("should applies default imagePullSecrets to pods", func() {
		ginkgo.By("creating a pod")
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pod",
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "nginx",
						Image: "nginx",
					},
				},
			},
		}
		_, err := client.CoreV1().Pods(ns).Create(context.TODO(), pod, metav1.CreateOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		ginkgo.By("checking if the pod has the default imagePullSecrets")
		pod, err = client.CoreV1().Pods(ns).Get(context.TODO(), pod.Name, metav1.GetOptions{})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = checkImagePullSecretsExist(pod, imagePullSecrets)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})
})

func checkImagePullSecretsExist(pod *v1.Pod, imagePullSecrets []string) error {
	m := make(map[string]struct{})
	for _, s := range pod.Spec.ImagePullSecrets {
		m[s.Name] = struct{}{}
	}

	for _, s := range imagePullSecrets {
		if _, ok := m[s]; !ok {
			return fmt.Errorf("imagepullsecret %s is not applied to pod %s/%s, imagePullSecrets: %v", s, pod.Namespace, pod.Name, pod.Spec.ImagePullSecrets)
		}
	}
	return nil
}
