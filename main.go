package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/zerofox-oss/go-msg"
	"github.com/zerofox-oss/go-msg/mem"
	"go.uber.org/ratelimit"
)

// Processor processes messages as fast as the limiter will allow
type Processor struct {
	limiter ratelimit.Limiter
}

func (r *Processor) Reload() error {
	// reload ratelimit
	rl, err := strconv.Atoi(os.Getenv("RATELIMIT"))
	if err != nil {
		return err
	}
	r.limiter = ratelimit.New(rl)

	return nil
}

func (r *Processor) Receive(ctx context.Context, m *msg.Message) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		// this will block based on time
		now := r.limiter.Take()

		body, err := msg.DumpBody(m)
		if err != nil {
			log.Printf("[ERROR] could not read message body: %v", err)
		}
		log.Printf("[INFO] processing message %s at %s\n", string(body), now)
	}
	return nil
}

// loads environment variables from file
func loadEnv() {
	log.Printf("[INFO] reloading env")

	f := ".env"
	if err := godotenv.Overload(f); err != nil {
		log.Printf("[ERROR] Failed to load %s: %s", f, err.Error())
	}
}

func init() {
	loadEnv()
}

func main() {
	messageCount := 100000

	c := make(chan *msg.Message, messageCount)
	defer close(c)

	srv := mem.NewServer(c, 10)
	r := &Processor{}
	r.Reload()

	for i := 0; i < messageCount; i++ {
		c <- &msg.Message{
			Body: bytes.NewBufferString(fmt.Sprintf("<<hello world! i'm %d>>", i)),
		}
	}

	go func() {
		if err := srv.Serve(r); err != nil {
			fmt.Errorf("[ERROR] server crashed: %s", err)
		}
	}()

	// set up signal channel to listen to all os signals
	s := make(chan os.Signal)
	signal.Notify(s)

	log.Printf("[INFO] finished queueing messages, processing...")
	for {
		select {
		case sig := <-s:
			switch sig {
			case syscall.SIGHUP:
				// reload env
				log.Printf("[WARNING] SIGHUP issued, reloading env...")
				loadEnv()
				r.Reload()

			case syscall.SIGKILL:
				// hard shutdown
				log.Printf("[WARNING] SIGKILL issued, hard shutdown...")
				os.Exit(1)

			case syscall.SIGTERM:
				// soft shutdown
				log.Printf("[WARNING] SIGTERM issued, clean shutdown...")
				ctx, cancelFunc := context.WithCancel(context.Background())
				defer cancelFunc()

				if err := srv.Shutdown(ctx); err != nil {
					log.Printf("[ERROR] shutdown failed: %v", err)
				}
				os.Exit(0)

			default:
				// ignore, continue on
				log.Printf("[WARNING] Ignoring signal")
			}
		}
	}
}
