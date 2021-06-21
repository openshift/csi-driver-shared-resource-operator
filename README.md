# shared-resources-operator

An operator to deploy the [Projected Shared Resource CSI Driver](https://github.com/openshift/gcp-pd-csi-driver) in OpenShift.

This operator will eventually be installed by the [cluster-storage-operator](https://github.com/openshift/cluster-storage-operator).

NOTE:  at the moment, using this driver is only supported via cloning this repository and executing the commands detailed below.

# Quick start

Before running the operator manually, you must remove the operator installed by CSO/CVO

```shell
# Scale down CVO and CSO
oc scale --replicas=0 deploy/cluster-version-operator -n openshift-cluster-version
oc scale --replicas=0 deploy/cluster-storage-operator -n openshift-cluster-storage-operator

# Delete operator resources from a clone of this repository
oc delete -F -f ./assets
```

To build and run the operator locally:

```shell
# Create only the resources the operator needs to run via CLI; this yaml will eventually reside in the cluster-storage-operator repository.
oc apply -f ./csi-driver-yaml-that-will-live-in-cluster-storage-operator.yaml 

# Build the operator
make

# Set the environment variables
export NODE_DRIVER_REGISTRAR_IMAGE=quay.io/openshift/origin-csi-node-driver-registrar:latest
export DRIVER_IMAGE=quay.io/openshift/origin-csi-driver-projected-resource:latest

# Run the operator via CLI
./shared-resources-operator start --kubeconfig $MY_KUBECONFIG --namespace openshift-cluster-csi-drivers
```

