# Test Environment

Enable buildx

```bash
docker buildx create --use
```

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/tn00869679/torcontroller/torcontroller-test-env:dev \
  -f dockerfile.testenv . --push
```
