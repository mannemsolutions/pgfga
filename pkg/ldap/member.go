package ldap

import (
	"errors"
	"strings"
)

// Member represents a member of a group, which could be either a user, or another group
type Member struct {
	dn       string
	pair     string
	name     string
	mType    memberType
	parents  Members
	children Members
}

func newMember(id string) (m *Member, err error) {
	m = &Member{
		parents:  Members{},
		children: Members{},
	}
	return m, m.setFromID(id)
}

// setFromID allows dn, id, and name to be set if they are not set yet, but determines it makes sense before doing so
func (m *Member) setFromID(id string) error {
	if m.dn != "" {
		return nil
	}
	if validDN(id) {
		pair := strings.Split(id, ",")[0]
		if m.pair != "" && m.pair != pair {
			return errors.New("trying to set dn, while pair is already set differently")
		}
		key := strings.Split(pair, "=")[0]
		name := strings.Split(pair, "=")[1]
		if m.name != "" && m.name != name {
			return errors.New("trying to set dn, while name is already set differently")
		}
		m.dn = id
		m.pair = pair
		m.name = name
		m.mType = getmemberType(key)
		return nil
	}
	if m.pair != "" {
		return nil
	}
	if validLDAPPair(id) {
		key := strings.Split(id, "=")[0]
		name := strings.Split(id, "=")[1]
		if m.name != "" && m.name != name {
			return errors.New("trying to set pair, while name is already set differently")
		}
		m.pair = id
		m.name = name
		m.mType = getmemberType(key)
	}
	if m.name != "" {
		return nil
	}
	m.name = id
	m.mType = unknownMType
	return nil
}

func (m *Member) setMType(mt memberType) (err error) {
	if mt == unknownMType || mt == m.mType {
		return nil
	}
	if m.mType != unknownMType {
		return errors.New("cannot set memberType when already set")
	}
	m.mType = mt
	return nil
}

// func (m Member) getMType() (mt memberType) {
// 	return m.mType
// }

// Name returns the name of this member
func (m Member) Name() (name string) {
	return m.name
}

// Pair returns the first pair in the DN
func (m Member) Pair() (pair string) {
	return m.pair
}

// DN returns the DN that represents this member
func (m Member) DN() (dn string) {
	return m.dn
}

func (m *Member) addParent(p *Member) {
	if m.dn == p.dn {
		// This is me, myself and I. Skipping.
		return
	}
	if _, exists := m.parents[p.name]; exists {
		// Already exists, so just return that one
		return
	}
	m.parents[p.name] = p
	p.children[m.name] = m
}

// MembershipTree returns a list of Memberships between Members (users and groups), and the groups they are member of
func (m Member) MembershipTree() (mss Memberships) {
	for _, member := range m.children {
		ms := Membership{
			member:   member,
			memberOf: &m,
		}
		mss = append(mss, ms)
		subMss := member.MembershipTree()
		mss = append(mss, subMss...)
	}
	return mss
}

// Members is a list of Members
type Members map[string]*Member

// GetByID can be used to find a Member object in a Members object by the id of the Member,
// which should be the name of the Member
func (ms Members) GetByID(id string, addWhenMissing bool) (m *Member, err error) {
	m, err = newMember(id)
	if err != nil {
		return m, err
	}
	if _, exists := ms[m.name]; exists {
		// Already exists, so just return that one
		return ms[m.name], nil
	}
	if !addWhenMissing {
		return &Member{}, nil
	}
	// ms is not a *Members, cause Members is already a points (map[string]Member).
	// So check if after leaving this method, that ms actually still holds the new values
	ms[m.name] = m
	ms[m.pair] = m
	ms[m.dn] = m
	return m, nil
}
