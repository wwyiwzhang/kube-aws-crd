# Kube-AWS-CRD

This project addresses the 'last mile' problem from creating an ELB service to manually creating/updating record set in AWS route53. Creating a CRD in kubernetes from scratch is still not very straight-forward but thanks to the *kubebuilder* project (https://github.com/kubernetes-sigs/kubebuilder), many functionalities such as scheme registration, RBAC, controller set up were created automatically.

## Design
There are two main components of this project, a controller and a CRD. The controller is responsible for watching the ELB services and checking if the ELB ingresses need to be updated in AWS route53. The CRD is where you can specify the services which need to be monitored by the controller and for each service, you will also need to provide other info such as the name of the hosted zone, TTL and CNAME.

## Installation
### Create custom resource definition (CRDs)
After `git clone` his repository, run 
```bash
cd kube-aws-crd; make install
```
to create the custom resource definition in the current k8s cluster. Please make sure the kubeconfig file is placed in the correct location.

### Create Controller
The controller is a k8s deployment object. You can use the following example as a reference. Since the controller needs to interact with AWS GO SDK library, you need to mount the AWS KEY and SECRET as a volume into the container.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: elbcontroller-manager
  namespace: default
  labels:
    app: elbcontroller
spec:
  selector:
    matchLabels:
      app: elbcontroller
  template:
    metadata:
        labels:
          app: elbcontroller
    spec:
      serviceAccount: elb
      containers:
      # Change the value of image field below to your controller image URL
      - image: wwyizhang/elbcontroller:latest
        name: elbcontroller
        volumeMounts:
        - mountPath: /root/.aws
          name: aws
      volumes:
      - name: aws
        secret:
          secretName: aws-key
```

### Create CRD instance

After the previous steps are complete, you can now create an ELB controller CRD to notify the controller which services need to be monitored. If any of the services whose ELB ingress has changed, the controller will initiate a request to update the record set in AWS route53. In case of a newly created ELB service, the controller will also create a new record set. Below is an example of ELB controller CRD, where you can specify multiple services for the controller to watch.

```yaml
apiVersion: lbcontrollers.loadbalancer.controller.io/v1alpha1
kind: LoadBalancerController
metadata:
  labels:
    controller-tools.k8s.io: "1.0"
  name: loadbalancercontroller-sample
spec:
  services: 
  - serviceName: nginx
  hostedZone: wwyiwzhang.net
  CNAME: some-app.wwyiwzhang.net
  TTL: 60
```

## Limitations
1. CRD will assume the hostedZones pre-exist before its creation
2. It currently only handles CNAME creation/updates
3. It currently only handles External Classic Load Balancer in AWS




