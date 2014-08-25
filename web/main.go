package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Bowbaq/bikage"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
)

func main() {
	env := get_env()

	server := new_server(env)
	go server.RefreshTrips()
	server.Run(env)
}

type server struct {
	bk      *bikage.Bikage
	refresh chan credentials
}

type credentials struct {
	Username string `form:"username" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func new_server(env Env) *server {
	bikage, err := bikage.NewBikage(env["GOOGLE_APIKEY"], env["MONGOHQ_URL"])
	if err != nil {
		panic(err)
	}

	return &server{
		bk:      bikage,
		refresh: make(chan credentials, 10),
	}
}

func (s *server) Run(env Env) {
	m := martini.Classic()

	// Setup middleware
	m.Use(secure_handler())
	if env["MARTINI_ENV"] == "production" {
		m.Use(new_relic_handler(env))
	}
	m.Use(gzip_handler())
	m.Use(render_handler())

	m.Get("/", s.IndexHandler)
	m.Post("/", binding.Form(credentials{}), s.StatsHandler)

	m.Get("/api/trips", binding.Json(credentials{}), s.TripsAPI)

	m.Run()
}

func (s *server) IndexHandler(r render.Render) {
	r.HTML(200, "home", nil)
}

func (s *server) StatsHandler(r render.Render, creds credentials, form_err binding.Errors) {
	if form_err.Len() > 0 {
		log.Println(form_err)
		r.Error(400)
		return
	}

	trips := s.bk.GetCachedTrips(creds.Username)
	s.refresh <- creds

	stats := s.bk.ComputeStats(trips)

	one_week_ago := time.Now().AddDate(0, 0, -6).Truncate(24 * time.Hour)
	last_week_dists := make([]float64, 0)
	last_week_days := make([]string, 0)

	for day := one_week_ago; day.Before(time.Now()); day = day.AddDate(0, 0, 1) {
		last_week_days = append(last_week_days, day.Format("Jan 02"))
		if daily_dist, ok := stats.DailyTotal[day.Format("01/02/2006")]; ok {
			last_week_dists = append(last_week_dists, float64(daily_dist)/1000)
		} else {
			last_week_dists = append(last_week_dists, 0)
		}
	}

	data := struct {
		Distance       string
		DailyDistances []float64
		Days           []string
	}{
		Distance:       fmt.Sprintf("%.1f km (%.1f mi)", stats.TotalKm(), stats.TotalMi()),
		DailyDistances: last_week_dists,
		Days:           last_week_days,
	}

	r.HTML(200, "stats", data)
}

func (s *server) TripsAPI(r render.Render, creds credentials) {
	trips, err := s.bk.GetTrips(creds.Username, creds.Password)
	if err != nil {
		r.Error(500)
		return
	}

	r.JSON(200, trips)
}

func (s *server) RefreshTrips() {
	jobs := make(map[string]time.Time)

	for creds := range s.refresh {
		// Skip running / recently ran refreshes
		if last_run, exist := jobs[creds.Username]; exist && time.Since(last_run) < 15*time.Minute {
			continue
		}

		log.Println("Refreshing trips for", creds.Username)
		jobs[creds.Username] = time.Now()
		go func() {
			s.bk.GetTrips(creds.Username, creds.Password)
		}()
	}
}
