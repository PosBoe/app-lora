package device

import (
	"net/http"
	"strconv"
	"time"

	// The Chisinau time zone must work in a container with no zoneinfo.
	_ "time/tzdata"

	"github.com/waneock/app-lora/core"
	"github.com/waneock/app-lora/pkg/clock"
	"github.com/waneock/app-lora/pkg/httpkit"
)

// dayZone is the time zone of the days of the pulse table: the product
// serves Moldova.
const dayZone = "Europe/Chisinau"

// pulseRanges are the day ranges that the pulse route takes.
var pulseRanges = map[int]bool{7: true, 30: true, 90: true, 365: true}

type readingHandler struct {
	devices *deviceService
	clock   clock.Clock
	guard   guard
	zone    *time.Location
}

func newReadingHandler(devices *deviceService, clk clock.Clock, g *core.Guard) (*readingHandler, error) {
	zone, err := time.LoadLocation(dayZone)
	if err != nil {
		return nil, err
	}
	return &readingHandler{devices: devices, clock: clk, guard: g, zone: zone}, nil
}

func registerReadingHandler(mux *http.ServeMux, h *readingHandler) {
	mux.Handle("GET /api/v1/devices/{id}/pulses", member(h.guard, h.pulses))
	mux.Handle("GET /api/v1/devices/{id}/alarms", member(h.guard, h.alarms))
}

type pulsePointResponse struct {
	At         time.Time `json:"at"`
	PulseCount int64     `json:"pulse_count"`
}

// dayPulseResponse is the pulse count of one day. Delta is null for the
// first day of a device, which has no day before it.
type dayPulseResponse struct {
	Day   string `json:"day"`
	Last  int64  `json:"last"`
	Delta *int64 `json:"delta"`
}

// pulsesResponse is the pulse data of a device. The frontend mirrors it in
// vue/src/device/use-readings.ts.
type pulsesResponse struct {
	Points []pulsePointResponse `json:"points"`
	Days   []dayPulseResponse   `json:"days"`
}

func (h *readingHandler) pulses(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	if _, err := h.devices.get(req.Context(), orgID, id); err != nil {
		return err
	}
	days := 30
	if n, err := strconv.Atoi(req.URL.Query().Get("days")); err == nil && pulseRanges[n] {
		days = n
	}
	from := startOfDay(h.clock.Now().In(h.zone)).AddDate(0, 0, -(days - 1))

	repo := newReadingRepo(h.devices.pool)
	points, err := repo.points(req.Context(), orgID, id, from)
	if err != nil {
		return err
	}
	dayRows, err := repo.days(req.Context(), orgID, id, from, dayZone)
	if err != nil {
		return err
	}
	prev, err := repo.lastBefore(req.Context(), orgID, id, from)
	if err != nil {
		return err
	}

	resp := pulsesResponse{
		Points: make([]pulsePointResponse, len(points)),
		Days:   make([]dayPulseResponse, len(dayRows)),
	}
	for i, p := range points {
		resp.Points[i] = pulsePointResponse(p)
	}
	for i, d := range dayRows {
		resp.Days[i] = dayPulseResponse{Day: d.Day.Format(time.DateOnly), Last: d.Last}
		if prev != nil {
			delta := d.Last - *prev
			resp.Days[i].Delta = &delta
		}
		last := d.Last
		prev = &last
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// alarmResponse is one alarm of a device. The frontend mirrors it in
// vue/src/device/use-readings.ts.
type alarmResponse struct {
	ID        int64      `json:"id"`
	Kind      string     `json:"kind"`
	RaisedAt  time.Time  `json:"raised_at"`
	ClearedAt *time.Time `json:"cleared_at"`
}

type alarmListResponse struct {
	Alarms []alarmResponse `json:"alarms"`
	Total  int             `json:"total"`
}

func (h *readingHandler) alarms(w http.ResponseWriter, req *http.Request, orgID int64) error {
	id, err := pathID(req)
	if err != nil {
		return err
	}
	if _, err := h.devices.get(req.Context(), orgID, id); err != nil {
		return err
	}
	offset := 0
	if page, err := strconv.Atoi(req.URL.Query().Get("page")); err == nil && page > 1 {
		offset = (page - 1) * defaultPerPage
	}
	rows, total, err := newReadingRepo(h.devices.pool).alarms(req.Context(), orgID, id, defaultPerPage, offset)
	if err != nil {
		return err
	}
	resp := alarmListResponse{Alarms: make([]alarmResponse, len(rows)), Total: total}
	for i, a := range rows {
		resp.Alarms[i] = alarmResponse(a)
	}
	httpkit.JSON(w, http.StatusOK, resp)
	return nil
}
