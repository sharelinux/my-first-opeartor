package main

import (
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/client-go/kubernetes/scheme"

	modelboxv1 "github.com/sharelinuxs/my-first-opeartor/api/v1"
)

var SchemeBuilder = runtime.SchemeBuilder{
	modelboxv1.AddToScheme,
	k8sschema.AddToScheme,
}
