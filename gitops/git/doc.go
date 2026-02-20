// Package git provides git repository operations and a strategy interface for
// creating pull requests across different git hosting platforms.
//
// The GitProvider interface abstracts PR creation. Implementations exist for
// GitHub, GitLab, and Bitbucket Server in sub-packages. GitProviderFunc is a
// convenience adapter that lets plain functions satisfy the interface.
//
// Repo wraps a local git clone with methods for branching, committing, and
// pushing. Clone creates a new Repo from a remote URL with optional mirror
// reference.
package git
