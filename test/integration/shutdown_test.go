package integration

import (
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestGracefulShutdown(t *testing.T) {
	// Build the proxy binary
	cmdBuild := exec.Command("go", "build", "-o", "proxy_test_bin", "../../cmd/proxy/main.go")
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("Failed to build proxy: %v", err)
	}
	defer os.Remove("proxy_test_bin")

	// Start the proxy
	cmdProxy := exec.Command("./proxy_test_bin", "--server-config", "test_server.yaml", "--backend-config", "test_backends.yaml")
	if err := cmdProxy.Start(); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer func() {
		if cmdProxy.Process != nil {
			_ = cmdProxy.Process.Kill()
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// 1. Verify Health Check OK
	resp, err := http.Get("http://localhost:9096/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Verify Proxy Request OK
	req, _ := http.NewRequest("GET", "http://localhost:8082/api/v1/test", nil)
	req.SetBasicAuth("admin", "password")
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to proxy request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 3. Set Unhealthy (Draining)
	resp, err = http.Post("http://localhost:9096/health/status", "application/json", strings.NewReader(`{"healthy": false}`))
	if err != nil {
		t.Fatalf("Failed to set health: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 4. Verify Health Check NG + Connection: close
	resp, err = http.Get("http://localhost:9096/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", resp.StatusCode)
	}
	if !resp.Close {
		t.Errorf("Expected resp.Close to be true")
	}
	resp.Body.Close()

	// 5. Verify Proxy Request OK + Connection: close
	req, _ = http.NewRequest("GET", "http://localhost:8082/api/v1/test", nil)
	req.SetBasicAuth("admin", "password")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to proxy request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if !resp.Close {
		t.Errorf("Expected resp.Close to be true")
	}
	resp.Body.Close()
}
