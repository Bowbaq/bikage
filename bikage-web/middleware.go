package main

import (
	"os"
	"strings"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/gorelic"
	"github.com/martini-contrib/gzip"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/secure"
)

type Env map[string]string

func get_env() Env {
	env := make(Env)

	for _, pair := range os.Environ() {
		parts := strings.Split(pair, "=")
		env[parts[0]] = parts[1]
	}

	return env
}

func secure_handler() martini.Handler {
	return secure.Secure(secure.Options{
		AllowedHosts:         []string{"bikage.herokuapp.com"},
		SSLRedirect:          true,
		SSLProxyHeaders:      map[string]string{"X-Forwarded-Proto": "https"},
		STSSeconds:           315360000,
		STSIncludeSubdomains: true,
		FrameDeny:            true,
		ContentTypeNosniff:   true,
		BrowserXssFilter:     true,
	})
}

func new_relic_handler(env Env) martini.Handler {
	gorelic.InitNewrelicAgent(env["NEW_RELIC_LICENSE_KEY"], env["NEW_RELIC_APP_NAME"], false)
	return gorelic.Handler
}

func gzip_handler() martini.Handler {
	return gzip.All()
}

func render_handler() martini.Handler {
	return render.Renderer(render.Options{})
}
