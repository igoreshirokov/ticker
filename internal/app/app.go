package app

import (
	_ "embed"
)

var AppName string

//go:embed assets/danger.ico
var IconBad []byte
//go:embed assets/info.ico
var IconGood []byte
