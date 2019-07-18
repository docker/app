package claim

import (
	"fmt"
	"math/rand"
	"regexp"
	"time"

	"github.com/oklog/ulid"

	"github.com/deislabs/cnab-go/bundle"
)

// Status constants define the CNAB status fields on a Result.
const (
	StatusSuccess  = "success"
	StatusFailure  = "failure"
	StatusUnderway = "underway"
	StatusUnknown  = "unknown"
)

// Action constants define the CNAB action to be taken
const (
	ActionInstall   = "install"
	ActionUpgrade   = "upgrade"
	ActionDowngrade = "downgrade"
	ActionUninstall = "uninstall"
	ActionStatus    = "status"
	ActionUnknown   = "unknown"
)

// Claim is an installation claim receipt.
//
// Claims reprsent information about a particular installation, and
// provide the necessary data to upgrade, uninstall, and downgrade
// a CNAB package.
type Claim struct {
	Name          string                    `json:"name"`
	Revision      string                    `json:"revision"`
	Created       time.Time                 `json:"created"`
	Modified      time.Time                 `json:"modified"`
	Bundle        *bundle.Bundle            `json:"bundle"`
	Result        Result                    `json:"result"`
	Parameters    map[string]interface{}    `json:"parameters"`
	Outputs       map[string]interface{}    `json:"outputs"`
	Files         map[string]string         `json:"files"`
	RelocationMap bundle.ImageRelocationMap `json:"relocationMap"`
}

// ValidName is a regular expression that indicates whether a name is a valid claim name.
var ValidName = regexp.MustCompile("^[a-zA-Z0-9_-]+$")

// New creates a new Claim initialized for an installation operation.
func New(name string) (*Claim, error) {

	if !ValidName.MatchString(name) {
		return nil, fmt.Errorf("invalid name %q. Names must be [a-zA-Z0-9-_]+", name)
	}

	now := time.Now()
	return &Claim{
		Name:     name,
		Revision: ULID(),
		Created:  now,
		Modified: now,
		Result: Result{
			Action: ActionUnknown,
			Status: StatusUnknown,
		},
		Parameters:    map[string]interface{}{},
		Outputs:       map[string]interface{}{},
		RelocationMap: bundle.ImageRelocationMap{},
	}, nil
}

// Update is a convenience for modifying the necessary fields on a Claim.
//
// Per spec, when a claim is updated, the action, status, revision, and modified fields all change.
// All but status and action can be computed.
func (c *Claim) Update(action, status string) {
	c.Result.Action = action
	c.Result.Status = status
	c.Modified = time.Now()
	c.Revision = ULID()
}

// Result tracks the result of a Duffle operation on a CNAB installation
type Result struct {
	Message string `json:"message"`
	Action  string `json:"action"`
	Status  string `json:"status"`
}

// ULID generates a string representation of a ULID.
func ULID() string {
	now := time.Now()
	entropy := rand.New(rand.NewSource(now.UnixNano()))
	return ulid.MustNew(ulid.Timestamp(now), entropy).String()
}
