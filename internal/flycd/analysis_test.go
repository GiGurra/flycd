package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"testing"
)

func TestTraverseDeepAppTree(t *testing.T) {
	path := "../../examples/projects"
	err := TraverseDeepAppTree(context.Background(), path, TraverseAppTreeOptions{
		ValidAppCb: func(node model.AppNode) error {
			fmt.Printf("Valid app: %s @ %s\n", node.AppConfig.App, node.Path)
			return nil
		},
	})

	if err != nil {
		t.Error(err)
	}
}
