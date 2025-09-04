package middleware

import (
 "net"
 "net/http"
 "os"
 "strings"

 "github.com/gin-gonic/gin"
)

const (
 RoleAdmin     = "admin"
 RoleOperator  = "operator"
 RoleViewer    = "viewer"
 RoleAnonymous = "anonymous"
)

// token sets are derived from env; do not cache at init to allow tests to Setenv.
// ADMIN_TOKENS, OPERATOR_TOKENS, VIEWER_TOKENS are comma-separated.

func toSet(csv string) map[string]struct{} {
 set := make(map[string]struct{})
 if csv == "" {
  return set
 }
 parts := strings.Split(csv, ",")
 for _, p := range parts {
  t := strings.TrimSpace(p)
  if t != "" {
   set[t] = struct{}{}
  }
 }
 return set
}

func tokensFromEnv() (map[string]struct{}, map[string]struct{}, map[string]struct{}) {
 return toSet(os.Getenv("ADMIN_TOKENS")), toSet(os.Getenv("OPERATOR_TOKENS")), toSet(os.Getenv("VIEWER_TOKENS"))
}

func roleFromTokenSets(token string, admin, operator, viewer map[string]struct{}) string {
 if token == "" { return RoleAnonymous }
 if _, ok := admin[token]; ok { return RoleAdmin }
 if _, ok := operator[token]; ok { return RoleOperator }
 if _, ok := viewer[token]; ok { return RoleViewer }
 return RoleAnonymous
}

// AuthMiddleware extracts caller role from Authorization: Bearer <token>.
// Optionally elevates localhost callers to admin when ALLOW_LOCAL_NOAUTH=true.
func AuthMiddleware() gin.HandlerFunc {
 return func(c *gin.Context) {
  role := RoleAnonymous
  auth := c.GetHeader("Authorization")
  if strings.HasPrefix(auth, "Bearer ") {
   tok := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
   adm, op, vw := tokensFromEnv()
   role = roleFromTokenSets(tok, adm, op, vw)
  }
  // Local dev bypass
  if role == RoleAnonymous && os.Getenv("ALLOW_LOCAL_NOAUTH") == "true" {
   host := c.Request.Host
   ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
   if ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(host, "localhost") {
    role = RoleAdmin
   }
  }
  c.Set("role", role)
  c.Next()
 }
}

// GetRole returns role stored by AuthMiddleware, or anonymous.
func GetRole(c *gin.Context) string {
 if v, ok := c.Get("role"); ok {
  if s, ok2 := v.(string); ok2 { return s }
 }
 return RoleAnonymous
}

// RequireAnyRole enforces that the caller's role is one of the allowed roles.
func RequireAnyRole(roles ...string) gin.HandlerFunc {
 return func(c *gin.Context) {
  cur := GetRole(c)
  for _, r := range roles {
   if cur == r {
    c.Next()
    return
   }
  }
  c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden", "role": cur})
 }
}
