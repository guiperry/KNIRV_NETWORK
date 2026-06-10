package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type PolicyEvaluator interface {
	EvaluateAction(nodeID, action string, context map[string]interface{}) (bool, string)
}

type IdentityChecker interface {
	IsRevoked(nodeID string) bool
}

type GovernanceMiddleware struct {
	policyEvaluator PolicyEvaluator
	identityChecker IdentityChecker
}

func NewGovernanceMiddleware(pe PolicyEvaluator, ic IdentityChecker) *GovernanceMiddleware {
	return &GovernanceMiddleware{
		policyEvaluator: pe,
		identityChecker: ic,
	}
}

func (gm *GovernanceMiddleware) ZeroTrustMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID := c.GetHeader("X-Node-ID")
		agentID := c.GetHeader("X-Agent-ID")

		if nodeID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-Node-ID header required"})
			return
		}

		if gm.identityChecker != nil && gm.identityChecker.IsRevoked(nodeID) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "node is revoked"})
			return
		}

		action := c.Request.Method + " " + c.Request.URL.Path

		ctx := map[string]interface{}{
			"node_id":  nodeID,
			"agent_id": agentID,
			"path":     c.Request.URL.Path,
			"method":   c.Request.Method,
		}

		if gm.policyEvaluator != nil {
			allowed, reason := gm.policyEvaluator.EvaluateAction(nodeID, action, ctx)
			if !allowed {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": reason})
				return
			}
		}

		c.Set("node_id", nodeID)
		c.Set("agent_id", agentID)
		c.Next()
	}
}

func (gm *GovernanceMiddleware) ZeroTrustMiddlewareHTTP() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nodeID := r.Header.Get("X-Node-ID")
			agentID := r.Header.Get("X-Agent-ID")

			if nodeID == "" {
				http.Error(w, `{"error":"X-Node-ID header required"}`, http.StatusUnauthorized)
				return
			}

			if gm.identityChecker != nil && gm.identityChecker.IsRevoked(nodeID) {
				http.Error(w, `{"error":"node is revoked"}`, http.StatusForbidden)
				return
			}

			action := r.Method + " " + r.URL.Path

			ctx := map[string]interface{}{
				"node_id":  nodeID,
				"agent_id": agentID,
				"path":     r.URL.Path,
				"method":   r.Method,
			}

			if gm.policyEvaluator != nil {
				allowed, reason := gm.policyEvaluator.EvaluateAction(nodeID, action, ctx)
				if !allowed {
					http.Error(w, `{"error":"`+reason+`"}`, http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
