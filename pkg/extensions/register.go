ckage extensions

import (
	"strings"
	"time"

	"k8s.io/client-go/1.5/kubernetes"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/apis/extensions/v1beta1"
)

func EnsureThirdPartyResourcesExist(ki kubernetes.Interface) error {
	if err := ensureThirdPartyResource(ki, "agents"); err != nil {
		return err
	}

	return nil
}

func RemoveThirdPartyResources(ki kubernetes.Interface) {
	fullName := strings.Join([]string{"agents", GroupName}, ".")
	ki.Extensions().ThirdPartyResources().Delete(fullName, &api.DeleteOptions{})
}

func ensureThirdPartyResource(ki kubernetes.Interface, name string) error {
	fullName := strings.Join([]string{name, GroupName}, ".")
	_, err := ki.Extensions().ThirdPartyResources().Get(fullName)
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

func WaitThirdPartyResources(ext ExtensionsClientset, timeout time.Duration, interval time.Duration) (err error) {
	timeoutChan := time.After(timeout)
	intervalChan := time.Tick(interval)
	for {
		select {
		case <-timeoutChan:
			return err
		case <-intervalChan:
			_, err = ext.Agents().List(api.ListOptions{})
			if err != nil {
				continue
			}
			return nil
		}
	}
}
