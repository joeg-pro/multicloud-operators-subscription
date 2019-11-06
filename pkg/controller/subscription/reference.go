// Copyright 2019 The Kubernetes Authors.
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

package subscription

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha1 "github.com/IBM/multicloud-operators-subscription/pkg/apis/app/v1alpha1"
)

var SecretKindStr = "Secret"
var ConfigMapKindStr = "ConfigMap"

//SercertReferredMarker is used as a label key to filter out the secert coming from reference
var SercertReferredMarker = "IsReferredBySub"

type referredObject interface {
	GetObjectKind() schema.ObjectKind
	DeepCopyObject() runtime.Object

	GetName() string
	SetName(name string)
	SetNamespace(namespace string)
	SetUID(uid types.UID)
	SetResourceVersion(version string)
	GetLabels() map[string]string
	SetLabels(labels map[string]string)
	// GetAnnotations() map[string]string
	SetAnnotations(annotations map[string]string)
}

//ListAndDeployReferredObject handles the create/update reconciler request
// the idea is, first it will try to get the referred secret from the subscription namespace
// if it can't find it,
////it could be it's a brand new secret request or it's trying to use a differenet one.
//// to address these, we will try to list the sercert within the subscription namespace with the subscription label.
//// if we are seeing these secret, we will delete the label of the reconciled subscription.
///// then we will create a new secret and label it
// if we can find a secret at the subscription namespace, it means there must be some other subscription is
// using it. In this case, we will just add an extra label to it

func (r *ReconcileSubscription) ListAndDeployReferredObject(instance *appv1alpha1.Subscription, gvk schema.GroupVersionKind, refObj referredObject) error {
	insName := instance.GetName()
	insNs := instance.GetNamespace()
	uObjList := &unstructured.UnstructuredList{}

	uObjList.SetGroupVersionKind(gvk)

	opts := &client.ListOptions{Namespace: insNs}
	err := r.Client.List(context.TODO(), uObjList, opts)

	if err != nil {
		klog.Errorf("Failed to list referred objects with error %v ", err)
		return err
	}

	found := false

	for _, obj := range uObjList.Items {
		u := obj.DeepCopy()
		lb := u.GetLabels()

		if len(lb) == 0 {
			lb = make(map[string]string)
		}

		if u.GetName() == refObj.GetName() {
			found = true

			lb[SercertReferredMarker] = "true"
			lb[insName] = "true"
			u.SetLabels(lb)

			err := r.Client.Update(context.TODO(), u)
			if err != nil {
				return err
			}

			continue
		}

		if lb[SercertReferredMarker] == "true" && lb[insName] == "true" {
			delete(lb, insName)

			if len(lb) >= 2 {
				u.SetLabels(lb)

				err := r.Client.Update(context.TODO(), u)
				if err != nil {
					return err
				}
			} else {
				err := r.Client.Delete(context.TODO(), u)
				if err != nil {
					return err
				}
			}
		}
	}

	if !found {
		lb := refObj.GetLabels()

		if len(lb) == 0 {
			lb = make(map[string]string)
		}

		t := types.UID("")

		lb[SercertReferredMarker] = "true"
		lb[insName] = "true"
		refObj.SetLabels(lb)
		refObj.SetNamespace(insNs)
		refObj.SetResourceVersion("")
		refObj.SetUID(t)

		err := r.Client.Create(context.TODO(), refObj)

		if err != nil {
			klog.Errorf("Got error %v, while creating referred object %v for subscription %v", err, refObj.GetName(), insName)
		}
	}

	return nil
}

func (r *ReconcileSubscription) DeleteReferredObjects(rq types.NamespacedName, gvk schema.GroupVersionKind) error {
	selector := &metav1.LabelSelector{MatchLabels: map[string]string{SercertReferredMarker: "true", rq.Name: "true"}}
	ls, _ := metav1.LabelSelectorAsSelector(selector)
	opts := &client.ListOptions{
		Namespace:     rq.Namespace,
		LabelSelector: ls,
	}
	uObjList := &unstructured.UnstructuredList{}

	uObjList.SetGroupVersionKind(gvk)

	err := r.Client.List(context.TODO(), uObjList, opts)

	if err != nil {
		return err
	}

	if len(uObjList.Items) == 0 {
		return nil
	}

	for _, obj := range uObjList.Items {
		u := obj.DeepCopy()
		lb := u.GetLabels()

		delete(lb, rq.Name)

		if len(lb) < 2 {
			err := r.Client.Delete(context.TODO(), u)
			if err != nil {
				return nil
			}
		} else {
			u.SetLabels(lb)
			err := r.Client.Update(context.TODO(), u)
			if err != nil {
				return nil
			}
		}
	}

	return nil
}