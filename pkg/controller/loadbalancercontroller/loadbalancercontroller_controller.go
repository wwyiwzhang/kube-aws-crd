/*
Copyright 2019 Weiwei Zhang.

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

package loadbalancercontroller

import (
	"context"
	"reflect"
	"time"

	lbcontrollersv1alpha1 "kube-aws-crd/pkg/apis/lbcontrollers/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/aws/aws-sdk-go/aws/session"
	route53 "github.com/aws/aws-sdk-go/service/route53"
	deepcopy "github.com/getlantern/deepcopy"
)

var log = logf.Log.WithName("controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new LoadBalancerController Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileLoadBalancerController{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("loadbalancercontroller-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to LoadBalancerController
	err = c.Watch(&source.Kind{Type: &lbcontrollersv1alpha1.LoadBalancerController{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create
	// Uncomment watch a Deployment created by LoadBalancerController - change this for objects you create
	err = c.Watch(&source.Kind{Type: &lbcontrollersv1alpha1.LoadBalancerController{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &lbcontrollersv1alpha1.LoadBalancerController{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileLoadBalancerController{}

// ReconcileLoadBalancerController reconciles a LoadBalancerController object
type ReconcileLoadBalancerController struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a LoadBalancerController object and makes changes based on the state read
// and what is in the LoadBalancerController.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=lbcontrollers.loadbalancer.controller.io,resources=loadbalancercontrollers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=lbcontrollers.loadbalancer.controller.io,resources=loadbalancercontrollers/status,verbs=get;update;patch
func (r *ReconcileLoadBalancerController) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the LoadBalancerController instance
	instance := &lbcontrollersv1alpha1.LoadBalancerController{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	sess, err := session.NewSession()
	if err != nil {
		log.Error("Failed to create new session," err)
		return
	}

	instanceCopy := interface{}
	deepcopy.Copy(instance, instanceCopy)
	svcStatus := instanceCopy.Spec.Status.LoadBalancerServiceStatus

	for item := range instance.Spec.Services {
		ctx := context.Background()
		serviceIngress := r.GetServiceIngress(ctx, item.ServiceName, request.Namespace)
		targetIngress := GetResourceRecordValue(sess, item.CNAME, item.HostedZone)
		if &targetIngress != nil && targetIngress == serviceIngress {
			log.Info("Detected no change for service: %s in namespace: %s", item.Service, request.Namespace)
			return reconcile.Result{}, nil
		} else { 
			if &targetIngress != nil && targetIngress != serviceIngress {
				log.Info("Deteced change for service: %s in namespace: %s", item.Service, request.Namespace)
			} else {
				log.Info("Detected new service ingress value for service: %s in namespace: %s", item.Service, request.Namespace)
			}
			err = UpsertResourceRecord(sess, item.CNAME, item.HostedZone, serviceIngress, item.TTL)
			if err != nil {
				log.Error("Failed to update load balancer ingress in route53 for service: %s in namespace: %s", item.Service, request.Namespace)
				return reconcile.Result{}, err
			}
			// Update ServiceStatus
			currentTime := metav1.Time{time.Now()}
			for idx, status := range instanceCopy.Spec.Status.LoadBalancerServiceStatus {
				if status.ServiceName == item.ServiceName {
					instanceCopy.Spec.Status.LoadBalancerServiceStatus[idx].LastUpdate = currentTime
					instanceCopy.Spec.Status.LoadBalancerServiceStatus[idx].Count += 1
				}
			}
			if !reflect.DeepEqual(instance, instanceCopy) {
				err = r.Update(context.TODO(), found)
				if err != nil {
					return reconcile.Result{}, err
				}
			}
			reconcile.Result{}, nil
		}
	}

	// Compare the service load balancer ingresses with AWS route53 entries

	// // TODO(user): Change this to be the object type created by your controller
	// deploy := &lbcontrollersv1alpha1.LoadBalancerController{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      instance.Name,
	// 		Namespace: instance.Namespace,
	// 	},
	// 	Spec: lbcontrollersv1alpha1.LoadBalancerControllerSpec{
	// 		Selector: &metav1.LabelSelector{
	// 			MatchLabels: map[string]string{"load-balancer-controller": instance.Name},
	// 		},
	// 		Services: []&lbcontrollersv1alpha1.LoadBalancerService{
	// 			{

	// 			},
	// 			{

	// 			},
	// 		},
	// 	},
	// }
	// if err := controllerutil.SetControllerReference(instance, deploy, r.scheme); err != nil {
	// 	return reconcile.Result{}, err
	// }

	// // TODO(user): Change this for the object type created by your controller
	// // Check if the Deployment already exists
	// found := &appsv1.Deployment{}
	// err = r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
	// if err != nil && errors.IsNotFound(err) {
	// 	log.Info("Creating Deployment", "namespace", deploy.Namespace, "name", deploy.Name)
	// 	err = r.Create(context.TODO(), deploy)
	// 	return reconcile.Result{}, err
	// } else if err != nil {
	// 	return reconcile.Result{}, err
	// }

	// // TODO(user): Change this for the object type created by your controller
	// // Update the found object and write the result back if there are any changes
	// if !reflect.DeepEqual(deploy.Spec, found.Spec) {
	// 	found.Spec = deploy.Spec
	// 	log.Info("Updating Deployment", "namespace", deploy.Namespace, "name", deploy.Name)
	// 	err = r.Update(context.TODO(), found)
	// 	if err != nil {
	// 		return reconcile.Result{}, err
	// 	}
	// }
	// return reconcile.Result{}, nil
}

func (r *ReconcileLoadBalancerController) GetServiceIngress(serviceName string, namespace string) string {
	svc := &corev1.Service{}
	err = r.Get(context.TODO(), types.NamespacedName{Name: serviceName, Namespace: namespace}, svc)
	if err == nil {
		return svc.Status.LoadBalancer.Ingress[0].Hostname
	} else if err != nil && errors.IsNotFound(err) {
		log.Info("Could not find load balancer service: %s in namespace: %s", serviceName, namespace)
	}
	return nil
}

// GetResourceRecordValue func assumes the hostedZone has been created
// it will get the target value by searching inside hostedZone for CNAME
func GetResourceRecordValue(session *route53.Route53, name string, hostedZone string) string {
	listParams := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(hostedZone),
		StartRecordName: aws.String(name),
		StartRecordType: aws.String("CNAME"),
	}
	respList, err := session.ListResourceRecordSets(listParams)

	if err != nil {
		log.Error(err.Error())
		return
	} else if len(respList) == 0 {
		log.Info("Could not find target value for CNAME: %s in hosted zone: %s", name, hostedZone)
		return
	} else {
		return respList.ResourceRecordSets[0].ResourceRecords[0].Value
	}
}

func UpsertResourceRecord(session *route53.Route53, name string, hostedZone string, loadBalancerIngress string, TTL int32) bool {
	rrsinput := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{ // Required
			Changes: []*route53.Change{ // Required
				{ // Required
					Action: aws.String("UPSERT"), // Required
					ResourceRecordSet: &route53.ResourceRecordSet{ // Required
						Name: aws.String(name),    // Required
						Type: aws.String("CNAME"), // Required
						ResourceRecords: []*route53.ResourceRecord{
							{ // Required
								Value: aws.String(loadBalancerIngress), // Required
							},
						},
						TTL: aws.Int64(TTL),
					},
				},
			},
			Comment: aws.String("Sample update."),
		},
		HostedZoneId: aws.String(hostedZone), // Required
	}
	resp, err := svc.ChangeResourceRecordSets(rrsinput)

	if err != nil {
		log.Error(err.Error())
		return err
	}
	return
}
