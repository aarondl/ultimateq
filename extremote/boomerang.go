package extremote

import (
	"github.com/aarondl/ultimateq/api"
	"golang.org/x/net/context"
)

type boomerang struct {
}

func (b boomerang) Connect(ctx context.Context, details *api.ConnectionDetails) (*api.Empty, error) {
}
