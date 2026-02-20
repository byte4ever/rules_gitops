// Package filter transforms Kubernetes manifests for integration testing by
// replacing persistent storage references with ephemeral volumes and adjusting
// certificate issuers. It drops PersistentVolumeClaim and Ingress objects,
// converts StatefulSet volumeClaimTemplates to emptyDir volumes, and replaces
// letsencrypt-prod issuers with self-signed ones.
package filter
