package pg

// RoleOptionList is meant to be a list of RoleOptions.
// For a unique list of options (either inverted or normal), you convert a RoleOptionsList to a map using ToMap()
// method.
type RoleOptionList []RoleOption

// ToMap converts a RoleOptionList to a RoleOptionMap
func (rol RoleOptionList) ToMap() RoleOptionMap {
	mapped := RoleOptionMap{}
	for _, opt := range rol {
		mapped[opt] = opt.Enabled()
	}
	return mapped
}

// ToValidMap converts a RoleOptionList to a map with RoleOption keys and boolean values, which is the fastest way of
// checking if it is defined
func (rol RoleOptionList) ToValidMap() map[RoleOption]bool {
	mapped := map[RoleOption]bool{}
	for _, opt := range rol {
		mapped[opt] = true
	}
	return mapped
}

// Inverted will invert all RoleOptions in the list and return a new list with inverted items
func (rol RoleOptionList) Inverted() RoleOptionList {
	var inverted RoleOptionList
	for _, option := range rol {
		inverted = append(inverted, option.Invert())
	}
	return inverted
}
