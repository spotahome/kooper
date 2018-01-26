package crd

import (
	"fmt"
	"time"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionscli "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/spotahome/kooper/log"
)

// Scope is the scope of a CRD.
type Scope = apiextensionsv1beta1.ResourceScope

const (
	// ClusterScoped represents a type of a cluster scoped CRD.
	ClusterScoped = apiextensionsv1beta1.ClusterScoped
	// NamespaceScoped represents a type of a namespaced scoped CRD.
	NamespaceScoped = apiextensionsv1beta1.NamespaceScoped
)

// Conf is the configuration required to create a CRD
type Conf struct {
	Kind       string
	NamePlural string
	Group      string
	Version    string
	Scope      Scope
}

// Interface is the CRD client that knows how to interact with k8s to manage them.
type Interface interface {
	// EnsureCreated will ensure the the CRD is present.
	EnsurePresent(conf Conf) error
	// WaitToBePresent will wait until the CRD is present.
	WaitToBePresent(conf Conf, timeout time.Duration) error
	// Delete will delete the CRD.
	Delete(conf Conf) error
}

// Client is the CRD client implementation using API calls to kubernetes.
type Client struct {
	aeClient apiextensionscli.Interface
	logger   log.Logger
}

// NewClient returns a new CRD client.
func NewClient(aeClient apiextensionscli.Interface, logger log.Logger) *Client {
	return &Client{
		aeClient: aeClient,
		logger:   logger,
	}
}

// EnsurePresent satisfies crd.Interface.
func (c *Client) EnsurePresent(conf Conf) error {
	// TODO: Check version of cluster equal or greater than 1.7
	crdName := fmt.Sprintf("%s.%s", conf.NamePlural, conf.Group)

	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   conf.Group,
			Version: conf.Version,
			Scope:   conf.Scope,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: conf.NamePlural,
				Kind:   conf.Kind,
			},
		},
	}

	_, err := c.aeClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("error creating crd %s: %s", crdName, err)
		}
		return nil
	}
	c.logger.Infof("crd %s created", crdName)
	// TODO: wait to be present.
	return nil
}

// WaitToBePresent satisfies crd.Interface.
func (c *Client) WaitToBePresent(conf Conf, timeout time.Duration) error {
	return fmt.Errorf("Not implemented")
}

// Delete satisfies crd.Interface.
func (c *Client) Delete(conf Conf) error {
	return fmt.Errorf("Not implemented")
}
