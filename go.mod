module github.com/rkojedzinszky/thermo-center-controller

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	golang.org/x/mod v0.4.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.0
)

replace github.com/go-logr/zapr => github.com/go-logr/zapr v0.2.0
