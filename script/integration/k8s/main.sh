#!/usr/bin/env bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

echo ""
echo "waiting up to 5 minutes for kubernetes to start..."

K8S_TIMEOUT="300"

SECONDS=0
while ! systemctl is-active --quiet multi-user.target; do
  sleep 2
  if [ "$SECONDS" -gt "$K8S_TIMEOUT" ]; then
    echo "ERROR: timed out waiting for kubernetes to start."
    exit 1
  fi
done

echo "kubernetes started in $SECONDS seconds."
echo ""

# load the locally-built flyteadmin image
docker load -i /images/flyteadmin

# start flyteadmin and dependencies
kubectl create -f "${DIR}/integration.yaml"

# in debug mode, run bash instead of running the tests
if [ -n "$DOCKERNETES_DEBUG" ]; then
  bash
fi

# wait for flyteadmin deployment to complete
kubectl -n flyte rollout status deployment flyteadmin

# get the name of the flyteadmin pod
POD_NAME=$(kubectl get pods -n flyte -o go-template="{{range .items}}{{.metadata.name}}:{{end}}" | tr ":" "\n" | grep flyteadmin)

# launch the integration tests
kubectl exec -it -n flyte "$POD_NAME" -- make -C /go/src/github.com/lyft/flyteadmin integration
