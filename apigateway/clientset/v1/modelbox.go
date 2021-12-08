package v1

import (
	"context"
	modelv1 "github.com/sharelinuxs/my-first-opeartor/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type ModelBoxInterface interface {
	List(opts metav1.ListOptions) (*modelv1.ModelBoxList, error)
	Get(name string, options metav1.GetOptions) (*modelv1.ModelBox, error)
	Create(*modelv1.ModelBox) (*modelv1.ModelBox, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	// ...
}

type modelBoxClient struct {
	restClient rest.Interface
	ns         string
}

func (c *modelBoxClient) List(opts metav1.ListOptions) (*modelv1.ModelBoxList, error) {
	ctx := context.Background()
	result := modelv1.ModelBoxList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("modelboxes").
		VersionedParams(&opts, scheme.ParameterCodec).
		//VersionedParams(&opts, runtime.NewParameterCodec(scheme.Scheme)).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *modelBoxClient) Get(name string, opts metav1.GetOptions) (*modelv1.ModelBox, error) {
	ctx := context.Background()
	result := modelv1.ModelBox{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("modelboxes").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *modelBoxClient) Create(project *modelv1.ModelBox) (*modelv1.ModelBox, error) {
	ctx := context.Background()
	result := modelv1.ModelBox{}
	err := c.restClient.
		Post().
		Namespace(c.ns).
		Resource("modelboxes").
		Body(project).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *modelBoxClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	ctx := context.Background()
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.ns).
		Resource("modelboxes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}
