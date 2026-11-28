package main

import (
	"embed"
	"io/fs"
)

//go:embed dashboard/*
var dashboardFiles embed.FS
