package middleware

import (
    "errors"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    apperrors "github.com/jamie-anson/project-beacon-runner/internal/errors"
)

func TestGetHTTPStatusCode_Mapping(t *testing.T) {
    cases := []struct{
        et apperrors.ErrorType
        want int
    }{
        {apperrors.ValidationError, http.StatusBadRequest},
        {apperrors.NotFoundError, http.StatusNotFound},
        {apperrors.ConflictError, http.StatusConflict},
        {apperrors.AuthenticationError, http.StatusUnauthorized},
        {apperrors.AuthorizationError, http.StatusForbidden},
        {apperrors.ExternalServiceError, http.StatusBadGateway},
        {apperrors.CircuitBreakerError, http.StatusServiceUnavailable},
        {apperrors.TimeoutError, http.StatusRequestTimeout},
        {apperrors.DatabaseError, http.StatusInternalServerError},
        {apperrors.InternalError, http.StatusInternalServerError},
    }
    for _, c := range cases {
        if got := getHTTPStatusCode(c.et); got != c.want {
            t.Fatalf("mapping %s got %d want %d", c.et, got, c.want)
        }
    }
}

func TestHandleError_WithAppError(t *testing.T) {
    gin.SetMode(gin.TestMode)
    rr := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(rr)
    c.Set("request_id", "req-1")

    HandleError(c, apperrors.NewValidationError("bad"))

    if rr.Code != http.StatusBadRequest {
        t.Fatalf("status %d", rr.Code)
    }
    if got := rr.Header().Get("X-Error-Logged"); got != "" {
        t.Fatalf("unexpected X-Error-Logged for validation error")
    }
}

func TestHandleError_WrapsNonAppError(t *testing.T) {
    gin.SetMode(gin.TestMode)
    rr := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(rr)
    c.Set("request_id", "req-2")

    HandleError(c, errors.New("boom"))

    if rr.Code != http.StatusInternalServerError {
        t.Fatalf("status %d", rr.Code)
    }
    if rr.Header().Get("X-Error-Logged") != "true" {
        t.Fatalf("expected X-Error-Logged header for internal error")
    }
}

func TestErrorHandler_RecoversFromNonErrorPanic(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := gin.New()
    r.Use(ErrorHandler())
    r.GET("/panic", func(c *gin.Context) {
        panic("not-an-error")
    })

    rr := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/panic", nil)
    r.ServeHTTP(rr, req)

    if rr.Code != http.StatusInternalServerError {
        t.Fatalf("status %d", rr.Code)
    }
}
