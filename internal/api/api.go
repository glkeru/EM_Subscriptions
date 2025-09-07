package emsub

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	config "github.com/glkeru/EM_Subscriptions/internal/config"
	interfaces "github.com/glkeru/EM_Subscriptions/internal/interfaces"
	model "github.com/glkeru/EM_Subscriptions/internal/model"
	utils "github.com/glkeru/EM_Subscriptions/internal/utils"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

type Server struct {
	router *mux.Router
	repo   interfaces.RepoSubcription
	logger *zap.Logger
	config *config.Config
}

func NewServer(repo interfaces.RepoSubcription, logger *zap.Logger, c *config.Config) (*Server, error) {
	router := mux.NewRouter().StrictSlash(true)
	router.Use(MiddlewareLog(logger, c))
	server := &Server{router, repo, logger, c}

	router.HandleFunc("/api/v1/subscription", server.SubscriptionPing).Methods(http.MethodHead)
	router.HandleFunc("/api/v1/subscription", server.SubscriptionCreate).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/subscription/{id}", server.SubscriptionRead).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/subscription", server.SubscriptionList).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/subscription/{id}", server.SubscriptionUpdate).Methods(http.MethodPut)
	router.HandleFunc("/api/v1/subscription/{id}", server.SubscriptionPatch).Methods(http.MethodPatch)
	router.HandleFunc("/api/v1/subscription/{id}", server.SubscriptionDelete).Methods(http.MethodDelete)

	router.HandleFunc("/api/v1/total", server.SubscriptionTotal).Methods(http.MethodGet)

	return server, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, res *http.Request) {
	s.router.ServeHTTP(w, res)
}

// ping
func (s *Server) SubscriptionPing(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// логирование ошибок
func (s *Server) LogError(msg, service string, err error, data any) {
	s.logger.Error(msg,
		zap.String("service", service),
		zap.Error(err),
		zap.Any("data", data),
	)
}

// Create
func (s *Server) SubscriptionCreate(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		s.LogError("get request body", "SubscriptionCreate", err, nil)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	subreq := &SubscriptionFull{}
	err = json.Unmarshal(body, subreq)
	if err != nil {
		s.LogError("get JSON body", "SubscriptionCreate", err, string(body))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// обязательность полей
	if subreq.ServiceName == "" || subreq.Price <= 0 || subreq.UserId == uuid.Nil || subreq.StartDate == "" {
		s.LogError("missing required fields", "SubscriptionCreate", err, subreq)
		http.Error(w, "missing required fields, required: service_name, user_id, pruce, start_date", http.StatusBadRequest)
		return
	}

	// парсинг json
	subs := &model.Subscription{}
	subs.ServiceName = subreq.ServiceName
	subs.UserId = subreq.UserId
	subs.Price = subreq.Price
	subs.StartDate, err = utils.ParseDate(subreq.StartDate, DateFormat)
	if err != nil {
		s.LogError("start_date parsing error", "SubscriptionCreate", err, subs.StartDate)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if subreq.EndDate != "" {
		dt, err := utils.ParseDate(subreq.EndDate, DateFormat)
		if err != nil {
			s.LogError("end_date parsing error", "SubscriptionCreate", err, subs.EndDate)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		subs.EndDate = &dt
	}

	id, err := s.repo.SubscriptionCreate(req.Context(), *subs)
	if err != nil {
		s.LogError("DB create subscription", "SubscriptionCreate", err, subs)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	subresp := &SubscriptionCreateResponse{}
	subresp.Id = id

	r, err := json.Marshal(subresp)
	if err != nil {
		s.LogError("JSON marshal error", "SubscriptionCreate", err, subresp)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}

// Read
func (s *Server) SubscriptionRead(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		s.LogError("ID parse error", "SubscriptionRead", err, vars["id"])
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sub, err := s.repo.SubscriptionRead(req.Context(), id)
	if err != nil {

		if errors.Is(err, model.ErrNotFound) {
			s.LogError("Subscription not found", "SubscriptionRead", err, id)
			http.Error(w, "Subscription not found", http.StatusNotFound)
			return
		}

		s.LogError("DB read subscription", "SubscriptionRead", err, id)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	subresp := &SubscriptionFull{}
	subresp.Id = sub.Id
	subresp.ServiceName = sub.ServiceName
	subresp.UserId = sub.UserId
	subresp.Price = sub.Price
	subresp.StartDate = sub.StartDate.Format(DateFormat)
	if sub.EndDate != nil {
		subresp.EndDate = sub.EndDate.Format(DateFormat)
	}

	r, err := json.Marshal(subresp)
	if err != nil {
		s.LogError("JSON marshal error", "SubscriptionRead", err, subresp)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(r)
}

// Update (PUT)
func (s *Server) SubscriptionUpdate(w http.ResponseWriter, req *http.Request) {

	vars := mux.Vars(req)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		s.LogError("ID parse error", "SubscriptionUpdate", err, vars["id"])
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		s.LogError("get request body", "SubscriptionUpdate", err, nil)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// парсинг json
	subreq := &SubscriptionFull{}
	err = json.Unmarshal(body, subreq)
	if err != nil {
		s.LogError("get JSON body", "SubscriptionUpdate", err, string(body))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// обязательность полей
	if subreq.ServiceName == "" || subreq.Price <= 0 || subreq.UserId == uuid.Nil || subreq.StartDate == "" {
		s.LogError("missing required fields", "SubscriptionCreate", err, subreq)
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	subs := &model.Subscription{}
	subs.Id = id
	subs.ServiceName = subreq.ServiceName
	subs.UserId = subreq.UserId
	subs.Price = subreq.Price
	subs.StartDate, err = utils.ParseDate(subreq.StartDate, DateFormat)
	if err != nil {
		s.LogError("start_date parsing error", "SubscriptionUpdate", err, subs.StartDate)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if subreq.EndDate != "" {
		dt, err := utils.ParseDate(subreq.EndDate, DateFormat)
		if err != nil {
			s.LogError("end_date parsing error", "SubscriptionUpdate", err, subs.EndDate)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		subs.EndDate = &dt
	}

	err = s.repo.SubscriptionUpdate(req.Context(), *subs)
	if err != nil {

		if errors.Is(err, model.ErrNotFound) {
			s.LogError("Subscription not found", "SubscriptionUpdate", err, id)
			http.Error(w, "Subscription not found", http.StatusNotFound)
			return
		}

		s.LogError("DB update error", "SubscriptionUpdate", err, subs)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Update (PATCH)
func (s *Server) SubscriptionPatch(w http.ResponseWriter, req *http.Request) {

	vars := mux.Vars(req)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		s.LogError("ID parse error", "SubscriptionPatch", err, vars["id"])
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		s.LogError("get request body", "SubscriptionPatch", err, nil)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// парсинг json
	fields := make(map[string]any)
	err = json.Unmarshal(body, &fields)
	if err != nil || len(fields) == 0 {
		s.LogError("get JSON body", "SubscriptionPatch", err, string(body))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// проверим user_id и service_name
	if v, ok := fields["user_id"]; ok {
		if v == "" {
			s.LogError("user_id is required", "SubscriptionPatch", err, nil)
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}
	}
	if v, ok := fields["service_name"]; ok {
		if v == "" {
			s.LogError("service_name is required", "SubscriptionPatch", err, v)
			http.Error(w, "service_name is required", http.StatusBadRequest)
			return
		}
	}
	if v, ok := fields["price"]; ok {
		if v == "" {
			s.LogError("price is required", "SubscriptionPatch", err, v)
			http.Error(w, "price is required", http.StatusBadRequest)
			return
		}
	}

	// парсим дату начала
	if v, ok := fields["start_date"]; ok {
		str, ok := v.(string)
		if !ok {
			s.LogError("start_date parsing error", "SubscriptionPatch", err, v)
			http.Error(w, "start_date parsing error", http.StatusBadRequest)
			return
		}
		start, err := utils.ParseDate(str, DateFormat)
		if err != nil {
			s.LogError("start_date parsing error", "SubscriptionPatch", err, v.(string))
			http.Error(w, "start_date parsing error", http.StatusBadRequest)
			return
		}
		fields["start_date"] = start
	}
	// парсим дату окончания
	if v, ok := fields["end_date"]; ok {
		str, ok := v.(string)
		if !ok {
			s.LogError("end_date parsing error", "SubscriptionPatch", err, v)
			http.Error(w, "end_date parsing error", http.StatusBadRequest)
			return

		}
		end, err := utils.ParseDate(str, DateFormat)
		if err != nil {
			s.LogError("end_date parsing error", "SubscriptionPatch", err, v.(string))
			http.Error(w, "end_date parsing error", http.StatusBadRequest)
			return
		}
		fields["end_date"] = end
	}

	err = s.repo.SubscriptionPatch(req.Context(), id, fields)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			s.LogError("Subscription not found", "SubscriptionPatch", err, id)
			http.Error(w, "Subscription not found", http.StatusNotFound)
			return
		}
		s.LogError("DB update error", "SubscriptionPatch", err, id)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Delete
func (s *Server) SubscriptionDelete(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		s.LogError("ID parse error", "SubscriptionDelete", err, vars["id"])
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = s.repo.SubscriptionDelete(req.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			s.LogError("Subscription not found", "SubscriptionDelete", err, id)
			http.Error(w, "Subscription not found", http.StatusNotFound)
			return
		}

		s.LogError("DB delete subscription", "SubscriptionDelete", err, id)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// List
func (s *Server) SubscriptionList(w http.ResponseWriter, req *http.Request) {
	vars := req.URL.Query()
	var user uuid.UUID
	var service string
	var start *time.Time
	var end *time.Time
	var limit int
	var offset int
	var err error

	strid := vars.Get("user_id")
	if strid != "" {
		user, err = uuid.Parse(strid)
		if err != nil {
			s.LogError("user_id format is wrong", "SubscriptionList", err, nil)
			http.Error(w, "user_id format is wrong", http.StatusBadRequest)
			return
		}
	}
	service = vars.Get("service_name")
	limit, err = strconv.Atoi(vars.Get("limit"))
	offset, err = strconv.Atoi(vars.Get("offset"))

	strid = vars.Get("start_date")
	if strid != "" {
		startdate, err := utils.ParseDate(strid, DateFormat)
		if err != nil {
			s.LogError("start_date format is wrong", "SubscriptionList", err, nil)
			http.Error(w, "start_date format is wrong", http.StatusBadRequest)
			return
		}
		start = &startdate
	}
	strid = vars.Get("end_date")
	if strid != "" {
		enddate, err := utils.ParseDate(strid, DateFormat)
		if err != nil {
			s.LogError("end_date format is wrong", "SubscriptionList", err, nil)
			http.Error(w, "end_date format is wrong", http.StatusBadRequest)
			return
		}
		end = &enddate
	}

	subs, err := s.repo.SubscriptionList(req.Context(), user, service, start, end, limit, offset)
	if err != nil {
		s.LogError("DB list error", "SubscriptionList", err, vars)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lensub := len(subs)
	resp := &SubscriptionListResponse{}
	resp.Data = make([]SubscriptionFull, 0, lensub)
	resp.Limit = limit
	resp.Limit = s.config.Limit
	// вернем дефолтный лимит, если обрезали выборку
	if resp.Limit == 0 && lensub > s.config.Limit {
		resp.Limit = s.config.Limit
	}

	resp.Offset = offset
	for _, v := range subs {
		var rdata SubscriptionFull
		rdata.Id = v.Id
		rdata.UserId = v.UserId
		rdata.ServiceName = v.ServiceName
		rdata.Price = v.Price
		rdata.StartDate = v.StartDate.Format(DateFormat)
		if v.EndDate != nil {
			rdata.EndDate = v.EndDate.Format(DateFormat)
		}
		resp.Data = append(resp.Data, rdata)
	}

	r, err := json.Marshal(resp)
	if err != nil {
		s.LogError("JSON marshal error", "SubscriptionList", err, resp)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(r)
}

// List
func (s *Server) SubscriptionTotal(w http.ResponseWriter, req *http.Request) {
	vars := req.URL.Query()
	var user uuid.UUID
	var service string
	var start *time.Time
	var end *time.Time
	var err error

	strid := vars.Get("user_id")
	if strid != "" {
		user, err = uuid.Parse(strid)
		if err != nil {
			s.LogError("user_id format is wrong", "SubscriptionTotal", err, nil)
			http.Error(w, "user_id format is wrong", http.StatusBadRequest)
			return
		}
	}
	service = vars.Get("service_name")
	strid = vars.Get("start_date")
	if strid != "" {
		startdate, err := utils.ParseDate(strid, DateFormat)
		if err != nil {
			s.LogError("start_date format is wrong", "SubscriptionTotal", err, nil)
			http.Error(w, "start_date format is wrong", http.StatusBadRequest)
			return
		}
		start = &startdate
	}
	strid = vars.Get("end_date")
	if strid != "" {
		enddate, err := utils.ParseDate(strid, DateFormat)
		if err != nil {
			s.LogError("end_date format is wrong", "SubscriptionTotal", err, nil)
			http.Error(w, "end_date format is wrong", http.StatusBadRequest)
			return
		}
		end = &enddate
	}

	resp := &SubscriptionTotalResponse{}
	resp.Price, err = s.repo.SubscriptionTotal(req.Context(), user, service, start, end)
	if err != nil {
		s.LogError("DB list error", "SubscriptionTotal", err, vars)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	r, err := json.Marshal(resp)
	if err != nil {
		s.LogError("JSON marshal error", "SubscriptionTotal", err, resp)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(r)
}
