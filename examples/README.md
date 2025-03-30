# Helm Values Manager Examples

This directory contains example values files and usage demonstrations for the Helm Values Manager tool.

## Example Files

- `test-upstream.yaml` - Example chart defaults (upstream values)
- `test-downstream.yaml` - Example user configuration with both redundant and unsupported values

## Tutorial: Optimizing Nginx Ingress Controller Values

This tutorial demonstrates how to use Helm Values Manager to optimize your values for the popular nginx-ingress controller.

### 1. Get the default values from the chart

```bash
# Add the ingress-nginx repository
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

# Download the default values to a file
helm show values ingress-nginx/ingress-nginx > nginx-upstream-values.yaml
```

### 2. Create your custom values file

Create a file called `my-nginx-values.yaml` with your customizations:

```yaml
controller:
  replicaCount: 2
  service:
    type: LoadBalancer
    externalTrafficPolicy: Local
  resources:
    limits:
      cpu: 100m
      memory: 90Mi
    requests:
      cpu: 100m
      memory: 90Mi
  metrics:
    enabled: true

# The following value doesn't exist in the upstream chart
invalidSetting: "this will be detected as unsupported"
```

### 3. Analyze with Helm Values Manager

```bash
helm value-manager -upstream nginx-upstream-values.yaml -downstream my-nginx-values.yaml -outdir nginx-analysis
```

### 4. Generate optimized values

```bash
helm value-manager -upstream nginx-upstream-values.yaml -downstream my-nginx-values.yaml -outdir nginx-analysis -optimize
```

### 5. Reviewing the output

```bash
# Check unsupported values
cat nginx-analysis/unsupported-values.yaml

# Check redundant values (if any exist in your custom file)
cat nginx-analysis/redundant-values.yaml

# Review optimized values
cat nginx-analysis/optimized-values.yaml
```

## Direct Helm Release Example

If you already have the ingress-nginx controller installed, you can analyze the active release:

```bash
# Analyze an existing release
helm value-manager -repo ingress-nginx -outdir nginx-analysis

# Optimize an existing release
helm value-manager -repo ingress-nginx -outdir nginx-analysis -optimize
```

This will:
1. Fetch the values from the current release
2. Compare with the chart defaults
3. Generate analysis files in the nginx-analysis directory

## Comparing Between Releases

You can also compare between different revisions of a chart:

```bash
# Compare with a specific revision
helm value-manager -repo ingress-nginx -revision 2 -outdir nginx-analysis
```

## Using in CI/CD

Here's an example of how you might use Helm Values Manager in a CI/CD pipeline:

```bash
#!/bin/bash
# Example CI script

# Run the values manager
helm value-manager -upstream nginx-upstream-values.yaml -downstream my-nginx-values.yaml -outdir ci-results

# Check if unsupported values exist
if [ -s "ci-results/unsupported-values.yaml" ]; then
  echo "Error: Unsupported values detected in configuration!"
  cat ci-results/unsupported-values.yaml
  exit 1
fi

# Deploy using optimized values if required
if [ "$OPTIMIZE" == "true" ]; then
  helm value-manager -upstream nginx-upstream-values.yaml -downstream my-nginx-values.yaml -outdir ci-results -optimize
  helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx -f ci-results/optimized-values.yaml
else
  helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx -f my-nginx-values.yaml
fi
```
