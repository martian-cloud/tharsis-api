// Package main is used to preview emails during development
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
)

const (
	port          = "9001"
	serverTimeout = 3 * time.Second
)

var templateCtx = builder.NewTemplateContext(os.Getenv("UI_URL"), os.Getenv("EMAIL_FOOTER"))

func handler(w http.ResponseWriter, _ *http.Request) {
	// Set the content type to HTML
	w.Header().Set("Content-Type", "text/html")

	emailType := builder.EmailType(os.Getenv("EMAIL_TYPE"))
	if emailType == "" {
		fmt.Fprint(w, "EMAIL_TYPE environment variable is not set")
		return
	}

	emailBuilder, err := emailType.NewBuilder()
	if err != nil {
		fmt.Fprintf(w, "failed to create email builder: %v", err)
		return
	}

	data := os.Getenv("EMAIL_DATA")
	if err = emailBuilder.InitFromData([]byte(data)); err != nil {
		fmt.Fprintf(w, "failed to initialize email builder: %v", err)
		return
	}

	resp, err := emailBuilder.Build(templateCtx)
	if err != nil {
		fmt.Fprintf(w, "failed to generate HTML: %v", err)
		return
	}

	// Write the HTML content to the response
	fmt.Fprint(w, resp)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: serverTimeout,
	}

	// Start the server on port 9001
	fmt.Println("Server listening on http://localhost:" + port)
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("failed to start email preview server: %v", err)
	}
}
