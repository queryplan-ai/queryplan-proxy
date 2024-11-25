package upstream

import "fmt"

type UpsreamProcess struct {
	stopAndDelete     chan struct{}
	stoppedAndDeleted chan error

	port     int
	password string
}

func (p *UpsreamProcess) Stop() {
	close(p.stopAndDelete)

	err := <-p.stoppedAndDeleted
	if err != nil {
		fmt.Printf("Error stopping and deleting upstream process: %v\n", err)
	} else {
		fmt.Println("MySQL container stopped and deleted successfully.")
	}
}

func (p *UpsreamProcess) Port() int {
	return p.port
}
