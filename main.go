package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type apiConfig struct {
	fileserverHits int
}

func main() {
	const port = "8080"
	mux := http.NewServeMux()

	apiConfig := apiConfig{}

	mux.Handle("/app/*", apiConfig.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	})
	mux.Handle("GET /admin/metrics", http.HandlerFunc(apiConfig.getMetricHandler))
	mux.Handle("GET /api/reset", http.HandlerFunc(apiConfig.resetMetricHandler))
	mux.Handle("POST /api/validate_chirp", http.HandlerFunc(validateChripHandler))

	corsMux := middlewareCors(mux)

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, corsMux))
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) getMetricHandler(w http.ResponseWriter, r *http.Request) {
	template := `<html>
<body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
</body>
</html>
`

	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(template, cfg.fileserverHits)))
}

func (cfg *apiConfig) resetMetricHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = 0
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

func validateChripHandler(w http.ResponseWriter, r *http.Request) {
	type chrip struct {
		Body string `json:"body"`
	}
	type errorBody struct {
		Error string `json:"error"`
	}
	type successBody struct {
		Valid bool `json:"valid"`
	}
	w.Header().Set("Content-Type", "application/json")
	var statusCode int = 200
	var message []byte

	decoder := json.NewDecoder(r.Body)
	params := chrip{}
	err := decoder.Decode(&params)
	if err != nil {
		respBody := errorBody{
			Error: "Error decoding parameters",
		}
		message, _ = json.Marshal(respBody)
		statusCode = 500
	}

	if len(params.Body) > 140 {
		respBody := errorBody{
			Error: "Chirp is too long",
		}
		message, _ = json.Marshal(respBody)
		statusCode = 400
	} else {
		respBody := successBody{
			Valid: true,
		}
		message, _ = json.Marshal(respBody)
	}

	w.WriteHeader(statusCode)
	w.Write(message)

	return
}

func handlerChirpsValidate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type returnVals struct {
		Valid bool `json:"valid"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	const maxChirpLength = 140
	if len(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	respondWithJSON(w, http.StatusOK, returnVals{
		Valid: true,
	})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	if code > 499 {
		log.Printf("Responding with 5XX error: %s", msg)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}
