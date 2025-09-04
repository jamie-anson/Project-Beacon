package serverbind

import (
    "fmt"
    "net"
    "os"
    "path/filepath"
    "sync"
    "testing"
)

// helper to grab a free TCP port by binding :0 and returning the listener and addr
func getFreeListener(t *testing.T) (net.Listener, string) {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to get free listener: %v", err)
	}
	return ln, ln.Addr().String()
}

func TestResolveAndListen_Strict_FailureOnConflict(t *testing.T) {
	// Pre-bind 8090
	lnBusy, err := net.Listen("tcp", ":8090")
	if err != nil {
		t.Skip("cannot bind :8090 on this environment; skipping strict conflict test")
		return
	}
	defer lnBusy.Close()

	ln, addr, err := ResolveAndListen("strict", ":8090", 8090, 8099)
	if err == nil {
		ln.Close()
		t.Fatalf("expected strict to fail when :8090 is busy, got addr=%s", addr)
	}
}

func TestResolveAndListen_Fallback_ScansRange(t *testing.T) {
	// Pre-bind 8090 to force fallback scanning
	lnBusy, err := net.Listen("tcp", ":8090")
	if err != nil {
		t.Skip("cannot bind :8090 on this environment; skipping fallback scan test")
		return
	}
	defer lnBusy.Close()

	rangeStart, rangeEnd := 8091, 8093
	ln, addr, err := ResolveAndListen("fallback", ":8090", rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("expected fallback to find a free port in range, got error: %v", err)
	}
	defer ln.Close()
	if _, p, _ := net.SplitHostPort(addr); p != "8091" && p != "8092" && p != "8093" {
		t.Fatalf("expected port in [%d-%d], got %s", rangeStart, rangeEnd, addr)
	}
}

func TestResolveAndListen_Ephemeral_BindsAndAddrFile(t *testing.T) {
	ln, addr, err := ResolveAndListen("ephemeral", ":8090", 8090, 8099)
	if err != nil {
		t.Fatalf("ephemeral bind failed: %v", err)
	}
	defer ln.Close()
	if _, p, _ := net.SplitHostPort(addr); p == "8090" || p == "0" || p == "" {
		t.Fatalf("unexpected ephemeral port: %s", addr)
	}
	// addr file write
	dir := t.TempDir()
	path := filepath.Join(dir, ".runner-http.addr")
	if err := WriteAddrFile(path, addr); err != nil {
		t.Fatalf("write addr file failed: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read addr file failed: %v", err)
	}
	if string(b) != addr {
		t.Fatalf("addr file content mismatch: got %q want %q", string(b), addr)
	}
}

func TestResolveAndListen_Fallback_Exhaustion(t *testing.T) {
	// Pre-bind the entire range and 8090
	rangeStart, rangeEnd := 12000, 12005
	var listeners []net.Listener
	for p := rangeStart; p <= rangeEnd; p++ {
		ln, err := net.Listen("tcp", ":"+string(rune(0)) /*placeholder*/)
		_ = ln
		_ = err
	}
	// Correct binding loop creating listeners
	for p := rangeStart; p <= rangeEnd; p++ {
		ln, err := net.Listen("tcp", ":"+itoa(p))
		if err != nil {
			t.Fatalf("failed to pre-bind port %d: %v", p, err)
		}
		listeners = append(listeners, ln)
	}
	ln8090, err := net.Listen("tcp", ":8090")
	if err != nil {
		for _, l := range listeners { l.Close() }
		t.Skip("cannot bind :8090; skipping exhaustion test")
		return
	}
	defer func(){ ln8090.Close(); for _, l := range listeners { l.Close() } }()

	_, _, err = ResolveAndListen("fallback", ":8090", rangeStart, rangeEnd)
	if err == nil {
		t.Fatalf("expected exhaustion error when range fully busy")
	}
}

// simple itoa avoiding strconv import in tests
func itoa(i int) string {
	return fmtInt(i)
}

func fmtInt(i int) string {
	// minimal implementation
	return fmt.Sprintf("%d", i)
}

func TestResolveAndListen_Ephemeral_Concurrency_UniquePorts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency in short mode")
	}
	N := 10
	ports := make(map[string]struct{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			ln, addr, err := ResolveAndListen("ephemeral", ":8090", 8090, 8099)
			if err != nil {
				t.Errorf("ephemeral bind failed: %v", err)
				return
			}
			defer ln.Close()
			_, p, _ := net.SplitHostPort(addr)
			mu.Lock()
			ports[p] = struct{}{}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if len(ports) < N/2 { // allow some reuse on fast systems
		t.Fatalf("expected many unique ports, got %d unique for %d binds", len(ports), N)
	}
}
