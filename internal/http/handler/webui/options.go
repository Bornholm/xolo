package webui

type Options struct {
}

type OptionFunc func(opts *Options)

func NewOptions(funcs ...OptionFunc) *Options {
	opts := &Options{}
	for _, fn := range funcs {
		fn(opts)
	}
	return opts
}
