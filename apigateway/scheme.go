package main

import (
	"k8s.io/apimachinery/pkg/runtime"

	modelboxv1 "github.com/sharelinuxs/my-first-opeartor/api/v1"
)

var SchemeBuilder = runtime.SchemeBuilder{
	modelboxv1.AddToScheme,
}
