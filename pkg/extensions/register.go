package extensions

import (
		"strings"

		meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
		"k8s.io/client-go/kubernetes"
		"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func EnsureThirdPartyResourcesExist(ki kubernetes.Interface) error {
		if err := ensureThirdPartyResource(ki, "agents"); err != nil {
				return err
		}

		return nil
}

func RemoveThirdPartyResources(ki kubernetes.Interface) {
		fullName := strings.Join([]string{"agents", GroupName}, ".")
		ki.Extensions().ThirdPartyResources().Delete(fullName, &meta_v1.DeleteOptions{})
}

func ensureThirdPartyResource(ki kubernetes.Interface, name string) error {
		fullName := strings.Join([]string{name, GroupName}, ".")
		_, err := ki.Extensions().ThirdPartyResources().Get(fullName, meta_v1.GetOptions{})
		if err == nil {
				return nil
		}

		resource := &v1beta1.ThirdPartyResource{
				Versions: []v1beta1.APIVersion{
						{Name: Version},
				}}
		resource.SetName(fullName)
		_, err = ki.Extensions().ThirdPartyResources().Create(resource)

		return err
}
