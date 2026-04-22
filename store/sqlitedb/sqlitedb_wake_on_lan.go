package sqlitedb

import (
	"database/sql"

	"github.com/DigitalTolk/wireguard-ui/model"
)

// GetWakeOnLanHosts returns all Wake-on-LAN hosts
func (o *SqliteDB) GetWakeOnLanHosts() ([]model.WakeOnLanHost, error) {
	rows, err := o.db.Query("SELECT mac_address, name, latest_used FROM wake_on_lan_hosts")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []model.WakeOnLanHost
	for rows.Next() {
		var h model.WakeOnLanHost
		if err := rows.Scan(&h.MacAddress, &h.Name, &h.LatestUsed); err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}
	return hosts, rows.Err()
}

// GetWakeOnLanHost returns a single Wake-on-LAN host by MAC address
func (o *SqliteDB) GetWakeOnLanHost(macAddress string) (*model.WakeOnLanHost, error) {
	host := &model.WakeOnLanHost{MacAddress: macAddress}
	resourceName, err := host.ResolveResourceName()
	if err != nil {
		return nil, err
	}

	var h model.WakeOnLanHost
	err = o.db.QueryRow(
		"SELECT mac_address, name, latest_used FROM wake_on_lan_hosts WHERE mac_address = ?",
		resourceName,
	).Scan(&h.MacAddress, &h.Name, &h.LatestUsed)
	if err == sql.ErrNoRows {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// DeleteWakeOnHostLanHost deletes a Wake-on-LAN host by MAC address
func (o *SqliteDB) DeleteWakeOnHostLanHost(macAddress string) error {
	host := &model.WakeOnLanHost{MacAddress: macAddress}
	resourceName, err := host.ResolveResourceName()
	if err != nil {
		return err
	}
	_, err = o.db.Exec("DELETE FROM wake_on_lan_hosts WHERE mac_address = ?", resourceName)
	return err
}

// SaveWakeOnLanHost creates or updates a Wake-on-LAN host
func (o *SqliteDB) SaveWakeOnLanHost(host model.WakeOnLanHost) error {
	resourceName, err := host.ResolveResourceName()
	if err != nil {
		return err
	}
	host.MacAddress = resourceName

	_, err = o.db.Exec(
		`INSERT INTO wake_on_lan_hosts (mac_address, name, latest_used)
		 VALUES (?, ?, ?)
		 ON CONFLICT(mac_address) DO UPDATE SET
		   name = excluded.name,
		   latest_used = excluded.latest_used`,
		host.MacAddress, host.Name, host.LatestUsed,
	)
	return err
}

// DeleteWakeOnHost deletes a Wake-on-LAN host
func (o *SqliteDB) DeleteWakeOnHost(host model.WakeOnLanHost) error {
	resourceName, err := host.ResolveResourceName()
	if err != nil {
		return err
	}
	_, err = o.db.Exec("DELETE FROM wake_on_lan_hosts WHERE mac_address = ?", resourceName)
	return err
}
