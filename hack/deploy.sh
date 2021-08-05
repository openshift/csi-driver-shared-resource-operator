#! /bin/bash

set -e
set -o pipefail

rm -rf _deploy
mkdir -p _deploy
cp -r manifests/* _deploy/

operatorImage=${OPERATOR_IMAGE:-quay.io/openshift/origin-csi-driver-shared-resource-operator:latest}
driverImage=${DRIVER_IMAGE:-quay.io/openshift/origin-csi-driver-shared-resource:latest}
nodeRegistrar=${NODE_DRIVER_REGISTRAR_IMAGE:-quay.io/openshift/origin-csi-node-node-registrar:latest}
logLevel=${LOG_LEVEL:-5}

echo "Deploying operator image ${operatorImage}"
echo "Deploying driver image ${driverImage}"
echo "Deploying node registrar image ${nodeRegistrar}"
echo "Using log level ${logLevel}"

sed -i -e "s|\${OPERATOR_IMAGE}|${operatorImage}|g" \
  -e "s|\${DRIVER_IMAGE}|${driverImage}|g" \
  -e "s|\${NODE_DRIVER_REGISTRAR_IMAGE}|${nodeRegistrar}|g" \
  -e "s|\${LOG_LEVEL}|${logLevel}|g" \
  _deploy/09_deployment.yaml

oc apply -f _deploy/
