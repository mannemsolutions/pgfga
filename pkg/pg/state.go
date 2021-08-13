package pg

import "fmt"

// State represents the state of a pg object (Present or Absent)
type State struct {
	value bool
}

var (
	// Present means the object should be created
	Present = State{true}
	// Absent means the object should be removed
	Absent  = State{false}

	toState = map[string]State{
		"Present":  Present,
		"Absent":  Absent,
		"": Present,
	}
)

func (s State) String() string {
	if s.value {
		return "Present"
	}
	return "Absent"
}

// MarshalYAML marshals the enum as a quoted json string
func (s State) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// UnmarshalYAML converts a yaml string to the enum value
func (s *State) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	if state, exists := toState[str]; exists {
		s.value = state.value
		return nil
	}
	return fmt.Errorf("invalid state %s (should be Present or Absent)", str)
}