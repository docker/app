package action

import (
	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/deislabs/cnab-go/driver"
)

// Upgrade runs an upgrade action
type Upgrade struct {
	Driver driver.Driver
}

// Run performs the upgrade steps and updates the Claim
func (u *Upgrade) Run(c *claim.Claim, creds credentials.Set, opCfgs ...OperationConfigFunc) error {
	invocImage, err := selectInvocationImage(u.Driver, c)
	if err != nil {
		return err
	}

	op, err := opFromClaim(claim.ActionUpgrade, stateful, c, invocImage, creds)
	if err != nil {
		return err
	}

	err = OperationConfigs(opCfgs).ApplyConfig(op)
	if err != nil {
		return err
	}

	opResult, err := u.Driver.Run(op)
	outputErrors := setOutputsOnClaim(c, opResult.Outputs)

	if err != nil {
		c.Update(claim.ActionUpgrade, claim.StatusFailure)
		c.Result.Message = err.Error()
		return err
	}
	c.Update(claim.ActionUpgrade, claim.StatusSuccess)

	return outputErrors
}
