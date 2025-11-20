# Running the Bindplane Operator Locally with Minikube

This guide explains how to build and run the Bindplane Operator locally using Minikube, without pushing images to a remote registry.

## Prerequisites

- Minikube installed and configured
- Docker (or your preferred container runtime) installed
- `kubectl` configured to use your Minikube cluster
- `make` installed
- Go 1.24+ installed

## Step 1: Start Minikube

Ensure Minikube is running:

```bash
minikube start
```

Verify that `kubectl` is pointing to your Minikube cluster:

```bash
kubectl cluster-info
```

You should see output indicating you're connected to the Minikube cluster.

## Step 2: Configure Docker to Use Minikube's Docker Daemon

**IMPORTANT:** This step is required before building the image. You must configure your Docker environment to use Minikube's Docker daemon:

```bash
eval $(minikube docker-env)
```

**Note:** This command sets environment variables that configure Docker to use Minikube's Docker daemon. You'll need to run this in each new terminal session, or add it to your shell profile if you want it to persist.

Verify it's working:

```bash
docker ps
```

You should see Minikube's containers listed. If you see an error or don't see Minikube containers, make sure Minikube is running (`minikube status`).

## Step 3: Build the Operator Image

**CRITICAL:** Make sure you've completed Step 2 first! You must run `eval $(minikube docker-env)` in your current terminal session before building.

**Note:** The manager deployment is configured with `imagePullPolicy: Never` to use local images from Minikube's Docker daemon without trying to pull from a registry.

Build the operator Docker image using the Makefile:

```bash
make docker-build IMG=bindplane-operator:local
```

This will:
- Generate manifests and code
- Build the Go binary
- Create a Docker image tagged as `bindplane-operator:local`

**If you get an error like "failed to read dockerfile"**, it means you haven't configured Docker to use Minikube's daemon. Go back to Step 2 and run `eval $(minikube docker-env)` first.

Alternatively, you can use Docker directly (but still requires Step 2):

```bash
docker build -t bindplane-operator:local .
```

## Step 4: Verify the Image is Available

Since you're using Minikube's Docker daemon, the image should already be available to Minikube. Verify:

```bash
minikube image ls | grep bindplane-operator
```

You should see `bindplane-operator:local` listed.

## Step 5: Install CRDs

Install the Custom Resource Definitions into your Minikube cluster:

```bash
make install
```

This will apply the CRDs defined in `config/crd/bases/`.

Verify the CRDs are installed:

```bash
kubectl get crds | grep bindplane
```

You should see `bindplanes.k8s.bindplane.com` listed.

## Step 6: Deploy the Operator

Deploy the operator to your Minikube cluster:

```bash
make deploy
```

The Makefile is configured to use `bindplane-operator:local` by default, so you don't need to specify the image. This will:
- Deploy the operator with all necessary RBAC resources
- Create the operator namespace (`bindplane-operator-system` by default)
- Use the `bindplane-operator:local` image you built in Step 3

If you need to use a different image, you can override it:

```bash
make deploy IMG=your-image:tag
```

Alternatively, you can use kustomize directly (which will use the hardcoded image from `config/manager/kustomization.yaml`):

```bash
kustomize build config/default | kubectl apply -f -
```

## Step 7: Verify the Operator is Running

Check that the operator pod is running:

```bash
kubectl get pods -n bindplane-operator-system
```

You should see a pod named `bindplane-operator-controller-manager-*` in `Running` state.

View the operator logs:

```bash
kubectl logs -n bindplane-operator-system -l control-plane=controller-manager --tail=50
```

## Step 8: Create a Bindplane Instance

Create a sample Bindplane custom resource. First, check the sample:

```bash
cat config/samples/bindplane_v1alpha1_bindplane.yaml
```

You'll need to provide a `license` field and configure the `store` with PostgreSQL details. Create your own instance:

```bash
kubectl apply -f - <<EOF
apiVersion: k8s.bindplane.com/v1alpha1
kind: Bindplane
metadata:
  name: bindplane-sample
  namespace: default
spec:
  license: "your-license-key-here"
  store:
    type: postgres
    postgres:
      host: "your-postgres-host"
      port: "5432"
      database: "bindplane"
      username: "bindplane"
      password: "bindplane"
EOF
```

Replace the placeholder values with your actual configuration.

## Step 9: Verify Resources are Created

The operator should create the supporting resources. Check for Transform Agent resources:

```bash
kubectl get deployment -l app.kubernetes.io/component=transform-agent
kubectl get service -l app.kubernetes.io/component=transform-agent
kubectl get serviceaccount -l app.kubernetes.io/component=transform-agent
```

Check for Prometheus resources:

```bash
kubectl get statefulset -l app.kubernetes.io/component=prometheus
kubectl get service -l app.kubernetes.io/component=prometheus
kubectl get serviceaccount -l app.kubernetes.io/component=prometheus
```

## Step 10: Monitor the Operator

Watch the operator logs in real-time:

```bash
kubectl logs -n bindplane-operator-system -l control-plane=controller-manager -f
```

## Troubleshooting

### Image Not Found or Build Errors

If you see image pull errors or build errors like "failed to read dockerfile", ensure you're using Minikube's Docker daemon:

```bash
# First, configure Docker to use Minikube
eval $(minikube docker-env)

# Verify you're using Minikube's Docker
docker ps
# You should see Minikube containers listed

# Then rebuild the image
make docker-build IMG=bindplane-operator:local
```

**Common issue:** If you get "failed to read dockerfile" errors, it usually means you haven't run `eval $(minikube docker-env)` in your current terminal session. Make sure to run this command before building.

### Operator Not Starting

Check the operator pod status:

```bash
kubectl describe pod -n bindplane-operator-system -l control-plane=controller-manager
```

Check for RBAC issues:

```bash
kubectl logs -n bindplane-operator-system -l control-plane=controller-manager | grep -i error
```

### Resources Not Being Created

Check the Bindplane resource status:

```bash
kubectl get bindplane -o yaml
```

Check operator logs for reconciliation errors:

```bash
kubectl logs -n bindplane-operator-system -l control-plane=controller-manager | grep -i reconcile
```

## Cleanup

To remove the operator and all created resources:

```bash
make undeploy
make uninstall
```

Or manually:

```bash
kubectl delete bindplane --all
kubectl delete -f config/default/
kubectl delete -f config/crd/bases/
```

To stop Minikube:

```bash
minikube stop
```

To delete the Minikube cluster:

```bash
minikube delete
```

## Rebuilding After Code Changes

When you make code changes, rebuild and redeploy:

```bash
# Rebuild the image
make docker-build IMG=bindplane-operator:local

# Restart the operator deployment
kubectl rollout restart deployment/bindplane-operator-controller-manager -n bindplane-operator-system
```

Or delete the pod to force a restart:

```bash
kubectl delete pod -n bindplane-operator-system -l control-plane=controller-manager
```

The deployment will automatically create a new pod with the updated image.

## Notes

- The `eval $(minikube docker-env)` command only affects the current terminal session. Run it again if you open a new terminal.
- If you switch between Minikube and other Kubernetes contexts, remember to run `eval $(minikube docker-env)` again.
- The operator namespace is `bindplane-operator-system` by default, but can be changed in `config/default/kustomization.yaml`.

