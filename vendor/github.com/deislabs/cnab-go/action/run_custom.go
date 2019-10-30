package action

import (
	"errors"

	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/deislabs/cnab-go/driver"
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
func (i *RunCustom) Run(c *claim.Claim, creds credentials.Set, opCfgs ...OperationConfigFunc) error {
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

	op, err := opFromClaim(i.Action, actionDef.Stateless, c, invocImage, creds)
	if err != nil {
		return err
	}

	err = OperationConfigs(opCfgs).ApplyConfig(op)
	if err != nil {
		return err
	}

	opResult, err := i.Driver.Run(op)

	// If this action says it does not modify the release, then we don't track
	// it in the claim. Otherwise, we do.
	if !actionDef.Modifies {
		return err
	}

	// update outputs in claim even if there were errors so users can see the output files.
	outputErrors := setOutputsOnClaim(c, opResult.Outputs)

	if err != nil {
		c.Update(i.Action, claim.StatusFailure)
		c.Result.Message = err.Error()
		return err
	}
	c.Update(i.Action, claim.StatusSuccess)

	return outputErrors
}
