package integration

import (
	"fmt"
	"net"
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
	output, err := cmdBuild.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build proxy: %v\nOutput:\n%s", err, output)
	}
	defer os.Remove("proxy_test_bin")

	// 1. Start Mock Backend
	mockBackend := http.NewServeMux()
	mockBackend.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message": "Hello from mock backend"}`))
	})
	// We need a specific port or updated config.
	// Easier to listen on random port and generate config.
	backendListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen for mock backend: %v", err)
	}
	backendPort := backendListener.Addr().(*net.TCPAddr).Port
	backendServer := &http.Server{
		Handler: mockBackend,
	}
	go func() {
		_ = backendServer.Serve(backendListener)
	}()
	defer backendServer.Close()

	// 2. Generate Backend Config
	backendConfigTmpl := `
backends:
  - name: "mock-backend"
    locations:
      - path: "/api/v1"
        backend_url: "http://127.0.0.1:%d"
        timeout: 5s
        methods: ["GET", "POST"]
    auth:
      type: "basic"
      config:
        username: "admin"
        password: "password"
    transform:
      - type: "header"
        req:
          add_headers:
            X-Proxy-Added: "true"
`
	backendConfigFile, err := os.CreateTemp("", "backends-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	defer os.Remove(backendConfigFile.Name())
	if _, err := fmt.Fprintf(backendConfigFile, backendConfigTmpl, backendPort); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	backendConfigFile.Close()

	// 3. Start Proxy
	// We assume server.yaml exists or we can generate it too.
	// To avoid port conflict, let's generate server config too with random ports.
	proxyPort := getFreePort(t)
	adminPort := getFreePort(t)

	serverConfigContent := fmt.Sprintf(`
port: %d
admin_port: %d
read_timeout: 10s
write_timeout: 10s
idle_timeout: 60s
`, proxyPort, adminPort)
	serverConfigFile, err := os.CreateTemp("", "server-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp server config: %v", err)
	}
	defer os.Remove(serverConfigFile.Name())
	if _, err := serverConfigFile.WriteString(serverConfigContent); err != nil {
		t.Fatalf("Failed to write temp server config: %v", err)
	}
	serverConfigFile.Close()

	cmdProxy := exec.Command("./proxy_test_bin", "--server-config", serverConfigFile.Name(), "--backend-config", backendConfigFile.Name())
	// Capture output for debugging
	// cmdProxy.Stdout = os.Stdout
	// cmdProxy.Stderr = os.Stderr
	if err := cmdProxy.Start(); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer func() {
		if cmdProxy.Process != nil {
			_ = cmdProxy.Process.Kill()
		}
	}()

	// Wait for server to start
	if err := waitForPort(adminPort); err != nil {
		t.Fatalf("Proxy failed to start: %v", err)
	}

	// 1. Verify Health Check OK
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", adminPort))
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 2. Verify Proxy Request OK
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/api/v1/test", proxyPort), nil)
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
	resp, err = http.Post(fmt.Sprintf("http://127.0.0.1:%d/health/status", adminPort), "application/json", strings.NewReader(`{"healthy": false}`))
	if err != nil {
		t.Fatalf("Failed to set health: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// 4. Verify Health Check NG + Connection: close
	resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", adminPort))
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
	req, _ = http.NewRequest("GET", fmt.Sprintf("http://127.0.0.1:%d/api/v1/test", proxyPort), nil)
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

func getFreePort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to resolve tcp addr: %v", err)
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to listen tcp: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForPort(port int) error {
	for i := 0; i < 20; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for port %d", port)
}
