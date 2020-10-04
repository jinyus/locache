package locache

import "time"

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

func runJanitor(c *locache, interval time.Duration) {
	j := &janitor{
		Interval: interval,
		stop:     make(chan bool),
	}
	c.janitor = j
	go j.Run(c)
}

func stopJanitor(c *Locache) {
	c.janitor.stop <- true
}

func (j *janitor) Run(c *locache) {
	//println("starting janitor")
	ticker := time.NewTicker(j.Interval)
	for {
		select {
		case <-ticker.C:
			//println("\njanitor invoked")
			go c.DeleteExpired()
		case <-j.stop:
			//println("\ncleaning up")
			ticker.Stop()
			return
		}
	}
}
