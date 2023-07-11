package main

import (
	"embed"
	"github.com/gigurra/flycd/internal/flycd/util/util_packaged"
)

//go:embed Dockerfile
var PackagedDockerfile string

//go:embed LICENSE
var PackagedLICENSE string

//go:embed README.md
var PackagedREADME string

//go:embed go.mod
var PackagedGoMod string

//go:embed go.sum
var PackagedGoSum string

//go:embed *.go
var PackagedRootGoFiles embed.FS

//go:embed cmd/*
var PackagedCmd embed.FS

//go:embed internal/*
var PackagedInternal embed.FS

var PackagedFileSystem = util_packaged.PackagedFileSystem{
	Files: []util_packaged.PackagedFile{
		{Name: "Dockerfile", Contents: PackagedDockerfile},
		{Name: "LICENSE", Contents: PackagedLICENSE},
		{Name: "README.md", Contents: PackagedREADME},
		{Name: "go.mod", Contents: PackagedGoMod},
		{Name: "go.sum", Contents: PackagedGoSum},
	},
	Directories: []embed.FS{
		PackagedCmd,
		PackagedInternal,
		PackagedRootGoFiles,
	},
}
