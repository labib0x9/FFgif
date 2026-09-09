// Package architecture_test enforces the dependency direction of the DDD
// layout with a static import-graph check.
//
// The rule these tests encode is the one the layout already implies: domain and
// port packages describe *what* the system does and must stay free of the
// libraries that decide *how* — no driver SDKs, no infra, no transport. When a
// vendor type reaches a domain or port signature, every consumer of that
// interface inherits the dependency, mocks have to import it, and swapping the
// implementation stops being a local change.
//
// The check is transitive: it follows imports within this module, so a domain
// package that reaches a banned library through an intermediate package fails
// just as loudly as one that imports it directly.
package architecture_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const modulePath = "github.com/labib0x9/ffgif"

// repoRoot is this file's directory (tests/architecture) walked up twice.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("go.mod not found at %s: %v", root, err)
	}
	return root
}

// directImports returns the import paths of every non-test .go file in the
// package rooted at the given import path.
func directImports(t *testing.T, root, pkgPath string) []string {
	t.Helper()
	dir := filepath.Join(root, strings.TrimPrefix(pkgPath, modulePath+"/"))

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // package does not exist; caller's walk will notice
	}

	fset := token.NewFileSet()
	seen := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", filepath.Join(dir, name), err)
		}
		for _, imp := range f.Imports {
			seen[strings.Trim(imp.Path.Value, `"`)] = true
		}
	}

	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// packagesUnder lists every package directory beneath prefix that holds Go
// source, skipping generated mock packages (they legitimately import the
// package under test plus gomock).
func packagesUnder(t *testing.T, root, prefix string) []string {
	t.Helper()
	base := filepath.Join(root, strings.TrimPrefix(prefix, modulePath+"/"))

	var pkgs []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == "mocks" {
			return filepath.SkipDir
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				pkgs = append(pkgs, modulePath+"/"+filepath.ToSlash(rel))
				return nil
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", base, err)
	}
	sort.Strings(pkgs)
	return pkgs
}

// importChain does a DFS through in-module imports and returns the first path
// from `from` to a package matching banned, or nil if none exists.
func importChain(t *testing.T, root, from string, banned func(string) bool) []string {
	t.Helper()
	type frame struct {
		pkg   string
		chain []string
	}
	visited := map[string]bool{from: true}
	stack := []frame{{pkg: from, chain: []string{from}}}

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for _, imp := range directImports(t, root, cur.pkg) {
			if banned(imp) {
				return append(append([]string{}, cur.chain...), imp)
			}
			// only follow packages inside this module
			if !strings.HasPrefix(imp, modulePath+"/") || visited[imp] {
				continue
			}
			visited[imp] = true
			stack = append(stack, frame{pkg: imp, chain: append(append([]string{}, cur.chain...), imp)})
		}
	}
	return nil
}

func isStdlib(path string) bool {
	// stdlib import paths have no dot in their first segment
	first := path
	if i := strings.Index(path, "/"); i >= 0 {
		first = path[:i]
	}
	return !strings.Contains(first, ".")
}

// allowedThirdPartyInDomain lists vendor packages a domain type may legitimately
// name. uuid is a plain value type with no I/O behind it.
var allowedThirdPartyInDomain = map[string]bool{
	"github.com/google/uuid": true,
}

// ===========================================================================

// The domain layer must not name a driver SDK, an infra package, a transport
// package, or an application service.
func TestDomainPackagesDoNotDependOnInfrastructure(t *testing.T) {
	root := repoRoot(t)

	banned := func(imp string) bool {
		switch {
		case isStdlib(imp):
			return false
		case strings.HasPrefix(imp, modulePath+"/internal/infra"),
			strings.HasPrefix(imp, modulePath+"/internal/transport"),
			strings.HasPrefix(imp, modulePath+"/internal/app"),
			strings.HasPrefix(imp, modulePath+"/config"):
			return true
		case strings.HasPrefix(imp, modulePath+"/"):
			return false // another domain/port/pkg package
		default:
			return !allowedThirdPartyInDomain[imp]
		}
	}

	for _, pkg := range packagesUnder(t, root, modulePath+"/internal/domain") {
		t.Run(strings.TrimPrefix(pkg, modulePath+"/"), func(t *testing.T) {
			if chain := importChain(t, root, pkg, banned); chain != nil {
				t.Errorf("domain package reaches a forbidden dependency:\n  %s",
					strings.Join(chain, "\n    -> "))
			}
		})
	}
}

// EXPECTED TO FAIL: internal/port/queue/repository.go puts the RabbitMQ driver
// in the port itself —
//
//	ConsumeEmail(...) (<-chan amqp.Delivery, error)
//
// amqp091-go's Delivery is a broker-specific struct carrying an Acknowledger,
// DeliveryTag, routing keys and AMQP headers. Because it appears in the Queue
// interface, every consumer of that port, every generated mock and every test
// has to import the RabbitMQ SDK, and the port can no longer be satisfied by
// SQS, NATS, or an in-memory fake without dragging amqp091-go along.
//
// The port should expose its own message/acknowledgement type and let
// internal/infra/rabbitmq translate.
func TestPortPackagesDoNotLeakDriverTypes(t *testing.T) {
	root := repoRoot(t)

	// SDKs that must never appear in a port signature.
	driverSDKs := []string{
		"github.com/rabbitmq/amqp091-go",
		"github.com/minio/minio-go",
		"github.com/redis/go-redis",
		"github.com/jmoiron/sqlx",
		"github.com/lib/pq",
		"gopkg.in/gomail",
	}

	banned := func(imp string) bool {
		if isStdlib(imp) {
			return false
		}
		if strings.HasPrefix(imp, modulePath+"/internal/infra") ||
			strings.HasPrefix(imp, modulePath+"/internal/transport") {
			return true
		}
		for _, sdk := range driverSDKs {
			if strings.HasPrefix(imp, sdk) {
				return true
			}
		}
		return false
	}

	for _, pkg := range packagesUnder(t, root, modulePath+"/internal/port") {
		t.Run(strings.TrimPrefix(pkg, modulePath+"/"), func(t *testing.T) {
			if chain := importChain(t, root, pkg, banned); chain != nil {
				t.Errorf("port package leaks an infrastructure dependency into its contract:\n  %s",
					strings.Join(chain, "\n    -> "))
			}
		})
	}
}

// Application services must talk to infrastructure only through domain and port
// interfaces, never by importing a concrete adapter.
func TestAppServicesDoNotImportInfrastructure(t *testing.T) {
	root := repoRoot(t)

	banned := func(imp string) bool {
		return strings.HasPrefix(imp, modulePath+"/internal/infra") ||
			strings.HasPrefix(imp, modulePath+"/internal/transport")
	}

	for _, pkg := range packagesUnder(t, root, modulePath+"/internal/app") {
		t.Run(strings.TrimPrefix(pkg, modulePath+"/"), func(t *testing.T) {
			// Only direct imports are checked here: an app package legitimately
			// reaches infra types transitively through the ports it depends on.
			for _, imp := range directImports(t, root, pkg) {
				if banned(imp) {
					t.Errorf("%s imports the concrete adapter %s; it should depend on a port interface",
						pkg, imp)
				}
			}
		})
	}
}

// A transport handler must not reach past the application layer into a
// repository or a driver.
func TestHandlersDoNotImportInfrastructure(t *testing.T) {
	root := repoRoot(t)

	for _, pkg := range packagesUnder(t, root, modulePath+"/internal/transport") {
		t.Run(strings.TrimPrefix(pkg, modulePath+"/"), func(t *testing.T) {
			for _, imp := range directImports(t, root, pkg) {
				if strings.HasPrefix(imp, modulePath+"/internal/infra") {
					t.Errorf("%s imports the concrete adapter %s; handlers should go through internal/app",
						pkg, imp)
				}
			}
		})
	}
}
