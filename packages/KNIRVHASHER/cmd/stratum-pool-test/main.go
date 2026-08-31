// cmd/stratum-pool-test/main.go
//
// Standalone test harness for the "pool must be up before CGMiner starts"
// theory. Deliberately NOT wired into hasher-server/hasher-host or any
// deployment flow - run this directly on the host machine (go run .), point
// CGMiner's pool config at this host's LAN IP and the port below, and watch
// the logs. If CGMiner connects, subscribes, and eventually submits a real
// share, the theory holds and the full host-side migration is worth doing.
// If it never submits despite a long-lived, always-up listener, the theory
// is wrong and we've avoided a much larger wasted refactor.
//
// This reuses internal/driver/device's StratumServer directly - the same,
// already-verified code that runs inside hasher-server - so a positive
// result here transfers directly to the real implementation.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"knirvhasher/internal/driver/device"
)

func main() {
	bindIP := flag.String("bind", "0.0.0.0", "address to bind the test pool on (0.0.0.0 = all interfaces, so it's reachable from the Antminer over LAN)")
	port := flag.Int("port", 18332, "port to listen on")
	user := flag.String("user", "knirvhasher", "pool username CGMiner should be configured with")
	pass := flag.String("pass", "knirvhasher", "pool password CGMiner should be configured with")
	shareTimeout := flag.Duration("share-timeout", 60*time.Second, "how long to wait for CGMiner to submit a real share for each test job")
	subscribeTimeout := flag.Duration("subscribe-timeout", 5*time.Minute, "how long to wait for a client to connect and subscribe before giving up")
	difficulty := flag.Float64("difficulty", 1, "share difficulty to advertise via mining.set_difficulty and verify submits against. Some ASIC firmware (this device's cgminer was started with --bitmain-checkn2diff) may silently refuse to mine jobs below whatever difficulty floor it considers sane - try progressively higher values (e.g. 8, 64, 1024, 16384 - real pools in this device's own pool list have shown difficulties around 16K-65K) if difficulty 1 never produces a share despite a clean subscribe/authorize handshake")
	flag.Parse()

	srv, err := device.NewStratumServer(*bindIP, *port, *user, *pass)
	if err != nil {
		log.Fatalf("failed to start stratum server: %v", err)
	}
	defer srv.Close()
	srv.SetDifficulty(*difficulty)

	log.Printf("=================================================================")
	log.Printf("Stratum pool test harness listening on %s:%d (difficulty=%v)", *bindIP, *port, *difficulty)
	log.Printf("Configure CGMiner's pool as: URL=<this-host-LAN-IP>:%d  worker=%s  password=%s", *port, *user, *pass)
	log.Printf("This process must already be running BEFORE CGMiner (re)connects -")
	log.Printf("that's the entire point of this test.")
	log.Printf("=================================================================")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("Shutting down...")
		srv.Close()
		os.Exit(0)
	}()

	for i := 0; ; i++ {
		log.Printf("--- waiting for a subscribed CGMiner connection (up to %s) ---", *subscribeTimeout)
		if err := srv.WaitForSubscribedClient(*subscribeTimeout); err != nil {
			log.Printf("still no subscribed client: %v (retrying)", err)
			continue
		}
		log.Printf("client subscribed - publishing test job %d", i)

		job := srv.NewJob([]byte("stratum-pool-standalone-test"), i)
		if err := srv.SubmitJob(job); err != nil {
			log.Printf("job %d: failed to publish (%v), retrying in 5s", i, err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("job %d published, waiting up to %s for a real submitted share...", i, *shareTimeout)
		result, err := srv.WaitForResult(job, *shareTimeout)
		if err != nil {
			log.Printf("job %d: %v", i, err)
		} else {
			log.Printf("############################################################")
			log.Printf("### JOB %d SOLVED - nonce=%d hash=%x", i, result.Nonce(), result.Hash())
			log.Printf("### CGMiner IS mining real work against this pool. Theory confirmed.")
			log.Printf("############################################################")
		}

		time.Sleep(5 * time.Second)
	}
}
