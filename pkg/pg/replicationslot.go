package pg

type replicationSlots map[string]replicationSlot

type replicationSlot struct {
	handler *Handler
	name    string
	State   State `yaml:"state"`
}

func newSlot(handler *Handler, name string) (rs *replicationSlot) {
	if rs, exists := handler.slots[name]; exists {
		return &rs
	}
	rs = &replicationSlot{
		handler: handler,
		name:    name,
		State:   Present,
	}
	handler.slots[name] = *rs
	return rs
}

func (rs replicationSlot) drop() (err error) {
	ph := rs.handler
	if !ph.strictOptions.Slots {
		log.Infof("skipping drop of replication slot %s (not running with strict option for slots", rs.name)
		return nil
	}
	exists, err := ph.conn.runQueryExists("SELECT slot_name FROM pg_replication_slots WHERE slot_name = $1", rs.name)
	if err != nil {
		return err
	}
	if exists {
		err = ph.conn.runQueryExec("SELECT pg_drop_physical_replication_slot($1)", rs.name)
		if err != nil {
			return err
		}
		log.Infof("Replication slot '%s' successfully dropped", rs.name)
	}
	return nil
}

func (rs replicationSlot) create() (err error) {
	conn := rs.handler.conn

	exists, err := conn.runQueryExists("SELECT slot_name FROM pg_replication_slots WHERE slot_name = $1", rs.name)
	if err != nil {
		return err
	}
	if !exists {
		err = conn.runQueryExec("SELECT pg_create_physical_replication_slot($1)", rs.name)
		if err != nil {
			return err
		}
		log.Infof("Created replication slot '%s'", rs.name)
	}
	return nil
}
