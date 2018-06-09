package utils

import (
    "fmt"

    "github.com/Masterminds/semver"
)

// CheckVersionGte checks a version number against a given minimum and returns
// an error if it's lower
func CheckVersionGte(version, minVersion string) error {
    constraint, err := semver.NewConstraint(">= " + minVersion)
    if err != nil {
        return err
    }
    v, err := semver.NewVersion(version)
    if err != nil {
        return fmt.Errorf("invalid version number: %s: %s", version, err.Error())
    }
    if !constraint.Check(v) {
        return fmt.Errorf("version too low: %s < %s", version, minVersion)
    }
    return nil
}
