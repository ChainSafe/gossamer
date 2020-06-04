package utils

import "testing"

type Framework struct {
	nodes []*Node
}

func InitFramework(qtyNodes int) (*Framework, error) {
	f := &Framework{	}
	nodes, err := InitNodes(qtyNodes)
	if err != nil {
		return nil, err
	}
	f.nodes = nodes
	return f, nil
}

func (fw *Framework) StartNodes(t *testing.T) (errorList []error) {
	for _, node := range fw.nodes {
		err := RestartGossamer(t, node)
		if err != nil {
			errorList = append(errorList, err)
		}
	}
	return errorList
}

func (fw *Framework)KillNodes(t *testing.T) []error {
	return TearDown(t, fw.nodes)
}
