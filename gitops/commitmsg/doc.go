// Package commitmsg generates and parses gitops target lists embedded in git
// commit messages. Targets are encoded between marker lines so that the prer
// package can detect which gitops deployments a commit carries.
package commitmsg
