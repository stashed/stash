#!/bin/bash

RESTIC_VERSION=0.8.3
STASH_VERSION=0.7.0
ARCH=arm64
REPOSITORY=appscode

GOPATH=$(go env GOPATH)
REPO_ROOT=$GOPATH/src/github.com/appscode/stash

# Build stash
go get github.com/appscode/stash
pushd ${REPO_ROOT}
git checkout ${STASH_VERSION}
export GOARCH=arm64
go build ./...
popd

mkdir -p /build_dir
cd /build_dir
cp $REPO_ROOT/dist/stash/stash-alpine-amd64 ./stash
chmod 755 ./stash

# Download restic
wget https://github.com/restic/restic/releases/download/v${RESTIC_VERSION}/restic_${RESTIC_VERSION}_linux_${ARCH}.bz2
bzip2 -d restic_${RESTIC_VERSION}_linux_${ARCH}.bz2
mv restic_${RESTIC_VERSION}_linux_${ARCH}.bz2 restic

# Build Docker container (done on ARM64 machine)
cat >> Dockerfile <<EOF
FROM alpine

RUN set -x \
  && apk add --update --no-cache ca-certificates

COPY restic /bin/restic
COPY stash /bin/stash

ENTRYPOINT ["/bin/stash"]
EXPOSE 56789 56790
EOF

# Build and push image
docker build -t ${REPOSITORY}/stash:${STASH_VERSION}-${ARCH} .
docker push ${REPOSITORY}/stash:${STASH_VERSION}-${ARCH}

# Create manifest (adjust on platform being used)
#wget https://github.com/estesp/manifest-tool/releases/download/v0.7.0/manifest-tool-linux-amd64
#mv manifest-tool-linux-amd64 manifest-tool
#chmod +x manifest-tool

# Generates the version manifest pointing to the arch images
#manifest-tool push from-args --platforms linux/amd64,linux/arm64 --template "${REPOSITORY}/stash:${STASH_VERSION}-ARCH" --target "${REPOSITORY}/stash:${STASH_VERSION}"

# Generates the :latest manifest pointing to the built arch images
#manifest-tool push from-args --platforms linux/amd64,linux/arm64 --template "${REPOSITORY}/stash:${STASH_VERSION}-ARCH" --target "${REPOSITORY}/stash:latest"
