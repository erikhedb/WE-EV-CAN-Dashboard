package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("../../data/state.json")
		if err != nil {
			http.Error(w, "state.json not available", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(w, f) // stream file content as-is
	})

	// Serve static files (index.html etc.)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	log.Println("UI server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
