package middleware

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func setupTestRouterForMiddleware(mw ...gin.HandlerFunc) *gin.Engine {
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()
    r.Use(gin.Recovery())
    r.POST("/test", append(mw, func(c *gin.Context) {
        c.Status(http.StatusNoContent)
    })...)
    return r
}

func TestValidateJSON_BlocksNonJSONOnPOST(t *testing.T) {
    r := setupTestRouterForMiddleware(ValidateJSON())
    req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("hello"))
    req.Header.Set("Content-Type", "text/plain")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d; body=%s", w.Code, w.Body.String())
    }
}

func TestValidateJSON_AllowsJSONContentType(t *testing.T) {
    r := setupTestRouterForMiddleware(ValidateJSON())
    req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString("{}"))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusNoContent {
        t.Fatalf("expected 204, got %d; body=%s", w.Code, w.Body.String())
    }
}

func routerWithJobSpec(mw ...gin.HandlerFunc) *gin.Engine {
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()
    r.Use(gin.Recovery())
    r.POST("/jobspec", append(mw, func(c *gin.Context) {
        // Ensure body is still readable after middleware
        var spec models.JobSpec
        if err := c.BindJSON(&spec); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        c.JSON(http.StatusOK, gin.H{"id": spec.ID})
    })...)
    return r
}

func TestValidateJobSpec_EmptyBody(t *testing.T) {
    r := routerWithJobSpec(ValidateJSON(), ValidateJobSpec())
    req := httptest.NewRequest(http.MethodPost, "/jobspec", http.NoBody)
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d; body=%s", w.Code, w.Body.String())
    }
}

func TestValidateJobSpec_MalformedJSON(t *testing.T) {
    r := routerWithJobSpec(ValidateJSON(), ValidateJobSpec())
    req := httptest.NewRequest(http.MethodPost, "/jobspec", bytes.NewBufferString("{ not-json }"))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d; body=%s", w.Code, w.Body.String())
    }
}

func TestValidateJobSpec_StructuralValidationError(t *testing.T) {
    r := routerWithJobSpec(ValidateJSON(), ValidateJobSpec())
    // Missing required fields => models.JobSpec.Validate() should error
    req := httptest.NewRequest(http.MethodPost, "/jobspec", bytes.NewBufferString(`{}`))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d; body=%s", w.Code, w.Body.String())
    }
}

func TestValidateJobSpec_ValidSpec_PassesAndBodyRestored(t *testing.T) {
    r := routerWithJobSpec(ValidateJSON(), ValidateJobSpec())

    spec := models.JobSpec{
        ID:      "mid-1",
        Version: "1.0",
        Benchmark: models.BenchmarkSpec{
            Name:      "Test",
            Container: models.ContainerSpec{Image: "alpine:latest"},
            Input:     models.InputSpec{Hash: "abc123"},
        },
        Constraints: models.ExecutionConstraints{Regions: []string{"US"}, MinRegions: 1, Timeout: 10 * time.Minute},
    }
    b, _ := json.Marshal(spec)

    req := httptest.NewRequest(http.MethodPost, "/jobspec", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
    }
}
