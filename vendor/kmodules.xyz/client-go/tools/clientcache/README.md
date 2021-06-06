# clientcache

This package provides a `rest.Config` wrapper implementation that caches
Kubernetes api responses using a RFC 7234 compliant cache for http responses.
This connection should be only used in a single "session" to avoid repetitive
api calls to Kubernetes api server for same requests.

The cache interface is provided by the https://github.com/gregjones/httpcache
package and various implementations can be found on the project's website.
