package v1

import (
	modelv1 "github.com/sharelinuxs/my-first-opeartor/api/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type ModelBoxV1Alpha1Interface interface {
	ModelBoxes(namespace string) ModelBoxInterface
}

type ModelBoxV1Alpha1Client struct {
	restClient rest.Interface
}

func NewForConfig(c *rest.Config) (*ModelBoxV1Alpha1Client, error) {

	//crdConfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	config := *c
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: modelv1.GroupName, Version: modelv1.Version}
	config.APIPath = "/apis"
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	// 该类型目前不存在 DirectCodecFactory
	//config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	//client, err := rest.UnversionedRESTClientFor(&config)
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &ModelBoxV1Alpha1Client{restClient: client}, nil
}

func (c *ModelBoxV1Alpha1Client) ModelBoxes(namespace string) ModelBoxInterface {
	crScheme, err := newCRScheme(SchemeBuilder...)
	if err != nil {
		klog.Errorf("ModelBoxes newCRScheme : %v", err)
	}
	return &modelBoxClient{
		restClient: c.restClient,
		ns:         namespace,
		crScheme:   crScheme,
	}
}
