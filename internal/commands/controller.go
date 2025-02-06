package commands

import "io"

type PlayerController struct {
	Paused      bool
	AudioStream io.Seeker
}

var Ctrl = &PlayerController{}
