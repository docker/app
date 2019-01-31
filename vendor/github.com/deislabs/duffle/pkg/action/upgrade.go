package action

import (
	"io"

	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/driver"
)

// Upgrade runs an upgrade action
type Upgrade struct {
	Driver driver.Driver
}

// Run performs the upgrade steps and updates the Claim
func (u *Upgrade) Run(c *claim.Claim, creds credentials.Set, w io.Writer) error {
	invocImage, err := selectInvocationImage(u.Driver, c)
	if err != nil {
		return err
	}

	op, err := opFromClaim(claim.ActionUpgrade, c, invocImage, creds, w)
	if err != nil {
		return err
	}
	if err := u.Driver.Run(op); err != nil {
		c.Update(claim.ActionUpgrade, claim.StatusFailure)
		c.Result.Message = err.Error()
		return err
	}

	c.Update(claim.ActionUpgrade, claim.StatusSuccess)
	return nil
}
