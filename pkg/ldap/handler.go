package ldap

import (
	"errors"

	"github.com/go-ldap/ldap/v3"
)

// Handler is the main struct that takes care of all heavy lifting
type Handler struct {
	config  Config
	conn    *ldap.Conn
	members Members
}

// NewLdapHandler can be used to instantiate a new Handler struct.
func NewLdapHandler(config Config) (lh *Handler) {
	config.setDefaults()
	return &Handler{
		config:  config,
		members: make(Members),
	}
}

func (lh *Handler) connect() (err error) {
	if lh.conn != nil {
		return nil
	}
	for i := 0; i < lh.config.MaxRetries; i++ {
		for _, server := range lh.config.Servers {
			conn, err := ldap.DialURL(server)
			if err != nil {
				continue
			}
			user, err := lh.config.user()
			if err != nil {
				return err
			}
			pwd, err := lh.config.password()
			if err != nil {
				return err
			}
			err = conn.Bind(user, pwd)
			if err != nil {
				return err
			}
			lh.conn = conn
			return nil
		}
	}
	return errors.New("none of the ldap servers are available")
}

// GetMembers can be used to get all ldap members of an LDAP group
func (lh Handler) GetMembers(baseDN string, filter string) (baseGroup *Member, err error) {
	err = lh.connect()
	if err != nil {
		return nil, err
	}
	baseGroup, err = lh.members.GetByID(baseDN, true)
	if err != nil {
		return nil, err
	}
	searchRequest := ldap.NewSearchRequest(baseDN, ldap.ScopeWholeSubtree, ldap.DerefAlways, 0, 0, false,
		filter, []string{"dn", "cn", "memberUid"}, nil)
	sr, err := lh.conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	for _, entry := range sr.Entries {
		group, err := lh.members.GetByID(entry.DN, true)
		if err != nil {
			return nil, err
		}
		group.addParent(baseGroup)
		for _, memberUID := range entry.GetAttributeValues("memberUid") {
			member, err := lh.members.GetByID(memberUID, true)
			if err != nil {
				return nil, err
			}
			member.addParent(group)
			err = member.setMType(userMType)
			if err != nil {
				return nil, err
			}
			log.Debugf("%s: %v", member.Name(), group.Name())
		}
	}
	return baseGroup, nil
}
