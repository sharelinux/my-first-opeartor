package v1

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"

	modelv1 "github.com/sharelinuxs/my-first-opeartor/api/v1"
)

type ModelBoxInterface interface {
	Create(mb *modelv1.ModelBox, opts metav1.CreateOptions) (*modelv1.ModelBox, error)
	Delete(name string, opts metav1.DeleteOptions) error
	Get(name string, options metav1.GetOptions) (*modelv1.ModelBox, error)
	List(opts metav1.ListOptions) (*modelv1.ModelBoxList, error)
	Watch(opts metav1.ListOptions) (watch.Interface, error)
	// ...
}

type modelBoxClient struct {
	restClient rest.Interface
	ns         string
	crScheme   *runtime.Scheme
}

func (c *modelBoxClient) List(opts metav1.ListOptions) (*modelv1.ModelBoxList, error) {
	ctx := context.Background()
	result := modelv1.ModelBoxList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource("modelboxes").
		// VersionedParams(&opts, scheme.ParameterCodec).   // 注释掉正常
		VersionedParams(&opts, runtime.NewParameterCodec(c.crScheme)).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *modelBoxClient) Get(name string, opts metav1.GetOptions) (*modelv1.ModelBox, error) {
	ctx := context.Background()
	result := modelv1.ModelBox{}
	err := c.restClient.Get().
		Namespace(c.ns).
		Resource("modelboxes").
		Name(name).
		// VersionedParams(&opts, scheme.ParameterCodec).   // 注释掉正常
		VersionedParams(&opts, runtime.NewParameterCodec(c.crScheme)).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *modelBoxClient) Create(modelbox *modelv1.ModelBox, opts metav1.CreateOptions) (result *modelv1.ModelBox, err error) {
	ctx := context.Background()
	result = &modelv1.ModelBox{}
	err = c.restClient.
		Post().
		Namespace(c.ns).
		Resource("modelboxes").
		// VersionedParams(&opts, scheme.ParameterCodec).   // 注释掉正常
		VersionedParams(&opts, runtime.NewParameterCodec(c.crScheme)).
		Body(modelbox).
		Do(ctx).
		Into(result)

	return
}

func (c *modelBoxClient) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	ctx := context.Background()
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.restClient.
		Get().
		Namespace(c.ns).
		Resource("modelboxes").
		// VersionedParams(&opts, scheme.ParameterCodec).   // 注释掉正常
		VersionedParams(&opts, runtime.NewParameterCodec(c.crScheme)).
		Timeout(timeout).
		Watch(ctx)
}

func (c *modelBoxClient) Delete(name string, opts metav1.DeleteOptions) error {
	ctx := context.Background()
	return c.restClient.
		Delete().
		Namespace(c.ns).
		Resource("modelboxes").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}
