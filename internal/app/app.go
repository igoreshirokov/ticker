package app

import (
	_ "embed"
)

//go:embed assets/danger.ico
var IconBad []byte
//go:embed assets/info.ico
var IconGood []byte
