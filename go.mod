module github.com/rkojedzinszky/thermo-center-controller

go 1.15

require (
	github.com/go-logr/logr v0.2.1
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	golang.org/x/mod v0.1.1-0.20191105210325-c90efee705ee
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.6.3
)

replace github.com/go-logr/zapr => github.com/go-logr/zapr v0.2.0
