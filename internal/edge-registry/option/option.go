package option

type EdgeRegistryOption interface {
	apply(*EdgeRegistryOptions)
}

type EdgeRegistryOptions struct {
}

func NewDefaultOptions() *EdgeRegistryOptions {
	return &EdgeRegistryOptions{}
}

type funcServerOption struct {
	f func(*EdgeRegistryOptions)
}

func (fdo *funcServerOption) apply(eo *EdgeRegistryOptions) {
	fdo.f(eo)
}

func newFuncServerOption(f func(*EdgeRegistryOptions)) *funcServerOption {
	return &funcServerOption{
		f: f,
	}
}
