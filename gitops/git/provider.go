package git

import "context"

// Pattern: Strategy -- swap git platform without
// changing PR creation logic.

// GitProvider creates pull requests on a git hosting
// platform.
type GitProvider interface {
	CreatePR(
		ctx context.Context,
		from string,
		to string,
		title string,
		body string,
	) error
}

// GitProviderFunc adapts a plain function to the
// GitProvider interface. When body is empty the title
// is used as body.
type GitProviderFunc func(
	ctx context.Context,
	from string,
	to string,
	title string,
	body string,
) error

// CreatePR delegates to the wrapped function. If body
// is empty, title is substituted.
func (f GitProviderFunc) CreatePR(
	ctx context.Context,
	from string,
	to string,
	title string,
	body string,
) error {
	if body == "" {
		body = title
	}

	return f(ctx, from, to, title, body)
}
