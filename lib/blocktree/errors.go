package blocktree

import (
	"errors"
)

var ErrParentNotFound = errors.New("cannot find parent block in blocktree")
var ErrBlockExists = errors.New("cannot add block to blocktree that already exists")
var ErrStartNodeNotFound = errors.New("start node does not exist")
var ErrEndNodeNotFound = errors.New("end node does not exist")
var ErrNilDatabase = errors.New("blocktree database is nil")
var ErrNilDescendant = errors.New("descendant node is nil")
var ErrDescendantNotFound = errors.New("could not find descendant node")
