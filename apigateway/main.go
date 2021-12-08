package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"os"
	"path/filepath"

	//corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//"sigs.k8s.io/controller-runtime/pkg/client/config"

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	modelv1 "github.com/sharelinuxs/my-first-opeartor/api/v1"
	clientsetv1 "github.com/sharelinuxs/my-first-opeartor/apigateway/clientset/v1"
)

// 第一种调用方式无异常，可以获取创建的CR资源列表
func main0() {
	ctx := context.Background()
	var err error
	var config *rest.Config
	// inCluster (Pod)、kubeconfig (kubectl)
	var kubeconfig *string

	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String(
			"kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// 使用ServiceAccount创建集群配置(InCluster模式) 需要去配置对应的RBAC权限， 默认的sa是default没有获取deployments的list权限
	if config, err = rest.InClusterConfig(); err != nil {
		// 使用 kubeconfig 文件来创建集群配置
		if config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig); err != nil {
			panic(err.Error())
		}
	}

	err = modelv1.AddToScheme(scheme.Scheme)
	if err != nil {
		fmt.Printf("modelv1.AddToScheme : %v", err)
	}

	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Group: modelv1.GroupName, Version: modelv1.Version}
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	//crdConfig.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	crdRestClient, err := rest.UnversionedRESTClientFor(&crdConfig)
	if err != nil {
		panic(err)
	}

	result := modelv1.ModelBoxList{}
	err = crdRestClient.
		Get().
		Resource("ModelBoxes").
		Do(ctx).
		Into(&result)

	if err != nil {
		fmt.Println("crdRestClient err:", err)
	}

	for _, item := range result.Items {
		fmt.Printf("Name: %s Image: %s\n", item.Name, item.Spec.Image)
	}
}

func main() {
	//ctx := context.Background()
	var err error
	var config *rest.Config
	// inCluster (Pod)、kubeconfig (kubectl)
	var kubeconfig *string

	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String(
			"kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// 使用ServiceAccount创建集群配置(InCluster模式) 需要去配置对应的RBAC权限， 默认的sa是default没有获取deployments的list权限
	if config, err = rest.InClusterConfig(); err != nil {
		// 使用 kubeconfig 文件来创建集群配置
		if config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig); err != nil {
			panic(err.Error())
		}
	}
	// TODO: List CR 优雅方式
	crdClientSet, err := clientsetv1.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	modelBoxes, err := crdClientSet.ModelBoxes("default").List(metav1.ListOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Printf("modelBoxes found: %+v\n", modelBoxes)

	cl, err := client.New(config, client.Options{})
	if err != nil {
		fmt.Println("failed to create client")
		os.Exit(1)
	}

	// TODO: List CR 非优雅方式
	//modelBoxList := &modelv1.ModelBoxList{}
	//
	//err = cl.List(context.Background(), modelBoxList, client.InNamespace("default"))
	//if err != nil {
	//	fmt.Printf("failed to list modelBox in namespace default: %v\n", err)
	//	os.Exit(1)
	//}

	//u := &unstructured.UnstructuredList{}
	//u.SetGroupVersionKind(schema.GroupVersionKind{
	//	Group:   "model.github.com",
	//	Kind:    "ModelBoxList",
	//	Version: "v1",
	//})
	//_ = cl.List(context.Background(), u, client.InNamespace("default"))
	//for _, item := range u.Items {
	//	fmt.Printf("item: %v\n", item)
	//}

	// TODO: 创建CR
	aStr := `{
    "apiVersion": "model.github.com/v1",
    "kind": "ModelBox",
    "metadata": {
       "name":"modelbox-sample0",
       "namespace":"default"
    },
    "spec": {
       "image": "nginx:1.7.9",
       "modelFileURL": "https://model-management.s3.cn-north-1.aws.com/model/example-model.zip",
       "name": "nginx",
       "ports": [{
           "name": "app-port",
           "port": 80,
           "targetPort": 80
       }],
       "replicas": 1,
       "resourceType": "small",
       "rollingUpdate": "30%",
       "serviceType": "ClusterIP"
    	}
	}`
	m := make(map[string]interface{})
	err = json.Unmarshal([]byte(aStr), &m)

	u1 := &unstructured.Unstructured{Object: m}

	u1.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "model.github.com",
		Kind:    "ModelBox",
		Version: "v1",
	})
	err = cl.Create(context.Background(), u1)
	if err != nil {
		fmt.Printf("cl.Create: %v\n", err)
	}

	// modified
	mb := modelv1.ModelBox{}
	crCli, err := NewCRClient(config, SchemeBuilder...)
	if err != nil {
		panic(err)
	}

	// TODO: operator with controller runtime client
	if err := crCli.Get(context.Background(), client.ObjectKey{
		Namespace: metav1.NamespaceDefault,
		Name:      "modelBox's name",
	}, &mb); err != nil {
		if k8serrors.IsNotFound(err) {
			logrus.Errorf("resource not found")
			return
		}
		panic(err)
	}

	// crCli.List()
	// crCli.Create()
	// crCli.Delete()

}

func NewCRClient(rc *rest.Config, schemes ...func(scheme *runtime.Scheme) error) (client.Client, error) {
	if rc == nil {
		return nil, fmt.Errorf("failed to get rest.Config")
	}
	sc := runtime.NewScheme()
	schemeBuilder := &runtime.SchemeBuilder{}

	for _, s := range schemes {
		schemeBuilder.Register(s)
	}

	if err := schemeBuilder.AddToScheme(sc); err != nil {
		logrus.Errorf("failed to add scheme, err: %v", err)
		return nil, err
	}

	return client.New(rc, client.Options{Scheme: sc})
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func int32Ptr(i int32) *int32 { return &i }
