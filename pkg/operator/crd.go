// Copyright 2016 The quartermaster Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operator

import (
	extensionsobj "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TPRGroup   = "storage.coreos.com"
	TPRVersion = "v1alpha1"

	TPRStorageNodeKind    = "StorageNode"
	TPRStorageStatusKind  = "StorageStatus"
	TPRStorageClusterKind = "StorageCluster"

	PluralTPRStorageNodeKind    = "storagenodes"
	PluralTPRStorageStatusKind  = "storagestatuses"
	PluralTPRStorageClusterKind = "storageclusters"
)

var (
	tprPluralList = []string{PluralTPRStorageClusterKind,
		PluralTPRStorageNodeKind,
		PluralTPRStorageStatusKind}
)

func (c *Operator) createTPRs() error {
	tprs := []*extensionsobj.CustomResourceDefinition{
		{
			ObjectMeta: meta.ObjectMeta{
				Name: PluralTPRStorageStatusKind + "." + TPRGroup,
			},
			Spec: extensionsobj.CustomResourceDefinitionSpec{
				Version: TPRVersion,
				Group:   TPRGroup,
				Scope:   extensionsobj.NamespaceScoped,
				Names: extensionsobj.CustomResourceDefinitionNames{
					Plural: PluralTPRStorageStatusKind,
					Kind:   TPRStorageStatusKind,
				},
			},
		},
		{
			ObjectMeta: meta.ObjectMeta{
				Name: PluralTPRStorageNodeKind + "." + TPRGroup,
			},
			Spec: extensionsobj.CustomResourceDefinitionSpec{
				Version: TPRVersion,
				Group:   TPRGroup,
				Scope:   extensionsobj.NamespaceScoped,
				Names: extensionsobj.CustomResourceDefinitionNames{
					Plural: PluralTPRStorageNodeKind,
					Kind:   TPRStorageNodeKind,
				},
			},
		},
		{
			ObjectMeta: meta.ObjectMeta{
				Name: PluralTPRStorageClusterKind + "." + TPRGroup,
			},
			Spec: extensionsobj.CustomResourceDefinitionSpec{
				Version: TPRVersion,
				Group:   TPRGroup,
				Scope:   extensionsobj.NamespaceScoped,
				Names: extensionsobj.CustomResourceDefinitionNames{
					Plural: PluralTPRStorageClusterKind,
					Kind:   TPRStorageClusterKind,
				},
			},
		},
	}
	tprClient := c.crdclient.ApiextensionsV1beta1().CustomResourceDefinitions()
	for _, tpr := range tprs {
		_, err := tprClient.Create(tpr)
		if apierrors.IsAlreadyExists(err) {
			logger.Debug("%v already registered", tpr.GetName())
		} else if err != nil {
			return err
		} else {
			logger.Info("%v CRD created", tpr.GetName())
		}
	}

	// We have to wait for the TPRs to be ready. Otherwise the initial watch may fail.
	for _, pluralTpr := range tprPluralList {
		err := WaitForCRDReady(c.kclient.Core().RESTClient(),
			TPRGroup,
			TPRVersion,
			pluralTpr)
		if err != nil {
			return err
		}
	}

	return nil
}
