// Copyright 2017 The quartermaster Authors
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

package glusterfs

import (
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/util/intstr"

	heketiclient "github.com/heketi/heketi/client/api/go-client"
)

func (st *GlusterStorage) heketiClient(namespace string) (*heketiclient.Client, error) {
	httpAddress, err := st.getHeketiAddress(namespace)
	if err != nil {
		return nil, err
	}

	// Create a new cluster
	// TODO(lpabon): Need to set user and secret
	return heketiclient.NewClientNoAuth(httpAddress), nil
}

func (st *GlusterStorage) getHeketiAddress(namespace string) (string, error) {
	service, err := st.client.Core().Services(namespace).Get("heketi")
	if err != nil {
		return "", logger.LogError("Failed to get cluster ip from Heketi service: %v", err)
	}
	if len(service.Spec.Ports) == 0 {
		return "", logger.LogError("Heketi service in namespace %s is missing port value", namespace)
	}
	address := fmt.Sprintf("http://%s:%d",
		service.Spec.ClusterIP,
		service.Spec.Ports[0].Port)
	logger.Debug("Got: %s", address)

	return address, nil
	// During development, running QM remotely, setup a portforward to heketi pod
	//return "http://localhost:8080", nil
}

func (st *GlusterStorage) deployHeketi(namespace string) error {
	// Create a service account for Heketi
	err := st.deployHeketiServiceAccount(namespace)
	if err != nil {
		return err
	}

	// Deployment
	err = st.deployHeketiPod(namespace)
	if err != nil {
		return err
	}

	// Create service to access Heketi API
	return st.deployHeketiService(namespace)
}

func (st *GlusterStorage) deployHeketiPod(namespace string) error {

	// Deployment for Heketi
	d := &extensions.Deployment{
		ObjectMeta: meta.ObjectMeta{
			Name:      "heketi",
			Namespace: namespace,
			Annotations: map[string]string{
				"description": "Defines how to deploy Heketi",
			},
			Labels: map[string]string{
				"glusterfs":     "heketi-deployment",
				"heketi":        "heketi-deployment",
				"quartermaster": "heketi",
			},
		},
		Spec: extensions.DeploymentSpec{
			Replicas: 1,
			Template: api.PodTemplateSpec{
				ObjectMeta: meta.ObjectMeta{
					Labels: map[string]string{"glusterfs": "heketi-deployment",
						"heketi":        "heketi-deployment",
						"quartermaster": "heketi",
						"name":          "heketi",
					},
					Name: "heketi",
				},
				Spec: api.PodSpec{
					ServiceAccountName: "heketi-service-account",
					Containers: []api.Container{
						api.Container{
							Name:            "heketi",
							Image:           "heketi/heketi:dev",
							ImagePullPolicy: api.PullIfNotPresent,
							Env: []api.EnvVar{
								api.EnvVar{
									Name:  "HEKETI_EXECUTOR",
									Value: "kubernetes",
								},
								api.EnvVar{
									Name:  "HEKETI_FSTAB",
									Value: "/var/lib/heketi/fstab",
								},
								api.EnvVar{
									Name:  "HEKETI_SNAPSHOT_LIMIT",
									Value: "14",
								},
							},
							Ports: []api.ContainerPort{
								api.ContainerPort{
									ContainerPort: 8080,
								},
							},
							VolumeMounts: []api.VolumeMount{
								api.VolumeMount{
									Name:      "db",
									MountPath: "/var/lib/heketi",
								},
							},
							ReadinessProbe: &api.Probe{
								TimeoutSeconds:      3,
								InitialDelaySeconds: 3,
								Handler: api.Handler{
									HTTPGet: &api.HTTPGetAction{
										Path: "/hello",
										Port: intstr.IntOrString{
											IntVal: 8080,
										},
									},
								},
							},
							LivenessProbe: &api.Probe{
								TimeoutSeconds:      3,
								InitialDelaySeconds: 30,
								Handler: api.Handler{
									HTTPGet: &api.HTTPGetAction{
										Path: "/hello",
										Port: intstr.IntOrString{
											IntVal: 8080,
										},
									},
								},
							},
						},
					},
					Volumes: []api.Volume{
						api.Volume{
							Name: "db",
						},
					},
				},
			},
		},
	}

	deployments := st.client.Extensions().Deployments(namespace)
	_, err := deployments.Create(d)
	if apierrors.IsAlreadyExists(err) {
		return nil
	} else if err != nil {
		logger.Err(err)
	}

	// Wait until deployment ready
	err = waitForDeploymentFn(st.client, namespace, d.GetName(), d.Spec.Replicas)
	if err != nil {
		return logger.Err(err)
	}

	logger.Debug("heketi deployed")
	return nil
}

func (st *GlusterStorage) deployHeketiServiceAccount(namespace string) error {
	s := &api.ServiceAccount{
		ObjectMeta: meta.ObjectMeta{
			Name:      "heketi-service-account",
			Namespace: namespace,
		},
	}

	// Submit the service account
	serviceaccounts := st.client.Core().ServiceAccounts(namespace)
	_, err := serviceaccounts.Create(s)
	if apierrors.IsAlreadyExists(err) {
		return nil
	} else if err != nil {
		return logger.Err(err)
	}

	logger.Debug("service account created")
	return nil
}

func (st *GlusterStorage) deployHeketiService(namespace string) error {
	s := &api.Service{
		ObjectMeta: meta.ObjectMeta{
			Name:      "heketi",
			Namespace: namespace,
			Labels: map[string]string{
				"glusterfs": "heketi-service",
				"heketi":    "support",
			},
			Annotations: map[string]string{
				"description": "Exposes Heketi Service",
			},
		},
		Spec: api.ServiceSpec{
			Selector: map[string]string{
				"name": "heketi",
			},
			Ports: []api.ServicePort{
				api.ServicePort{
					Name: "heketi",
					Port: 8080,
					TargetPort: intstr.IntOrString{
						IntVal: 8080,
					},
				},
			},
		},
	}

	// Submit the service
	services := st.client.Core().Services(namespace)
	_, err := services.Create(s)
	if apierrors.IsAlreadyExists(err) {
		return nil
	} else if err != nil {
		logger.Err(err)
	}

	logger.Debug("service account created")
	return nil
}
