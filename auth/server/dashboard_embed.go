package main

import (
	"embed"
)

//go:embed dashboard/*
var dashboardFiles embed.FS
