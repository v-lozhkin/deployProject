package usecase

import (
	"context"
	"fmt"
	"github.com/v-lozhkin/deployProject/internal/pkg/models"
)

func (u usecase) Update(ctx context.Context, item models.Item) error {
	defer u.stat.MethodDuration.WithLabels(map[string]string{"method_name": "Update"}).Start().Stop()

	if err := item.Validate(); err != nil {
		return fmt.Errorf("item's validate failed: %w", err)
	}

	if err := u.repo.Update(ctx, item); err != nil {
		return fmt.Errorf("failed to update item in repo: %w", err)
	}

	return nil
}
