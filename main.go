package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/spiffe/go-spiffe/v2/svid/jwtsvid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
)

const socketPath = "unix:///tmp/spire-agent/public/api.sock"

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	clientOptions := workloadapi.WithClientOptions(workloadapi.WithAddr(socketPath))

	audience := "spire-test-s3"

	jwtSource, err := workloadapi.NewJWTSource(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("unable to create JWTSource: %w", err)
	}
	defer jwtSource.Close()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

loop:
	for {
		svid, err := jwtSource.FetchJWTSVID(ctx, jwtsvid.Params{
			Audience: audience,
		})
		if err != nil {
			return fmt.Errorf("unable to fetch SVID: %w", err)
		}

		path := "/tmp/token"
		content := svid.Marshal()

		f, err := os.Create(path)
		if err != nil {
			log.Fatal(err)
		}

		n, err := f.WriteString(content)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("wrote %d bytes\n", n)

		select {
		case <-ticker.C:
			continue
		case <-interrupt:
			break loop
		}
	}
	return nil
}
