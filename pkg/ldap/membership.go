package ldap

// Membership represents the relationship between a member (user or group) and the group they are member of
type Membership struct {
	member   *Member
	memberOf *Member
}

// Memberships is a list of all Memberships
type Memberships []Membership

// GetMember returns the Member that this Membership is about
func (m *Membership) GetMember() (member Member) {
	return *m.member
}
