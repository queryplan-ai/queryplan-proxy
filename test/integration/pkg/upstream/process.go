package upstream

type UpsreamProcess struct {
	done     chan error
	port     int
	password string
}

func (p *UpsreamProcess) Stop() {
	close(p.done)
}

func (p *UpsreamProcess) Port() int {
	return p.port
}
