/*
Copyright 2025 The OpenYurt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta2

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openyurtio/openyurt/pkg/apis"
	appsv1beta2 "github.com/openyurtio/openyurt/pkg/apis/apps/v1beta2"
	"github.com/openyurtio/openyurt/pkg/projectinfo"
)

func TestValidateCreate(t *testing.T) {
	testcases := map[string]struct {
		pool    runtime.Object
		errcode int
	}{
		"it is a normal nodepool": {
			pool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type:                   appsv1beta2.Edge,
					LeaderElectionStrategy: string(appsv1beta2.ElectionStrategyRandom),
				},
			},
			errcode: 0,
		},
		"it is not a nodepool": {
			pool:    &corev1.Node{},
			errcode: http.StatusBadRequest,
		},
		"invalid annotation": {
			pool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type: appsv1beta2.Edge,
					Annotations: map[string]string{
						"-&#foo": "invalid annotation",
					},
				},
			},
			errcode: http.StatusUnprocessableEntity,
		},
		"invalid pool type": {
			pool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type: "invalid type",
				},
			},
			errcode: http.StatusUnprocessableEntity,
		},
		"invalid leader election strategy": {
			pool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type:                   appsv1beta2.Edge,
					LeaderElectionStrategy: "invalid strategy",
				},
			},
			errcode: http.StatusUnprocessableEntity,
		},
	}

	handler := &NodePoolHandler{}
	for k, tc := range testcases {
		t.Run(k, func(t *testing.T) {
			_, err := handler.ValidateCreate(context.TODO(), tc.pool)
			if tc.errcode == 0 {
				require.NoError(t, err, "Expected error code %d, got %v", tc.errcode, err)
				return
			}
			require.Error(t, err, "Expected error code %d, got %v", tc.errcode, err)

			statusErr := err.(*errors.StatusError)
			assert.Equal(t, tc.errcode, int(statusErr.Status().Code), "Expected error code %d, got %v", tc.errcode, err)
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	testcases := map[string]struct {
		oldPool runtime.Object
		newPool runtime.Object
		errcode int
	}{
		"update a normal nodepool": {
			oldPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type:                   appsv1beta2.Edge,
					LeaderElectionStrategy: string(appsv1beta2.ElectionStrategyRandom),
				},
			},
			newPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type: appsv1beta2.Edge,
					Labels: map[string]string{
						"foo": "bar",
					},
					LeaderElectionStrategy: string(appsv1beta2.ElectionStrategyMark),
				},
			},
			errcode: 0,
		},
		"oldPool is not a nodepool": {
			oldPool: &corev1.Node{},
			newPool: &appsv1beta2.NodePool{},
			errcode: http.StatusBadRequest,
		},
		"newPool is not a nodepool": {
			oldPool: &appsv1beta2.NodePool{},
			newPool: &corev1.Node{},
			errcode: http.StatusBadRequest,
		},
		"invalid pool type": {
			oldPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type: appsv1beta2.Edge,
				},
			},
			newPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type: "invalid type",
				},
			},
			errcode: http.StatusForbidden,
		},
		"type is changed": {
			oldPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type: appsv1beta2.Edge,
				},
			},
			newPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type: appsv1beta2.Cloud,
				},
			},
			errcode: http.StatusForbidden,
		},
		"host network is changed": {
			oldPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type:        appsv1beta2.Edge,
					HostNetwork: false,
				},
			},
			newPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type:        appsv1beta2.Edge,
					HostNetwork: true,
				},
			},
			errcode: http.StatusForbidden,
		},
		"interConnectivity is changed": {
			oldPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type:        appsv1beta2.Edge,
					HostNetwork: false,
				},
			},
			newPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type:        appsv1beta2.Edge,
					HostNetwork: true,
				},
			},
			errcode: http.StatusForbidden,
		},
		"leaderElectionStrategy is changed": {
			oldPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type:                   appsv1beta2.Edge,
					HostNetwork:            false,
					LeaderElectionStrategy: "mark",
				},
			},
			newPool: &appsv1beta2.NodePool{
				Spec: appsv1beta2.NodePoolSpec{
					Type:                   appsv1beta2.Edge,
					HostNetwork:            false,
					LeaderElectionStrategy: "random",
				},
			},
			errcode: 0,
		},
	}

	handler := &NodePoolHandler{}
	for k, tc := range testcases {
		t.Run(k, func(t *testing.T) {
			_, err := handler.ValidateUpdate(context.TODO(), tc.oldPool, tc.newPool)
			if tc.errcode == 0 {
				require.NoError(t, err, "Expected error code %d, got %v", tc.errcode, err)
				return
			}
			require.Error(t, err)
			statusErr := err.(*errors.StatusError)
			assert.Equal(t, tc.errcode, int(statusErr.Status().Code), "Expected error code %d, got %v", tc.errcode, err)
		})
	}
}

func prepareNodes() []client.Object {
	nodes := []client.Object{
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node1",
				Labels: map[string]string{
					projectinfo.GetNodePoolLabel(): "hangzhou",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
	}
	return nodes
}

func prepareNodePools() []client.Object {
	pools := []client.Object{
		&appsv1beta2.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				Name: "hangzhou",
			},
			Spec: appsv1beta2.NodePoolSpec{
				Type: appsv1beta2.Edge,
				Labels: map[string]string{
					"region": "hangzhou",
				},
			},
		},
		&appsv1beta2.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				Name: "beijing",
			},
			Spec: appsv1beta2.NodePoolSpec{
				Type: appsv1beta2.Edge,
				Labels: map[string]string{
					"region": "beijing",
				},
			},
		},
	}
	return pools
}

func TestValidateDelete(t *testing.T) {
	nodes := prepareNodes()
	pools := prepareNodePools()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal("Fail to add kubernetes clint-go custom resource")
	}
	apis.AddToScheme(scheme)

	c := fakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(pools...).WithObjects(nodes...).Build()

	testcases := map[string]struct {
		pool    runtime.Object
		errcode int
	}{
		"delete a empty nodepool": {
			pool: &appsv1beta2.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "beijing",
				},
			},
			errcode: 0,
		},
		"delete a nodepool with node in it": {
			pool: &appsv1beta2.NodePool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hangzhou",
				},
			},
			errcode: http.StatusForbidden,
		},
		"it is not a nodepool": {
			pool:    &corev1.Node{},
			errcode: http.StatusBadRequest,
		},
	}

	handler := &NodePoolHandler{
		Client: c,
	}
	for k, tc := range testcases {
		t.Run(k, func(t *testing.T) {
			_, err := handler.ValidateDelete(context.TODO(), tc.pool)
			if tc.errcode == 0 {
				require.NoError(t, err, "Expected error code %d, got %v", tc.errcode, err)
				return
			}
			require.Error(t, err)
			statusErr := err.(*errors.StatusError)
			assert.Equal(t, tc.errcode, int(statusErr.Status().Code), "Expected error code %d, got %v", tc.errcode, err)
		})
	}
}
