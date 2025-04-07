package externalities

import "github.com/tidwall/btree"

type Extensions struct {
	extensions btree.Map[string, any] //TODO check this
}

func NewExtensions() Extensions {
	return Extensions{
		extensions: btree.Map[string, any]{},
	}
}

type Externalities interface {
	// TODO: add methods
}
