package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Bowbaq/bikage"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
)

func main() {
	env := get_env()

	server := new_server(env)
	go server.refresh_trips()
	server.Run(env)
}

type server struct {
	bk      *bikage.Bikage
	refresh chan *refresh_job
}

type credentials struct {
	Username string `binding:"required"`
	Password string `binding:"required"`
}

func new_server(env Env) *server {
	bikage, err := bikage.NewBikage(env["GOOGLE_APIKEY"], env["MONGODB_URI"])
	if err != nil {
		panic(err)
	}

	return &server{
		bk:      bikage,
		refresh: make(chan *refresh_job, 10),
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

	m.Post("/api/trips", binding.Json(credentials{}), s.TripsAPI)
	m.Post("/api/stats", binding.Json(credentials{}), s.StatsAPI)

	m.Run()
}

func (s *server) IndexHandler(r render.Render) {
	r.HTML(200, "home", nil)
}

func (s *server) StatsAPI(req *http.Request, r render.Render, creds credentials) {
	job := new_refresh_job(creds)
	s.refresh <- job

	if req.URL.Query().Get("cached") == "" {
		<-job.done
	}

	stats := s.bk.ComputeStats(s.bk.GetCachedTrips(creds.Username))

	one_month_ago := time.Now().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	last_month_dists := make([]float64, 0)
	last_month_speeds := make([]float64, 0)
	last_month_days := make([]string, 0)

	for day := one_month_ago; day.Before(time.Now()); day = day.AddDate(0, 0, 1) {
		last_month_days = append(last_month_days, day.Format("Jan 02"))
		if daily_dist, ok := stats.DailyDistanceTotal[day.Format("01/02/2006 EST")]; ok {
			last_month_dists = append(last_month_dists, float64(daily_dist)/1000)
		} else {
			last_month_dists = append(last_month_dists, 0)
		}
		if daily_speed, ok := stats.DailySpeedTotal[day.Format("01/02/2006 EST")]; ok {
			last_month_speeds = append(last_month_speeds, float64(daily_speed)/1000)
		} else {
			last_month_speeds = append(last_month_speeds, 0)
		}
	}

	data := struct {
		Distance       string
		Speed          string
		DailyDistances []float64
		DailySpeeds    []float64
		Days           []string
	}{
		Distance:       fmt.Sprintf("%.1f km (%.1f mi)", stats.TotalKm(), stats.TotalMi()),
		Speed:          fmt.Sprintf("%.1f km/h (%.1f mph)", stats.AvgSpeed, stats.AvgSpeed/1.60934),
		DailyDistances: last_month_dists,
		DailySpeeds:    last_month_speeds,
		Days:           last_month_days,
	}

	r.JSON(200, data)
}

func (s *server) TripsAPI(r render.Render, creds credentials) {
	job := new_refresh_job(creds)
	s.refresh <- job
	<-job.done

	r.JSON(200, s.bk.GetCachedTrips(creds.Username))
}

type refresh_job struct {
	creds credentials
	done  chan bool
}

type job_descriptor struct {
	last_run time.Time
	requests []*refresh_job
}

const job_refresh_interval = 15 * time.Minute

func new_refresh_job(creds credentials) *refresh_job {
	return &refresh_job{creds, make(chan bool, 1)}
}

func (s *server) refresh_trips() {
	var lock sync.Mutex
	jobs := make(map[string]*job_descriptor)

	for job := range s.refresh {
		lock.Lock()
		descriptor, exists := jobs[job.creds.Username]

		// return immediately if recently refreshed and not running
		if exists && len(descriptor.requests) == 0 && time.Since(descriptor.last_run) < job_refresh_interval {
			log.Println("Refresh ran recently for", job.creds.Username)
			job.done <- true
			lock.Unlock()
			continue
		}

		// job is currently running, add request to the list, will be signaled on completion
		if exists && len(descriptor.requests) > 0 {
			log.Println("Queuing signal for", job.creds.Username)
			descriptor.requests = append(descriptor.requests, job)
			lock.Unlock()
			continue
		}

		// job needs to be run
		jobs[job.creds.Username] = &job_descriptor{
			last_run: time.Now(),
			requests: []*refresh_job{job},
		}
		lock.Unlock()

		go func() {
			log.Println("Refreshing trips for", job.creds.Username)
			s.bk.GetTrips(job.creds.Username, job.creds.Password)

			lock.Lock()
			adescriptor := jobs[job.creds.Username]
			for _, req := range adescriptor.requests {
				req.done <- true
			}
			adescriptor.requests = []*refresh_job{}
			lock.Unlock()
		}()
	}
}
