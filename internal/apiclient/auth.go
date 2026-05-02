package apiclient

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/browser"

	"github.com/c3xdev/c3x/internal/logging"
	"github.com/c3xdev/c3x/internal/ui"
)

// AuthClient represents a client for C3X's authentication process.
type AuthClient struct {
	Host string
}

type callbackServerResp struct {
	err     error
	apiKey  string
	infoMsg string
}

// Login opens a browser with authentication URL and starts a HTTP server to
// wait for a callback request.
func (a AuthClient) Login(contextVals map[string]interface{}) (string, string, error) {
	state := uuid.NewString()

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", "", err
	}
	port := listener.Addr().(*net.TCPAddr).Port

	q := url.Values{}
	q.Set("cli_port", fmt.Sprint(port))
	q.Set("cli_state", state)
	q.Set("cli_version", fmt.Sprintf("%v", contextVals["version"]))
	q.Set("os", fmt.Sprintf("%v", contextVals["os"]))
	q.Set("utm_source", "cli")

	startURL := fmt.Sprintf("%s/login?%s", a.Host, q.Encode())

	fmt.Println("\nIf the redirect doesn't work, either:")
	fmt.Println("- Use this URL:")
	fmt.Printf("    %s\n", startURL)
	fmt.Println("\n- Or log in/sign up at https://dashboard.c3x.dev, copy your API key\n    from Org Settings and run `c3x configure set api_key MY_KEY`")
	fmt.Printf("\nWaiting...\n\n")

	_ = browser.OpenURL(startURL)

	apiKey, info, err := a.startCallbackServer(listener, state)
	if err != nil {
		return "", "", err
	}

	return apiKey, info, nil
}

func (a AuthClient) startCallbackServer(listener net.Listener, generatedState string) (string, string, error) {
	shutdown := make(chan callbackServerResp, 1)
	var once sync.Once

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			return
		}

		query := r.URL.Query()
		state := query.Get("cli_state")
		apiKey := query.Get("api_key")
		infoMsg := query.Get("info")
		redirectTo := query.Get("redirect_to")

		if state != generatedState {
			logging.Logger.Debug().Msg("Invalid state received")
			w.WriteHeader(400)
			return
		}

		u, err := url.Parse(redirectTo)
		if err != nil {
			logging.Logger.Debug().Msg("Unable to parse redirect_to URL")
			w.WriteHeader(400)
			return
		}

		origin := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		if origin != a.Host {
			logging.Logger.Debug().Msg("Invalid redirect_to URL")
			w.WriteHeader(400)
			return
		}

		http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect) //nolint:gosec // origin validated above
		// Flush the response, otherwise the HTTP redirect response doesn't always get sent
		// before the server shuts down.
		flusher, ok := w.(http.Flusher)
		if ok {
			flusher.Flush()
		}
		once.Do(func() {
			shutdown <- callbackServerResp{apiKey: apiKey, infoMsg: infoMsg}
		})
	})

	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer timeoutCancel()

	go func() {
		<-timeoutCtx.Done()
		if timeoutCtx.Err() == context.DeadlineExceeded {
			once.Do(func() {
				shutdown <- callbackServerResp{err: fmt.Errorf("timeout")}
			})
			srv.Close()
		}
	}()

	go func() {
		_ = srv.Serve(listener) // nolint: gosec
	}()

	resp := <-shutdown

	// Gracefully shut down the server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)

	if resp.infoMsg != "" {
		return "", resp.infoMsg, nil
	}

	if resp.apiKey == "" || resp.err != nil {
		return "", "", fmt.Errorf("Authentication failed. Please get your API token from %s", ui.LinkString("https://dashboard.c3x.dev"))
	}

	return resp.apiKey, "", nil
}
