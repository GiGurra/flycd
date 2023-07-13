package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestTraverseDeepAppTree(t *testing.T) {
	path := "../../examples/projects"
	err := TraverseDeepAppTree(context.Background(), path, model.TraverseAppTreeOptions{
		ValidAppCb: func(node model.AppNode) error {
			fmt.Printf("Valid app: %s @ %s\n", node.AppConfig.App, node.Path)
			return nil
		},
	})

	if err != nil {
		t.Error(err)
	}
}

func TestTraverseDeepAppTree_cyclicDetection(t *testing.T) {
	path := "../../examples/cyclic"

	actual := make([]string, 0)

	err := TraverseDeepAppTree(context.Background(), path, model.TraverseAppTreeOptions{
		ValidAppCb: func(node model.AppNode) error {
			actual = append(actual, fmt.Sprintf("Valid app: %s", node.AppConfig.App))
			return nil
		},
		InvalidAppCb: func(node model.AppNode) error {
			actual = append(actual, fmt.Sprintf("Invalid app: %s", node.AppConfig.App))
			return nil
		},
		BeginProjectCb: func(node model.ProjectNode) error {
			actual = append(actual, fmt.Sprintf("Begin project: %s", node.ProjectConfig.Project))
			return nil
		},
		EndProjectCb: func(node model.ProjectNode) error {
			actual = append(actual, fmt.Sprintf("End project: %s", node.ProjectConfig.Project))
			return nil
		},
	})

	if err != nil {
		t.Error(err)
	}

	for _, step := range actual {
		fmt.Printf("step: %s\n", step)
	}

	desired := []string{
		"Begin project: cyclic",
		"Valid app: my-app-root",
		"Valid app: my-app",
		"End project: cyclic",
	}

	diff := cmp.Diff(actual, desired)
	if diff != "" {
		t.Fatalf("Steps are not equal: %v", diff)
	}
}
