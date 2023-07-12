package flycd

import (
	"fmt"
	"testing"
)

func TestTraverseDeepAppTree(t *testing.T) {
	path := "../../examples"
	err := TraverseDeepAppTree(path, TraverseAppTreeOptions{
		ValidAppCb: func(node AppNode) error {
			fmt.Printf("Valid app: %s @ %s\n", node.AppConfig.App, node.Path)
			return nil
		},
	})

	if err != nil {
		t.Error(err)
	}
}
