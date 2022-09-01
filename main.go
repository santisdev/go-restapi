package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sync"
)

var (
	listUsersRe  = regexp.MustCompile(`^\/users[\/]*$`)
	getUserRe    = regexp.MustCompile(`^\/users\/(\d+)*$`)
	createUserRe = regexp.MustCompile(`^\/users[\/]*$`)
	deleteUserRe = regexp.MustCompile(`^\/users\/(\d+)*$`)
)

type user struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// memory data
type datastore struct {
	m             map[string]user
	*sync.RWMutex //mutex to manage concurrently reading and writting
}
type userHandler struct {
	store *datastore
}

func (h *userHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	switch {
	case r.Method == http.MethodGet && listUsersRe.MatchString(r.URL.Path):
		h.List(w, r)
		return

	case r.Method == http.MethodGet && getUserRe.MatchString(r.URL.Path):
		h.Get(w, r)
		return

	case r.Method == http.MethodPost && createUserRe.MatchString(r.URL.Path):
		h.Create(w, r)
		return

	case r.Method == http.MethodDelete && deleteUserRe.MatchString(r.URL.Path):
		h.Delete(w, r)
		return

	default:
		notFound(w, r) // if we don't match any paths
		return
	}

}

func (h *userHandler) List(w http.ResponseWriter, r *http.Request) {
	users := make([]user, 0, len(h.store.m))
	h.store.RLock() //use the mutex to lock the read access
	for _, u := range h.store.m {
		users = append(users, u)
	}
	h.store.RUnlock()
	jsonBytes, err := json.Marshal(users)
	if err != nil {
		internalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *userHandler) Get(w http.ResponseWriter, r *http.Request) {
	//Get the user id
	matches := getUserRe.FindStringSubmatch(r.URL.Path) //first match is the whole string
	if len(matches) < 2 {
		notFound(w, r)
		return
	}
	h.store.RLock()
	user, ok := h.store.m[matches[1]]
	h.store.RUnlock()
	if !ok {
		notFound(w, r) //change it to usernotfound
		return
	}
	jsonBytes, err := json.Marshal(user)
	if err != nil {
		internalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *userHandler) Create(w http.ResponseWriter, r *http.Request) {
	u := user{}
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		badRequest(w, r)
		return
	}
	h.store.Lock()
	h.store.m[u.ID] = u
	h.store.Unlock()

	jsonBytes, err := json.Marshal(u)
	if err != nil {
		internalServerError(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)

}

func (h *userHandler) Delete(w http.ResponseWriter, r *http.Request) {
	//Get the user id
	matches := getUserRe.FindStringSubmatch(r.URL.Path) //first match is the whole string
	if len(matches) < 2 {
		notFound(w, r)
		return
	}

	h.store.RLock()
	user, ok := h.store.m[matches[1]]
	h.store.RUnlock()
	if !ok {
		notFound(w, r) //change it to usernotfound
		return
	}
	h.store.Lock()
	delete(h.store.m, matches[1])
	h.store.Unlock()

	jsonBytes, err := json.Marshal(user)
	if err != nil {
		internalServerError(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(`{"error": "not found"}`))
}

func badRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(`{"error": "bad request"}`))
}

func internalServerError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(`{"error": "internal server error"}`))
}

func main() {
	mux := http.NewServeMux()

	//initialize user handler
	userH := &userHandler{
		store: &datastore{
			m: map[string]user{
				"1": user{
					ID:   "1",
					Name: "Charles",
				},
			},
			RWMutex: &sync.RWMutex{},
		},
	}
	mux.Handle("/users/", userH)
	http.ListenAndServe("localhost:8080", mux)
}
