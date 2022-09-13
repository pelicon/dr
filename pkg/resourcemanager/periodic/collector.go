package periodic

import (
	"context"
)

type collector struct {
}

func New() *collector {
	return &collector{}
}

func (c *collector) Start(ctx context.Context) {}
