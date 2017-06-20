# Build Instructions

## Requirements
- go1.5+
- glide

## Build Binary
```sh
# Install/Update dependency (needs glide)
glide slow

# Build
./hack/make.py
```

## Build Docker
```sh
# Build Docker image
# This will build Backup Controller Binary and use it in docker
./hack/docker/setup.sh
```

###### Push Docker Image
```sh
# This will push docker image to other repositories

# Add docker tag for your repository
docker tag appscode/stash:<tag> <image>:<tag>

# Push Image
docker push <image>:<tag>

# Example:
docker tag appscode/stash:default sauman/stash:default
docker push sauman/stash:default
```
