# http-backoff
HTTP client with backoff for golang

```golang
func main() {
	credentials := clientcredentials.Config{
		ClientID:     "client_id",
		ClientSecret: "client_secret",
		TokenURL:     "token_url",
		Scopes:       []string{ "scope" },
	}

  opts := []backoff.Option{
		backoff.WithClient(backoff.NewOAuth2ClientWithOtel(credentials)),
		backoff.WithInitialInterval(100 * time.Millisecond),
		backoff.WithMaxInterval(1 * time.Second),
		backoff.WithMaxRetry(attempts),
		backoff.WithTimeout(timeout),
	}

  httpClient := backoff.NewBackoffClient(opts...)

}

```
