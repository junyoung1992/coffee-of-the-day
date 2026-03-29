package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"coffee-of-the-day/backend/internal/domain"
	"coffee-of-the-day/backend/internal/service"
)

// --- JSON 요청/응답 타입 ---
// 도메인 타입과 분리하여 HTTP 전송 포맷을 독립적으로 관리한다.

type cafeDetailJSON struct {
	CafeName    string   `json:"cafe_name"`
	Location    *string  `json:"location,omitempty"`
	CoffeeName  string   `json:"coffee_name"`
	BeanOrigin  *string  `json:"bean_origin,omitempty"`
	BeanProcess *string  `json:"bean_process,omitempty"`
	RoastLevel  *string  `json:"roast_level,omitempty"`
	TastingTags []string `json:"tasting_tags"`
	TastingNote *string  `json:"tasting_note,omitempty"`
	Impressions *string  `json:"impressions,omitempty"`
	Rating      *float64 `json:"rating,omitempty"`
}

type brewDetailJSON struct {
	BeanName      string   `json:"bean_name"`
	BeanOrigin    *string  `json:"bean_origin,omitempty"`
	BeanProcess   *string  `json:"bean_process,omitempty"`
	RoastLevel    *string  `json:"roast_level,omitempty"`
	RoastDate     *string  `json:"roast_date,omitempty"`
	TastingTags   []string `json:"tasting_tags"`
	TastingNote   *string  `json:"tasting_note,omitempty"`
	BrewMethod    string   `json:"brew_method"`
	BrewDevice    *string  `json:"brew_device,omitempty"`
	CoffeeAmountG *float64 `json:"coffee_amount_g,omitempty"`
	WaterAmountMl *float64 `json:"water_amount_ml,omitempty"`
	WaterTempC    *float64 `json:"water_temp_c,omitempty"`
	BrewTimeSec   *int     `json:"brew_time_sec,omitempty"`
	GrindSize     *string  `json:"grind_size,omitempty"`
	BrewSteps     []string `json:"brew_steps"`
	Impressions   *string  `json:"impressions,omitempty"`
	Rating        *float64 `json:"rating,omitempty"`
}

type coffeeLogResponse struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id"`
	RecordedAt string          `json:"recorded_at"`
	Companions []string        `json:"companions"`
	LogType    string          `json:"log_type"`
	Memo       *string         `json:"memo,omitempty"`
	CreatedAt  string          `json:"created_at"`
	UpdatedAt  string          `json:"updated_at"`
	Cafe       *cafeDetailJSON `json:"cafe,omitempty"`
	Brew       *brewDetailJSON `json:"brew,omitempty"`
}

type createLogRequest struct {
	RecordedAt string          `json:"recorded_at"`
	Companions []string        `json:"companions"`
	LogType    string          `json:"log_type"`
	Memo       *string         `json:"memo"`
	Cafe       *cafeDetailJSON `json:"cafe"`
	Brew       *brewDetailJSON `json:"brew"`
}

type updateLogRequest struct {
	RecordedAt string          `json:"recorded_at"`
	Companions []string        `json:"companions"`
	LogType    string          `json:"log_type"`
	Memo       *string         `json:"memo"`
	Cafe       *cafeDetailJSON `json:"cafe"`
	Brew       *brewDetailJSON `json:"brew"`
}

type listLogsResponse struct {
	Items      []coffeeLogResponse `json:"items"`
	NextCursor *string             `json:"next_cursor"`
	HasNext    bool                `json:"has_next"`
}

type errorResponse struct {
	Error string  `json:"error"`
	Field *string `json:"field,omitempty"`
}

// --- 핸들러 ---

// LogHandler는 커피 기록 관련 HTTP 요청을 처리한다.
// 비즈니스 로직은 LogService에 위임한다.
type LogHandler struct {
	svc service.LogService
}

func NewLogHandler(svc service.LogService) *LogHandler {
	return &LogHandler{svc: svc}
}

// CreateLog는 POST /api/v1/logs를 처리한다.
func (h *LogHandler) CreateLog(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req createLogRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "요청 본문을 파싱할 수 없습니다")
		return
	}

	log, err := h.svc.CreateLog(r.Context(), userID, service.CreateLogRequest{
		RecordedAt: req.RecordedAt,
		Companions: req.Companions,
		LogType:    domain.LogType(req.LogType),
		Memo:       req.Memo,
		Cafe:       cafeJSONToDomain(req.Cafe),
		Brew:       brewJSONToDomain(req.Brew),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, logToResponse(log))
}

// ListLogs는 GET /api/v1/logs를 처리한다.
func (h *LogHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	q := r.URL.Query()

	filter := service.ListLogsFilter{}

	if lt := q.Get("log_type"); lt != "" {
		logType := domain.LogType(lt)
		filter.LogType = &logType
	}
	if df := q.Get("date_from"); df != "" {
		filter.DateFrom = &df
	}
	if dt := q.Get("date_to"); dt != "" {
		filter.DateTo = &dt
	}
	if c := q.Get("cursor"); c != "" {
		filter.Cursor = &c
	}
	if ls := q.Get("limit"); ls != "" {
		n, err := strconv.Atoi(ls)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "limit은 0 이상의 정수여야 합니다")
			return
		}
		filter.Limit = n
	}

	result, err := h.svc.ListLogs(r.Context(), userID, filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	items := make([]coffeeLogResponse, len(result.Items))
	for i, item := range result.Items {
		items[i] = logToResponse(item)
	}

	writeJSON(w, http.StatusOK, listLogsResponse{
		Items:      items,
		NextCursor: result.NextCursor,
		HasNext:    result.HasNext,
	})
}

// GetLog는 GET /api/v1/logs/{id}를 처리한다.
func (h *LogHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	logID := chi.URLParam(r, "id")

	log, err := h.svc.GetLog(r.Context(), userID, logID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, logToResponse(log))
}

// UpdateLog는 PUT /api/v1/logs/{id}를 처리한다.
func (h *LogHandler) UpdateLog(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	logID := chi.URLParam(r, "id")

	var req updateLogRequest
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "요청 본문을 파싱할 수 없습니다")
		return
	}

	log, err := h.svc.UpdateLog(r.Context(), userID, logID, service.UpdateLogRequest{
		RecordedAt: req.RecordedAt,
		Companions: req.Companions,
		LogType:    domain.LogType(req.LogType),
		Memo:       req.Memo,
		Cafe:       cafeJSONToDomain(req.Cafe),
		Brew:       brewJSONToDomain(req.Brew),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, logToResponse(log))
}

// DeleteLog는 DELETE /api/v1/logs/{id}를 처리한다.
func (h *LogHandler) DeleteLog(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	logID := chi.URLParam(r, "id")

	if err := h.svc.DeleteLog(r.Context(), userID, logID); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- 공통 헬퍼 ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// writeServiceError는 서비스 계층 오류를 HTTP 상태 코드로 매핑한다.
// ValidationError는 field 정보를 함께 응답하여 프론트에서 인라인 오류를 표시할 수 있도록 한다.
func writeServiceError(w http.ResponseWriter, err error) {
	var ve *service.ValidationError
	switch {
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.As(err, &ve):
		field := ve.Field
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: ve.Message, Field: &field})
	default:
		writeError(w, http.StatusInternalServerError, "내부 오류가 발생했습니다")
	}
}

// --- 도메인 ↔ JSON 변환 ---

func logToResponse(log domain.CoffeeLogFull) coffeeLogResponse {
	resp := coffeeLogResponse{
		ID:         log.ID,
		UserID:     log.UserID,
		RecordedAt: log.RecordedAt,
		Companions: log.Companions,
		LogType:    string(log.LogType),
		Memo:       log.Memo,
		CreatedAt:  log.CreatedAt,
		UpdatedAt:  log.UpdatedAt,
	}
	// nil 슬라이스를 빈 배열로 보장: 프론트엔드에서 null 체크를 피하기 위함
	if resp.Companions == nil {
		resp.Companions = []string{}
	}
	if log.Cafe != nil {
		resp.Cafe = cafeDomainToJSON(log.Cafe)
	}
	if log.Brew != nil {
		resp.Brew = brewDomainToJSON(log.Brew)
	}
	return resp
}

func cafeDomainToJSON(c *domain.CafeDetail) *cafeDetailJSON {
	if c == nil {
		return nil
	}
	j := &cafeDetailJSON{
		CafeName:    c.CafeName,
		Location:    c.Location,
		CoffeeName:  c.CoffeeName,
		BeanOrigin:  c.BeanOrigin,
		BeanProcess: c.BeanProcess,
		TastingTags: c.TastingTags,
		TastingNote: c.TastingNote,
		Impressions: c.Impressions,
		Rating:      c.Rating,
	}
	if j.TastingTags == nil {
		j.TastingTags = []string{}
	}
	if c.RoastLevel != nil {
		s := string(*c.RoastLevel)
		j.RoastLevel = &s
	}
	return j
}

func brewDomainToJSON(b *domain.BrewDetail) *brewDetailJSON {
	if b == nil {
		return nil
	}
	j := &brewDetailJSON{
		BeanName:      b.BeanName,
		BeanOrigin:    b.BeanOrigin,
		BeanProcess:   b.BeanProcess,
		RoastDate:     b.RoastDate,
		TastingTags:   b.TastingTags,
		TastingNote:   b.TastingNote,
		BrewMethod:    string(b.BrewMethod),
		BrewDevice:    b.BrewDevice,
		CoffeeAmountG: b.CoffeeAmountG,
		WaterAmountMl: b.WaterAmountMl,
		WaterTempC:    b.WaterTempC,
		BrewTimeSec:   b.BrewTimeSec,
		GrindSize:     b.GrindSize,
		BrewSteps:     b.BrewSteps,
		Impressions:   b.Impressions,
		Rating:        b.Rating,
	}
	if j.TastingTags == nil {
		j.TastingTags = []string{}
	}
	if j.BrewSteps == nil {
		j.BrewSteps = []string{}
	}
	if b.RoastLevel != nil {
		s := string(*b.RoastLevel)
		j.RoastLevel = &s
	}
	return j
}

func cafeJSONToDomain(j *cafeDetailJSON) *domain.CafeDetail {
	if j == nil {
		return nil
	}
	d := &domain.CafeDetail{
		CafeName:    j.CafeName,
		Location:    j.Location,
		CoffeeName:  j.CoffeeName,
		BeanOrigin:  j.BeanOrigin,
		BeanProcess: j.BeanProcess,
		TastingTags: j.TastingTags,
		TastingNote: j.TastingNote,
		Impressions: j.Impressions,
		Rating:      j.Rating,
	}
	if j.RoastLevel != nil {
		r := domain.RoastLevel(*j.RoastLevel)
		d.RoastLevel = &r
	}
	return d
}

func brewJSONToDomain(j *brewDetailJSON) *domain.BrewDetail {
	if j == nil {
		return nil
	}
	d := &domain.BrewDetail{
		BeanName:      j.BeanName,
		BeanOrigin:    j.BeanOrigin,
		BeanProcess:   j.BeanProcess,
		RoastDate:     j.RoastDate,
		TastingTags:   j.TastingTags,
		TastingNote:   j.TastingNote,
		BrewMethod:    domain.BrewMethod(j.BrewMethod),
		BrewDevice:    j.BrewDevice,
		CoffeeAmountG: j.CoffeeAmountG,
		WaterAmountMl: j.WaterAmountMl,
		WaterTempC:    j.WaterTempC,
		BrewTimeSec:   j.BrewTimeSec,
		GrindSize:     j.GrindSize,
		BrewSteps:     j.BrewSteps,
		Impressions:   j.Impressions,
		Rating:        j.Rating,
	}
	if j.RoastLevel != nil {
		r := domain.RoastLevel(*j.RoastLevel)
		d.RoastLevel = &r
	}
	return d
}
