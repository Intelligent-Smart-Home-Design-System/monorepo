package catalogbuilder

type Builder struct {
	// map of device type to slice of device field names that are used for identification
	identifyingAttributes map[string][]string
}

func NewBuilder(identifyingAttributes map[string][]string) *Builder {
	return &Builder{
		identifyingAttributes: identifyingAttributes,
	}
}
