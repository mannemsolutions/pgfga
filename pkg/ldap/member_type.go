package ldap

type memberType int

const (
	groupMType memberType = iota
	userMType
	unknownMType
)

func getmemberType(key string) (mt memberType) {
	switch key {
	case "cn":
		return groupMType
	case "uid":
		return userMType
	default:
		return unknownMType
	}
}
