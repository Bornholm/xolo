package token

type Options struct {
	SessionName string
}

type OptionFunc func(opts *Options)

func NewOptions(funcs ...OptionFunc) *Options {
	opts := &Options{
		SessionName: "xolo_auth_token",
	}

	for _, fn := range funcs {
		fn(opts)
	}

	return opts
}

func WithSessionName(sessionName string) OptionFunc {
	return func(opts *Options) {
		opts.SessionName = sessionName
	}
}
