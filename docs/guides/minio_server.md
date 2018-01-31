# Deploying Minio Server in Kubernetes Cluster with Self-signed Certificate

Minio is an open source object storage server compatible with Amazon S3 cloud storage service. You can deploy Minio server in docker container locally, in a kubernetes cluster, Microsoft Azure, GCP etc. You can find a guide for Minio server [here](https://docs.minio.io/).

This tutorial will show you how to deploy Minio Server in Kubernetes cluster with a self-signed certificate.

## Prerequisites

To begin with this tutorial we will need some tools and concepts. This section will focus on these prerequisite materials.

### Kubernetes Cluster

At first, you need to have a Kubernetes cluster, and the kubectl command-line tool must be configured to communicate with your cluster. If you do not already have a cluster, you can create one by using Minikube. You can create a cluster in minikube by following [this guide](https://kubernetes.io/docs/getting-started-guides/minikube/).

You must be familer with these kubernetes resources:

- [Secret](https://kubernetes.io/docs/concepts/configuration/secret/)
- [Persistent Volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
- [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
- [Service](https://kubernetes.io/docs/concepts/services-networking/service/)

### Self-signed Certificate

A Certificate is used to verify the identity of server or client. Usually, a certificate issued by trusted third party is used to verify identity. We can also use a self-signed certificate. In this tutorial, we will use a self-signed certificate to verify the identity of Minio server.

You can generate self-signed certificate easily with our [onessl](https://github.com/appscode/onessl) tool.

Here is an example how we can generate a self-signed certificate using `onessl` tool.

First install onessl by,

```console
$ curl -fsSL -o onessl https://github.com/appscode/onessl/releases/download/0.1.0/onessl-linux-amd64 \
  && chmod +x onessl \
  && sudo mv onessl /usr/local/bin/
```

Now generate CA's root certificate,

```console
$ onessl create ca-cert
```

This will create two files `ca.crt` and `ca.key`.

Now, generate  certificate for server,

```console
$ onessl create server-cert --domains minio-service.default.svc
```

This will generate two files `server.crt` and `server.key`.

Minio server will start TLS secure service if it find `public.crt` and `private.key` files in `/root/.minio/certs/` directory of the docker container. The `public.crt` file is concatenation of `server.crt` and `ca.crt` where `private.key` file is only the `server.key` file.

Let's generate `public.crt` and `private.key` file,

```console
$ cat {server.crt,ca.crt} > public.crt
$ cat server.key > private.key
```

Be sure about the order of `server.crt`  and `ca.crt`. The order will be `server's certificate`, any `intermediate certificates` and finally the `CA's root certificate`. The intermediate certificates are required if the server certificate is created using a certificate which is not the root certificate but signed by the root certificate. [onessl](https://github.com/appscode/onessl) use root certificate by default to generate server certificate if no certificate path is specified by `--cert-dir` flag. Hence, the intermediate certificates are not required here.

We will create a kubernetes secret with this `public.crt` and `private.key` files and mount the secret to `/root/.minio/certs/` directory of minio container.

## Create Secret

Now, let's create a secret from `public.crt` and `private.key` files,

```console
$ kubectl create secret generic minio-server-secret \
                              --from-file=./public.crt \
                              --from-file=./private.key
secret "minio-server-secret" created

$ kubectl label secret minio-server-secret app=minio -n default
```

Now, verify that the secret is created successfully

```console
$ kubectl get secret minio-server-secret -o yaml
```

If secret is created successfully then you will see output like this,

```yaml
apiVersion: v1
data:
  private.key: <base64 encoded private.key data>
  public.crt: <base64 encoded public.key data>
kind: Secret
metadata:
  creationTimestamp: 2018-01-26T12:02:09Z
  name: minio-server-secret
  namespace: default
  resourceVersion: "40701"
  selfLink: /api/v1/namespaces/default/secrets/minio-server-secret
  uid: bc57add7-0290-11e8-9a26-080027b344c9
  labels:
    app: minio
type: Opaque
```

## Create Persistent Volume Claim

Minio server needs a Persistent Volume to store data. Let's create a `Persistent Volume Claim` to request Persistent Volume from the cluster.

```console
$ kubectl apply -f ./persistentVoluemClaim.yaml
persistentvolumeclaim "minio-pv-claim" created
```

YAML for PersistentVolumeClaim,

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  # This name uniquely identifies the PVC. Will be used in minio deployment.
  name: minio-pv-claim
  labels:
    app: minio
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    # This is the request for storage. Should be available in the cluster.
    requests:
      storage: 2Gi
```

## Create Deployment

Minio deployment creates pod where the Minio server will run. Let's create a deployment for minio server by,

```console
$ kubectl apply -f ./minio-deployment.yaml
deployment "minio-deployment" created
```

YAML for minio-deployment

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  # This name uniquely identifies the Deployment
  name: minio-deployment
  labels:
    app: minio
spec:
  strategy:
    type: Recreate # If pod fail, we want to recreate pod rather than restarting it.
  template:
    metadata:
      labels:
        # Label is used as a selector in the service.
        app: minio-server
    spec:
      volumes:
      # Refer to the PVC have created earlier
      - name: storage
        persistentVolumeClaim:
          # Name of the PVC created earlier
          claimName: minio-pv-claim
      # Refer to minio-server-secret we have created earlier
      - name: minio-server-secret
        secret:
          secretName: minio-server-secret
      containers:
      - name: minio
        # Pulls the default Minio image from Docker Hub
        image: minio/minio
        args:
        - server
        - --address
        - ":443"
        - /storage
        env:
        # Minio access key and secret key
        - name: MINIO_ACCESS_KEY
          value: "<your minio access key(any string)>"
        - name: MINIO_SECRET_KEY
          value: "<your minio secret key(any string)>"
        ports:
        - containerPort: 443
          # This ensures containers are allocated on separate hosts. Remove hostPort to allow multiple Minio containers on one host
          hostPort: 443
        # Mount the volumes into the pod
        volumeMounts:
        - name: storage # must match the volume name, above
          mountPath: "/storage"
        - name: minio-server-secret
          mountPath: "/root/.minio/certs/" # directory where the certificates will be mounted
```

## Create Service

Now, the final touch. Minio server is running on the cluster. Let's create a service so that other pods can access the server.

```console
$ kubectl apply -f ./serviceForMinioServer.yaml
service "minio-service" created
```

YAML for minio-service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: minio-service
  labels:
    app: minio
spec:
  type: LoadBalancer
  ports:
    - port: 443
      targetPort: 443
      protocol: TCP
  selector:
    app: minio-server # must match with the label used in the deployment
```

Now we need NodePort of the service. Let's find out the NodePort,

```console
$ kubectl get service minio-service
NAME            TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)         AGE
minio-service   LoadBalancer   10.106.121.137   <pending>     443:30722/TCP   49s
```

Look at the `PORT(S)` column. Here `30722` of `443:30722/TCP` is the NodePort. Now, we can access the Minio Server using the following address: `https://<cluster-ip>:30772`. For minikube, the address is `https://192.168.99.100:30722`.

## Cleanup

To cleanup the Kubernetes resources created by this tutorial, run:

```console
$ kubectl delete deployment minio-deployment
$ kubectl delete service minio-service
$ kubectl delete secret minio-server-secret
```
