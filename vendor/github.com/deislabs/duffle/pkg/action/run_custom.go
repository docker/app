package action

import (
	"errors"
	"io"

	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/driver"
)

var (
	// ErrBlockedAction indicates that the requested action was not allowed.
	ErrBlockedAction = errors.New("action not allowed")
	// ErrUndefinedAction indicates that a bundle does not define this action.
	ErrUndefinedAction = errors.New("action not defined for bundle")
)

// RunCustom allows the execution of an arbitrary target in a CNAB bundle.
type RunCustom struct {
	Driver driver.Driver
	Action string
}

// blockedActions is a list of actions that cannot be run as custom.
//
// This prevents accidental circumvention of standard behavior.
var blockedActions = map[string]struct{}{"install": {}, "uninstall": {}, "upgrade": {}}

// Run executes a status action in an image
func (i *RunCustom) Run(c *claim.Claim, creds credentials.Set, w io.Writer) error {
	if _, ok := blockedActions[i.Action]; ok {
		return ErrBlockedAction
	}

	actionDef, ok := c.Bundle.Actions[i.Action]
	if !ok {
		return ErrUndefinedAction
	}

	invocImage, err := selectInvocationImage(i.Driver, c)
	if err != nil {
		return err
	}

	op, err := opFromClaim(i.Action, c, invocImage, creds, w)
	if err != nil {
		return err
	}

	err = i.Driver.Run(op)

	// If this action says it does not modify the release, then we don't track
	// it in the claim. Otherwise, we do.
	if !actionDef.Modifies {
		return err
	}

	status := claim.StatusSuccess
	if err != nil {
		c.Result.Message = err.Error()
		status = claim.StatusFailure
	}

	c.Update(i.Action, status)
	return err
}
