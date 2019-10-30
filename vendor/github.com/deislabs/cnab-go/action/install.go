package action

import (
	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/deislabs/cnab-go/driver"
)

// Install describes an installation action
type Install struct {
	Driver driver.Driver // Needs to be more than a string
}

// Run performs an installation and updates the Claim accordingly
func (i *Install) Run(c *claim.Claim, creds credentials.Set, opCfgs ...OperationConfigFunc) error {
	invocImage, err := selectInvocationImage(i.Driver, c)
	if err != nil {
		return err
	}

	op, err := opFromClaim(claim.ActionInstall, stateful, c, invocImage, creds)
	if err != nil {
		return err
	}

	err = OperationConfigs(opCfgs).ApplyConfig(op)
	if err != nil {
		return err
	}

	opResult, err := i.Driver.Run(op)

	// update outputs in claim even if there were errors so users can see the output files.
	outputErrors := setOutputsOnClaim(c, opResult.Outputs)

	if err != nil {
		c.Update(claim.ActionInstall, claim.StatusFailure)
		c.Result.Message = err.Error()
		return err
	}
	c.Update(claim.ActionInstall, claim.StatusSuccess)

	return outputErrors
}
