package metis

import (
	"embed"
	dash "github.com/b-harvest/metisian/metis/dashboard"
)

//go:embed static/*
var content embed.FS

func init() {
	dash.Content = content
}
