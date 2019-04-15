//go:generate ./build/generate.sh

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

import (
	"github.com/gin-gonic/gin"
	"github.com/goproxyio/goproxy/pkg/proxy"
)

var listen string
var cacheDir string

func init() {
	log.SetOutput(os.Stdout)
	flag.StringVar(&cacheDir, "cacheDir", "", "go modules cache dir")
	flag.StringVar(&listen, "listen", "0.0.0.0:8081", "service listen address")
	flag.Parse()
	isGitValid := checkGitVersion()
	if !isGitValid {
		log.Fatal("Error in git version, please check your git installed in local, must be great 2.0")
	}
}

func checkGitVersion() bool {
	var err error
	var ret []byte
	cmd := exec.Command("git", "version")
	if ret, err = cmd.Output(); err != nil {
		return false
	}
	if strings.HasPrefix(string(ret), "git version 2") {
		return true
	}
	return false
}

func main() {
	errCh := make(chan error)

	log.Printf("goproxy: %s inited. listen on %s\n", time.Now().Format("2006-01-02 15:04:05"), listen)

	if cacheDir == "" {
		cacheDir = "/go"
		gpEnv := os.Getenv("GOPATH")
		if gpEnv != "" {
			gp := filepath.SplitList(gpEnv)
			if gp[0] != "" {
				cacheDir = gp[0]
			}
		}
	}
	fullCacheDir := filepath.Join(cacheDir, "pkg", "mod", "cache", "download")
	if _, err := os.Stat(fullCacheDir); os.IsNotExist(err) {
		log.Printf("goproxy: cache dir %s is not exist. To create it.\n", fullCacheDir)
		if err := os.MkdirAll(fullCacheDir, 0755); err != nil {
			log.Fatalf("goproxy: make cache dir failed: %s", err)
		}
	}

	var engine = gin.Default()
	var handle = proxy.NewProxy(cacheDir)

	engine.GET("/*action", func(ctx *gin.Context) {
		handle.ServeHTTP(ctx.Writer, ctx.Request)
		return
	})

	server := http.Server{
		Addr:    listen,
		Handler: engine,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			errCh <- err
		}
	}()

	signCh := make(chan os.Signal)
	signal.Notify(signCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		log.Fatal(err)
	case sign := <-signCh:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		log.Printf("goproxy: Server gracefully %s", sign)
	}
}
