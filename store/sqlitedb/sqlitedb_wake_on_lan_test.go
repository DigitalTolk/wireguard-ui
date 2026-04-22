package sqlitedb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/model"
)

func TestSaveAndGetWakeOnLanHost(t *testing.T) {
	db := newTestDB(t)

	now := time.Now().UTC()
	host := model.WakeOnLanHost{
		MacAddress: "AA:BB:CC:DD:EE:FF",
		Name:       "Test Host",
		LatestUsed: &now,
	}

	err := db.SaveWakeOnLanHost(host)
	require.NoError(t, err)

	got, err := db.GetWakeOnLanHost("AA:BB:CC:DD:EE:FF")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Test Host", got.Name)
	// MAC is normalized to uppercase dash format
	assert.Equal(t, "AA-BB-CC-DD-EE-FF", got.MacAddress)
}

func TestGetWakeOnLanHosts(t *testing.T) {
	db := newTestDB(t)

	db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "AA:BB:CC:DD:EE:01", Name: "Host1"})
	db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "AA:BB:CC:DD:EE:02", Name: "Host2"})

	hosts, err := db.GetWakeOnLanHosts()
	require.NoError(t, err)
	assert.Len(t, hosts, 2)
}

func TestDeleteWakeOnHostLanHost(t *testing.T) {
	db := newTestDB(t)

	db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "AA:BB:CC:DD:EE:FF", Name: "Del"})

	err := db.DeleteWakeOnHostLanHost("AA:BB:CC:DD:EE:FF")
	require.NoError(t, err)

	got, err := db.GetWakeOnLanHost("AA:BB:CC:DD:EE:FF")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestDeleteWakeOnHost(t *testing.T) {
	db := newTestDB(t)

	host := model.WakeOnLanHost{MacAddress: "AA:BB:CC:DD:EE:FF", Name: "Del"}
	db.SaveWakeOnLanHost(host)

	err := db.DeleteWakeOnHost(host)
	require.NoError(t, err)

	hosts, err := db.GetWakeOnLanHosts()
	require.NoError(t, err)
	assert.Len(t, hosts, 0)
}

func TestSaveWakeOnLanHost_Upsert(t *testing.T) {
	db := newTestDB(t)

	db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "AA:BB:CC:DD:EE:FF", Name: "Old"})
	db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "AA:BB:CC:DD:EE:FF", Name: "New"})

	got, err := db.GetWakeOnLanHost("AA:BB:CC:DD:EE:FF")
	require.NoError(t, err)
	assert.Equal(t, "New", got.Name)
}

func TestWakeOnLanHost_InvalidMac(t *testing.T) {
	db := newTestDB(t)
	err := db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "invalid", Name: "Bad"})
	assert.Error(t, err)
}

func TestGetWakeOnLanHost_InvalidMac(t *testing.T) {
	db := newTestDB(t)
	_, err := db.GetWakeOnLanHost("invalid-mac")
	assert.Error(t, err)
}

func TestGetWakeOnLanHost_NotFound(t *testing.T) {
	db := newTestDB(t)
	got, err := db.GetWakeOnLanHost("AA:BB:CC:DD:EE:99")
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestDeleteWakeOnHostLanHost_InvalidMac(t *testing.T) {
	db := newTestDB(t)
	err := db.DeleteWakeOnHostLanHost("invalid-mac")
	assert.Error(t, err)
}

func TestDeleteWakeOnHost_InvalidMac(t *testing.T) {
	db := newTestDB(t)
	err := db.DeleteWakeOnHost(model.WakeOnLanHost{MacAddress: "invalid-mac"})
	assert.Error(t, err)
}

func TestGetWakeOnLanHost_DashFormat(t *testing.T) {
	db := newTestDB(t)

	db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "AA:BB:CC:DD:EE:FF", Name: "DashTest"})

	// Query using dash format
	got, err := db.GetWakeOnLanHost("AA-BB-CC-DD-EE-FF")
	require.NoError(t, err)
	assert.Equal(t, "DashTest", got.Name)
}

func TestSaveWakeOnLanHost_WithNilLatestUsed(t *testing.T) {
	db := newTestDB(t)

	err := db.SaveWakeOnLanHost(model.WakeOnLanHost{MacAddress: "11:22:33:44:55:66", Name: "NilTime"})
	require.NoError(t, err)

	got, err := db.GetWakeOnLanHost("11:22:33:44:55:66")
	require.NoError(t, err)
	assert.Equal(t, "NilTime", got.Name)
	assert.Nil(t, got.LatestUsed)
}
