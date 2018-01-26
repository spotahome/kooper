package crd_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionscli "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubetesting "k8s.io/client-go/testing"

	"github.com/spotahome/kooper/client/crd"
	"github.com/spotahome/kooper/log"
)

func TestCRDEnsurePresent(t *testing.T) {
	tests := []struct {
		name   string
		crd    crd.Conf
		expCrd *apiextensionsv1beta1.CustomResourceDefinition
		retErr error
		expErr bool
	}{
		{
			name: "Creating a non existe CRD should create a crd without error",
			crd: crd.Conf{
				Kind:       "Test",
				NamePlural: "tests",
				Scope:      crd.ClusterScoped,
				Group:      "toilettesting",
				Version:    "v99",
			},
			expCrd: &apiextensionsv1beta1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tests.toilettesting",
				},
				Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
					Group:   "toilettesting",
					Version: "v99",
					Scope:   crd.ClusterScoped,
					Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
						Plural: "tests",
						Kind:   "Test",
					},
				},
			},
			retErr: nil,
			expErr: false,
		},
		{
			name: "Creating a CRD that errors should return an error.",
			crd: crd.Conf{
				Kind:       "Test",
				NamePlural: "tests",
				Scope:      crd.ClusterScoped,
				Group:      "toilettesting",
				Version:    "v99",
			},
			expCrd: nil,
			retErr: errors.New("wanted error"),
			expErr: true,
		},
		{
			name: "Creating a CRD that exists shouldn't return an error.",
			crd: crd.Conf{
				Kind:       "Test",
				NamePlural: "tests",
				Scope:      crd.ClusterScoped,
				Group:      "toilettesting",
				Version:    "v99",
			},
			expCrd: &apiextensionsv1beta1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tests.toilettesting",
				},
				Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
					Group:   "toilettesting",
					Version: "v99",
					Scope:   crd.ClusterScoped,
					Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
						Plural: "tests",
						Kind:   "Test",
					},
				},
			},
			retErr: kubeerrors.NewAlreadyExists(schema.GroupResource{}, ""),
			expErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Mock calls
			cli := &apiextensionscli.Clientset{}
			cli.AddReactor("create", "customresourcedefinitions", func(action kubetesting.Action) (bool, runtime.Object, error) {
				return true, nil, test.retErr
			})

			crdCli := crd.NewClient(cli, log.Dummy)
			err := crdCli.EnsurePresent(test.crd)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				// Check the client calls.
				actions := cli.Actions()
				if assert.Len(actions, 1) {
					createAction, ok := actions[0].(kubetesting.CreateActionImpl)
					if assert.True(ok, "the action should be a creation") {
						assert.Equal(test.expCrd, createAction.GetObject())
					}
				}
			}

		})
	}
}
