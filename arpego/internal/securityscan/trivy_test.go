package securityscan

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrivyScannerParsesFindings(t *testing.T) {
	workspace := t.TempDir()
	command := writeExecutable(t, workspace, `#!/bin/sh
cat <<'JSON'
{"Results":[
  {"Target":"workspace/file.go","Vulnerabilities":[
    {
      "VulnerabilityID":"CVE-1",
      "Title":"Critical vuln",
      "Severity":"CRITICAL",
      "PkgName":"stdlib",
      "InstalledVersion":"1.0",
      "FixedVersion":"1.1",
      "PrimaryURL":"https://example.com/CVE-1"
    }
  ]},
  {"Target":"workspace/Dockerfile","Misconfigurations":[
    {"ID":"DS001","Title":"Docker root user","Severity":"HIGH","PrimaryURL":"https://example.com/DS001"}
  ]},
  {"Target":"workspace/.env","Secrets":[
    {"RuleID":"secret-1","Title":"Hardcoded token","Severity":"HIGH"}
  ]}
]}
JSON
`)

	scanner := NewTrivyScanner(command, time.Second, time.Minute)
	result := scanner.ScanWorkspace(context.Background(), workspace)

	if result.Status != "findings" {
		t.Fatalf("status = %q want findings", result.Status)
	}
	if result.Summary.Total != 3 {
		t.Fatalf("total = %d want 3", result.Summary.Total)
	}
	if result.Summary.Critical != 1 {
		t.Fatalf("critical = %d want 1", result.Summary.Critical)
	}
	if result.Summary.High != 2 {
		t.Fatalf("high = %d want 2", result.Summary.High)
	}
	if result.ScannedAt == nil {
		t.Fatal("expected scanned_at to be set")
	}
	if len(result.Findings) != 3 {
		t.Fatalf("findings len = %d want 3", len(result.Findings))
	}
	if result.Findings[0].ID != "CVE-1" {
		t.Fatalf("first finding = %q want CVE-1", result.Findings[0].ID)
	}
}

func TestTrivyScannerReturnsUnavailableForMissingWorkspace(t *testing.T) {
	scanner := NewTrivyScanner("/bin/true", time.Second, time.Minute)
	result := scanner.ScanWorkspace(context.Background(), filepath.Join(t.TempDir(), "missing"))
	if result.Status != "unavailable" {
		t.Fatalf("status = %q want unavailable", result.Status)
	}
	if result.Error == nil {
		t.Fatal("expected error message")
	}
}

func TestTrivyScannerCachesResults(t *testing.T) {
	workspace := t.TempDir()
	counterPath := filepath.Join(workspace, "count.txt")
	command := writeExecutable(t, workspace, `#!/bin/sh
count_file="$1"
count=0
if [ -f "$count_file" ]; then
  count=$(cat "$count_file")
fi
count=$((count + 1))
printf "%s" "$count" > "$count_file"
cat <<'JSON'
{"Results":[]}
JSON
`)

	wrapper := writeExecutable(t, workspace, "#!/bin/sh\nexec "+command+" "+counterPath+"\n")
	scanner := NewTrivyScanner(wrapper, time.Second, time.Minute)
	_ = scanner.ScanWorkspace(context.Background(), workspace)
	_ = scanner.ScanWorkspace(context.Background(), workspace)

	data, err := os.ReadFile(counterPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "1" {
		t.Fatalf("counter = %q want 1", string(data))
	}
}

func writeExecutable(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "script-"+time.Now().Format("150405.000000000")+".sh")
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}
