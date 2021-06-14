# dynamic-factory

Kubernetes client-go and lister packages use similar but different function
signature for `LIST` and `GET` api calls. This makes it difficult to switch
from one implemenation to the other. This package implements a dynamic factory
interface that can be used to either direct read api objects from Kubernetes
api server or read from locally cached indexer/lister.
