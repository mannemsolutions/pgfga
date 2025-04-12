package ldap

import "regexp"

func validDN(dn string) bool {
	validDn := regexp.MustCompile(`^([a-zA-Z]+=[a-zA-Z0-9]+,)*[a-zA-Z]+=[a-zA-Z0-9]+$`)
	return validDn.MatchString(dn)
}

func validLDAPPair(pair string) (isValid bool) {
	validPair := regexp.MustCompile(`^[a-zA-Z]+=[a-zA-Z0-9]+$`)
	return validPair.MatchString(pair)
}
