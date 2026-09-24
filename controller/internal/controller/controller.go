type Controller struct {
	registry *topology.SwitchRegistry
}

func New() *Controller {
	return &Controller{
		registry: topology.NewSwitchRegistry(),
	}
}

func (c *Controller) Start() {
	//configurara OpenFlow
	// arrancar servidor
}