package cliopts

import (
	"github.com/spf13/pflag"
)

// ParametersOptions are shared CLI options about docker app parameters
type ParametersOptions struct {
	ParametersFiles []string
	Overrides       []string
}

// AddFlags adds the shared CLI flags to the given flag set
func (o *ParametersOptions) AddFlags(flags *pflag.FlagSet) {
	flags.StringArrayVar(&o.ParametersFiles, "parameters-file", []string{}, "Override parameters file")
	flags.StringArrayVarP(&o.Overrides, "set", "s", []string{}, "Override parameter value")
}
