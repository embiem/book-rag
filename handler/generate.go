package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/embiem/book-rag/rag"
)

type GenerateRequest struct {
	Query string `json:"query"`
}

func HandleGenerate(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var payload GenerateRequest
	err := decoder.Decode(&payload)
	if err != nil {
		slog.Error("HandleGenerate: Body Decode error", "err", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request body"))
	}

	bookID, err := EnsureBookExists(r)
	if err != nil {
		if bookErr, ok := err.(HttpError); ok {
			w.WriteHeader(bookErr.Status)
			w.Write([]byte(bookErr.Msg))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}
		return
	}

	queryResult, err := QueryBook(r.Context(), QueryBookRequest{
		Query: payload.Query,
		Limit: 10,
	}, bookID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error querying the book"))
		return
	}

	prompt := fmt.Sprintf(`You are an assistant in a book publishing company. Your task is to help with the following query:
	"%s".

	Here is some context that we pulled from the book:
	
	---

	%s

	---

	Now help answering the following query: "%s"`, payload.Query, PrettifyPassages(queryResult.Passages), payload.Query)

	response, err := rag.GenerateText(r.Context(), prompt)
	if err != nil {
		slog.Error("Error during LLM generation", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Could not generate response"))
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}
