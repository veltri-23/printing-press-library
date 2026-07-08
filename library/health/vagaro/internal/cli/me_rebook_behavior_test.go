// Copyright 2026 Trevin Chow and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mvanhorn/printing-press-library/library/health/vagaro/internal/vagaro"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAppointments_flat(t *testing.T) {
	data := json.RawMessage(`{"status":"success","data":[
		{"appointmentId":9001,"businessId":93458,"businessName":"Central Barber","serviceId":34098477,"serviceName":"Skin Fade","serviceProviderId":43931725,"serviceProviderName":"Ronnel Getz","startDate":"2026-06-01"},
		{"appointmentId":9002,"businessId":55,"serviceId":66,"providerId":77}
	]}`)
	appts, err := parseAppointments(data)
	require.NoError(t, err)
	require.Len(t, appts, 2)

	assert.Equal(t, "9001", appts[0].AppointmentID)
	assert.Equal(t, "93458", appts[0].BusinessID)
	assert.Equal(t, "Central Barber", appts[0].BusinessName)
	assert.Equal(t, "34098477", appts[0].ServiceID)
	assert.Equal(t, "Skin Fade", appts[0].ServiceName)
	assert.Equal(t, "43931725", appts[0].ProviderID)
	assert.Equal(t, "Ronnel Getz", appts[0].ProviderName)

	assert.Equal(t, "55", appts[1].BusinessID)
	assert.Equal(t, "77", appts[1].ProviderID)
}

func TestParseAppointments_nested(t *testing.T) {
	data := json.RawMessage(`{"data":[
		{"id":1,"business":{"id":93458,"name":"Central Barber"},"service":{"id":34098477,"name":"Skin Fade"},"serviceProvider":{"id":43931725,"name":"Ronnel Getz"}}
	]}`)
	appts, err := parseAppointments(data)
	require.NoError(t, err)
	require.Len(t, appts, 1)
	assert.Equal(t, "93458", appts[0].BusinessID)
	assert.Equal(t, "Skin Fade", appts[0].ServiceName)
	assert.Equal(t, "43931725", appts[0].ProviderID)
}

func TestParseAppointments_empty(t *testing.T) {
	appts, err := parseAppointments(json.RawMessage(`{"status":"success","data":[]}`))
	require.NoError(t, err)
	assert.Empty(t, appts)

	// Non-JSON (e.g. an auth HTML page) is an error, not a silent empty.
	_, err = parseAppointments(json.RawMessage(`<html>login</html>`))
	assert.Error(t, err)
}

func TestFilterGroupsToWindow(t *testing.T) {
	groups := []vagaro.SlotGroup{
		{Date: "24 Jul 2026", Provider: "Ronnel", Times: []string{"10:00 AM", "11:00 AM"}},
		{Date: "31 Jul 2026", Provider: "George", Times: []string{"09:00 AM"}}, // outside window
	}
	from := time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	out := filterGroupsToWindow(groups, from, to)
	require.Len(t, out, 1)
	assert.Equal(t, "24 Jul 2026", out[0].Date)
	assert.Equal(t, []string{"10:00 AM", "11:00 AM"}, out[0].Times)
}
