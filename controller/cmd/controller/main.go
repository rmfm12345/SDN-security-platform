package main

import (
	"github.com/rmfm12345/SDN_Network_Security/controller/internal/controller"
)

func main() {
	ctrl := controller.New()
	ctrl.Start()
}
