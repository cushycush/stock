package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// stringList accepts either a YAML scalar or a YAML sequence of strings and
// normalizes both to a []string. This makes `hostname: work-laptop` and
// `hostname: [work-laptop, home-laptop]` both valid in when: filters.
type stringList []string

func (s *stringList) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var v string
		if err := node.Decode(&v); err != nil {
			return err
		}
		*s = stringList{v}
		return nil
	case yaml.SequenceNode:
		var v []string
		if err := node.Decode(&v); err != nil {
			return err
		}
		*s = stringList(v)
		return nil
	default:
		return fmt.Errorf("expected string or list of strings, got %v", node.Tag)
	}
}
