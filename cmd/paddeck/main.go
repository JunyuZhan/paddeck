package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/JunyuZhan/paddeck/internal/api"
	"github.com/JunyuZhan/paddeck/internal/collector"
	"github.com/JunyuZhan/paddeck/internal/config"
	"github.com/JunyuZhan/paddeck/internal/store"
	"github.com/JunyuZhan/paddeck/web"
)

// version is injected with -ldflags "-X main.version=v0.3.0".
var version = "dev"

func main() {
	var (
		configPath = flag.String("config", "", "config file (default: auto-discover config.yaml)")
		dbPath     = flag.String("db", "", "history database (default: /data/paddeck.db or ./data/paddeck.db)")
		initConfig = flag.Bool("init", false, "write a starter config.yaml and exit")
		force      = flag.Bool("force", false, "overwrite an existing file with -init")
		check      = flag.Bool("check", false, "validate the config and probe every target, then exit")
		showVer    = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("paddeck %s\n", version)
		return
	}

	if *initConfig {
		path := *configPath
		if path == "" {
			path = "config.yaml"
		}
		if err := config.WriteStarter(path, *force); err != nil {
			log.Fatalf("%v", err)
		}
		fmt.Printf("wrote %s\n\nNext:\n  1. edit it (hosts, services)\n"+
			"  2. run: paddeck -check\n  3. run: paddeck\n", path)
		return
	}

	path, err := config.Resolve(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		log.Fatalf("config %s: %v", path, err)
	}

	if *check {
		os.Exit(runCheck(cfg, path))
	}

	db := resolveDB(*dbPath)
	st, err := store.Open(db)
	if err != nil {
		log.Fatalf("store %s: %v", db, err)
	}
	defer st.Close()

	mgr := collector.NewManager(cfg, st)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go mgr.Start(ctx)
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := st.Prune(cfg.Collector.RetentionDays); err != nil {
					log.Printf("prune: %v", err)
				}
			}
		}
	}()

	static, err := fs.Sub(web.FS, ".")
	if err != nil {
		log.Fatalf("web: %v", err)
	}
	srv := &api.Server{Mgr: mgr, Store: st, Web: static}
	httpServer := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("PadDeck %s listening on %s (config %s, db %s)", version, cfg.Server.Listen, path, db)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	os.Exit(0)
}

// resolveDB keeps `paddeck` runnable from a checkout without flags.
func resolveDB(p string) string {
	if p != "" {
		return p
	}
	if env := os.Getenv("PADDECK_DB"); env != "" {
		return env
	}
	if st, err := os.Stat("/data"); err == nil && st.IsDir() {
		return "/data/paddeck.db"
	}
	_ = os.MkdirAll("./data", 0o755)
	return filepath.Join("data", "paddeck.db")
}

// runCheck prints a probe report and returns the process exit code.
func runCheck(cfg *config.Config, path string) int {
	fmt.Printf("PadDeck %s config check\n", version)
	fmt.Printf("config: %s  (%d hosts · %d pve · %d services)\n\n",
		path, len(cfg.Hosts), len(cfg.PVE), len(cfg.Services))
	if len(cfg.Hosts)+len(cfg.PVE)+len(cfg.Services) == 0 {
		fmt.Println("nothing configured yet - add hosts/pve/services to the config")
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	results := collector.Check(ctx, cfg)

	group := ""
	okCount, failed := 0, 0
	groupName := map[string]string{"host": "hosts", "pve": "pve", "service": "services"}
	for _, r := range results {
		if r.Group != group {
			if group != "" {
				fmt.Println()
			}
			group = r.Group
			fmt.Printf("%s\n", groupName[group])
		}
		mark := "ok  "
		if r.OK {
			okCount++
		} else {
			mark = "FAIL"
			failed++
		}
		fmt.Printf("  %s  %-14s %-18s %s\n", mark, r.Name, r.Target, r.Detail)
		if !r.OK && r.Hint != "" {
			fmt.Printf("        hint: %s\n", r.Hint)
		}
	}
	fmt.Printf("\n%d ok, %d failed\n", okCount, failed)
	if failed > 0 {
		return 1
	}
	return 0
}
