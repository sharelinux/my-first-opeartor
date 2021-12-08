package v1

import (
	modelboxv1 "github.com/sharelinuxs/my-first-opeartor/api/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
)

var SchemeBuilder = runtime.SchemeBuilder{
	modelboxv1.AddToScheme,
}

func newCRScheme(schemes ...func(scheme *runtime.Scheme) error) (*runtime.Scheme, error) {
	sc := runtime.NewScheme()
	schemeBuilder := &runtime.SchemeBuilder{}

	for _, s := range schemes {
		schemeBuilder.Register(s)
	}
	if err := schemeBuilder.AddToScheme(sc); err != nil {
		logrus.Errorf("failed to add scheme, err: %v", err)
		return nil, err
	}
	return sc, nil
}
