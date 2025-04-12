package pg

// StrictOptions can be set to have PgFga remove undefined users, databases, extensions or slots
type StrictOptions struct {
	Users      bool `yaml:"users"`
	Databases  bool `yaml:"databases"`
	Extensions bool `yaml:"extensions"`
	Slots      bool `yaml:"replication_slots"`
}
