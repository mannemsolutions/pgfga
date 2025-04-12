package pg

// Handler holds all data for the Handle Method.
type Handler struct {
	conn          *Conn
	strictOptions StrictOptions
	databases     Databases
	roles         Roles
	slots         replicationSlots
}

// NewPgHandler can be used to handle all PostgreSQL actions tha PgFga needs to undertake
func NewPgHandler(connParams DSN, options StrictOptions, databases Databases, slots []string) (ph *Handler) {
	ph = &Handler{
		conn:          NewConn(connParams),
		strictOptions: options,
		databases:     databases,
		roles:         make(Roles),
		slots:         make(replicationSlots),
	}
	for _, slotName := range slots {
		slot := newSlot(ph, slotName)
		ph.slots[slotName] = *slot
	}
	ph.setDefaults()
	return ph
}

func (ph *Handler) setDefaults() {
	for name, db := range ph.databases {
		db.handler = ph
		db.name = name
		db.setDefaults()
	}
	for name, rs := range ph.slots {
		rs.handler = ph
		rs.name = name
	}
}

func (ph *Handler) getDb(dbName string) (d *Database) {
	// NewDatabase does everything we need to do
	return newDatabase(ph, dbName, "")
}

// GetRole will get the requested role, creating it if needed.
func (ph *Handler) GetRole(roleName string) (d *Role, err error) {
	// NewRole does everything we need to do
	return NewRole(ph, roleName, RoleOptionMap{}, Present)
}

// GrantRole will grant a role to to another user / role
func (ph *Handler) GrantRole(granteeName string, grantedName string) (err error) {
	// NewDatabase does everything we need to do
	grantee, err := ph.GetRole(granteeName)
	if err != nil {
		return err
	}
	granted, err := ph.GetRole(grantedName)
	if err != nil {
		return err
	}
	return grantee.GrantRole(granted)
}

// CreateOrDropDatabases will create databases if needed, and (if strict option is enabled for databases) will drop
// databases that should not exist.
func (ph *Handler) CreateOrDropDatabases() (err error) {
	for _, d := range ph.databases {
		if d.State.Bool() {
			err = d.create()
		} else {
			err = d.drop()
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateOrDropSlots will create database slots if needed, and (if strict option is enabled for databases) will drop
// slots that should not exist.
func (ph *Handler) CreateOrDropSlots() (err error) {
	for _, d := range ph.slots {
		if d.State.Bool() {
			err = d.create()
		} else {
			err = d.drop()
		}
		if err != nil {
			return err
		}
	}
	return nil
}
