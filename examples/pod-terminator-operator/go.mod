module github.com/spotahome/kooper/examples/pod-terminator-operator

go 1.14

replace github.com/spotahome/kooper => ../../

require (
	github.com/sirupsen/logrus v1.4.2
	github.com/spotahome/kooper v0.0.0
	github.com/stretchr/testify v1.4.0
	k8s.io/api v0.15.10
	k8s.io/apiextensions-apiserver v0.15.10
	k8s.io/apimachinery v0.15.12-beta.0
	k8s.io/client-go v0.15.10
)
