package bikage

import (
	"fmt"
	"time"
)

type Route struct {
	From Station
	To   Station
}

func (r Route) String() string {
	return fmt.Sprintf("%s -> %s", r.From, r.To)
}

type Trip struct {
	Id        string
	Route     Route
	StartedAt time.Time
	EndedAt   time.Time
}

type Trips []Trip

func (t Trips) Len() int           { return len(t) }
func (t Trips) Less(i, j int) bool { return t[i].StartedAt.Before(t[j].StartedAt) }
func (t Trips) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

func (t Trip) String() string {
	return fmt.Sprintf(
		"(%s, start: %s, end: %s)",
		t.Route,
		t.StartedAt.Format("Jan 02, 15:04"),
		t.EndedAt.Format("Jan 02, 15:04"),
	)
}
