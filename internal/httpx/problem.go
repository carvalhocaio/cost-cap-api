package httpx

import "net/http"

const problemContentType = "application/problem+json"

type Problem struct {
	Type     string       `json:"type"`
	Title    string       `json:"title"`
	Status   int          `json:"status"`
	Detail   string       `json:"detail,omitempty"`
	Instance string       `json:"instance,omitempty"`
	Errors   []FieldError `json:"errors,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func WriteProblem(w http.ResponseWriter, r *http.Request, status int, detail string) {
	writeProblem(w, r, Problem{Status: status, Detail: detail})
}

func writeProblem(w http.ResponseWriter, r *http.Request, problem Problem) {
	problem.Type = "about:blank"
	problem.Title = http.StatusText(problem.Status)
	problem.Instance = r.URL.Path

	if err := writeBody(w, problem.Status, problemContentType, problem); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
