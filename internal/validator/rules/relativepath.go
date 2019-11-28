package rules

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type relativePathRule struct {
	volumes map[string]interface{}
	service string
}

func NewRelativePathRule() Rule {
	return &relativePathRule{
		volumes: map[string]interface{}{},
	}
}

func (s *relativePathRule) Collect(parent string, key string, value interface{}) {
	if parent == "volumes" {
		s.volumes[key] = value
	}
}

func (s *relativePathRule) Accept(parent string, key string) bool {
	if parent == "services" {
		s.service = key
	}
	return regexp.MustCompile("services.(.*).volumes").MatchString(parent + "." + key)
}

func (s *relativePathRule) Validate(value interface{}) []error {
	if m, ok := value.(map[string]interface{}); ok {
		src, ok := m["source"]
		if !ok {
			return []error{fmt.Errorf("invalid volume in service %q", s.service)}
		}
		_, volumeExists := s.volumes[src.(string)]
		if !filepath.IsAbs(src.(string)) && !volumeExists {
			return []error{fmt.Errorf("can't use relative path as volume source (%q) in service %q", src, s.service)}
		}
	}

	if m, ok := value.([]interface{}); ok {
		errs := []error{}
		for _, p := range m {
			str, ok := p.(string)
			if !ok {
				errs = append(errs, fmt.Errorf("invalid volume in service %q", s.service))
				continue
			}

			parts := strings.Split(str, ":")
			if len(parts) <= 1 {
				errs = append(errs, fmt.Errorf("invalid volume definition (%q) in service %q", str, s.service))
				continue
			}

			volumeName := parts[0]
			_, volumeExists := s.volumes[volumeName]
			if !filepath.IsAbs(volumeName) && !volumeExists {
				errs = append(errs, fmt.Errorf("can't use relative path as volume source (%q) in service %q", str, s.service))
				continue
			}
		}

		if len(errs) > 0 {
			return errs
		}
	}
	return nil
}
