package main

import (
	"fmt"
	"net/http"
	"time"

	"fortio.org/fortio/fhttp"
	"fortio.org/fortio/fnet"
	"fortio.org/fortio/periodic"
	"github.com/magefile/mage/sh"
)

// LoadTest runs load tests against the ftw deployment.
func LoadTest() error {
	for _, threads := range []int{1, 2, 4} {
		for _, payloadSize := range []int{0, 100, 1000, 10000} {
			for _, conf := range []string{"envoy-config.yaml", "envoy-config-nowasm.yaml"} {
				if err := doLoadTest(conf, payloadSize, threads); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func doLoadTest(conf string, payloadSize int, threads int) error {
	if err := sh.RunV("docker-compose", "--file", "ftw/docker-compose.yml", "build", "--pull"); err != nil {
		return err
	}
	defer func() {
		_ = sh.RunV("docker-compose", "--file", "ftw/docker-compose.yml", "kill")
		_ = sh.RunV("docker-compose", "--file", "ftw/docker-compose.yml", "down", "-v")
	}()
	if err := sh.RunWithV(map[string]string{"ENVOY_CONFIG": fmt.Sprintf("/conf/%s", conf)}, "docker-compose",
		"--file", "ftw/docker-compose.yml", "run", "--service-ports", "--rm", "-d", "envoy"); err != nil {
		return err
	}

	// Wait for Envoy to start.
	for i := 0; i < 1000; i++ {
		if resp, err := http.Get("http://localhost:8080/anything"); err != nil {
			continue
		} else {
			if resp.Body != nil {
				resp.Body.Close()
			}
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	opts := &fhttp.HTTPRunnerOptions{
		RunnerOptions: periodic.RunnerOptions{
			QPS:        100,
			NumThreads: threads,
			Duration:   10 * time.Second,
		},
		HTTPOptions: fhttp.HTTPOptions{
			URL:     "http://localhost:8080/anything",
			Payload: fnet.GenerateRandomPayload(payloadSize),
		},
	}

	fmt.Printf("Running load test with config=%s, payloadSize=%d, threads=%d\n", conf, payloadSize, threads)
	res, err := fhttp.RunHTTPTest(opts)
	if err != nil {
		return err
	}
	rr := res.Result()
	fmt.Printf("All done %d calls (plus %d warmup) %.3f ms avg, %.1f qps\n",
		rr.DurationHistogram.Count,
		0,
		1000.*rr.DurationHistogram.Avg,
		rr.ActualQPS)

	return nil
}
