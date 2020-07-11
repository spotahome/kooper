module github.com/spotahome/kooper/examples/pod-terminator-operator/v2

go 1.14

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.3+incompatible
	github.com/spotahome/kooper/v2 => ../../
)

require (
	github.com/sirupsen/logrus v1.6.0
	github.com/spotahome/kooper/v2 v2.0.0-rc.1
	github.com/stretchr/testify v1.5.1
	k8s.io/api v0.17.4
	k8s.io/apiextensions-apiserver v0.15.10
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v0.17.4
)
