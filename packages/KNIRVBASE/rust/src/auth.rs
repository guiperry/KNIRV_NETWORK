use base64::{engine::general_purpose::STANDARD as BASE64, Engine as _};
use chrono::{DateTime, Duration, Utc};
use hmac::{Hmac, Mac};
use serde::{Deserialize, Serialize};
use sha2::Sha256;
use std::collections::HashMap;

type HmacSha256 = Hmac<Sha256>;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum Permission {
    ReadOnly,
    ReadWrite,
    Admin,
}

impl Permission {
    pub fn as_str(&self) -> &'static str {
        match self {
            Permission::ReadOnly => "read",
            Permission::ReadWrite => "write",
            Permission::Admin => "admin",
        }
    }

    pub fn from_str(s: &str) -> Option<Self> {
        match s {
            "read" => Some(Permission::ReadOnly),
            "write" => Some(Permission::ReadWrite),
            "admin" => Some(Permission::Admin),
            _ => None,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    pub user_id: String,
    pub wallet_addr: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub session_token: Option<String>,
    pub permissions: Vec<Permission>,
    pub exp: i64,
    pub iat: i64,
    pub nbf: i64,
}

impl Claims {
    pub fn new(
        user_id: String,
        wallet_addr: String,
        permissions: Vec<Permission>,
        duration: Duration,
    ) -> Self {
        let now = Utc::now();
        Self {
            user_id,
            wallet_addr,
            session_token: None,
            permissions,
            exp: (now + duration).timestamp(),
            iat: now.timestamp(),
            nbf: now.timestamp(),
        }
    }

    pub fn has_permission(&self, required: &Permission) -> bool {
        self.permissions
            .iter()
            .any(|p| p == required || *p == Permission::Admin)
    }
}

pub struct TokenManager {
    secret_key: Vec<u8>,
    token_duration: Duration,
}

impl TokenManager {
    pub fn new(secret_key: String) -> Self {
        Self {
            secret_key: secret_key.into_bytes(),
            token_duration: Duration::hours(1),
        }
    }

    pub fn with_duration(mut self, duration: Duration) -> Self {
        self.token_duration = duration;
        self
    }

    pub fn generate_token(&self, claims: &Claims) -> Result<String, String> {
        let header = BASE64.encode(
            serde_json::to_vec(&serde_json::json!({
                "alg": "HS256",
                "typ": "JWT"
            }))
            .map_err(|e| format!("failed to encode header: {}", e))?,
        );

        let payload = BASE64.encode(
            serde_json::to_vec(claims).map_err(|e| format!("failed to encode payload: {}", e))?,
        );

        let message = format!("{}.{}", header, payload);
        let signature = self.sign(&message);

        Ok(format!("{}.{}", message, signature))
    }

    pub fn validate_token(&self, token: &str) -> Result<Claims, String> {
        let parts: Vec<&str> = token.split('.').collect();
        if parts.len() != 3 {
            return Err("invalid token format".to_string());
        }

        let message = format!("{}.{}", parts[0], parts[1]);
        let expected_sig = self.sign(&message);

        if expected_sig != parts[2] {
            return Err("invalid signature".to_string());
        }

        let payload_bytes = BASE64
            .decode(parts[1])
            .map_err(|e| format!("failed to decode payload: {}", e))?;

        let claims: Claims = serde_json::from_slice(&payload_bytes)
            .map_err(|e| format!("failed to parse claims: {}", e))?;

        let now = Utc::now().timestamp();
        if claims.exp < now {
            return Err("token expired".to_string());
        }

        if claims.nbf > now {
            return Err("token not yet valid".to_string());
        }

        Ok(claims)
    }

    pub fn generate_hybrid_token(
        &self,
        user_id: &str,
        wallet_addr: &str,
        session_token: &str,
        permissions: Vec<Permission>,
    ) -> Result<String, String> {
        let mut claims = Claims::new(
            user_id.to_string(),
            wallet_addr.to_string(),
            permissions,
            self.token_duration,
        );
        claims.session_token = Some(session_token.to_string());

        self.generate_token(&claims)
    }

    pub fn validate_session_token(
        &self,
        token: &str,
        expected_session: &str,
    ) -> Result<bool, String> {
        let claims = self.validate_token(token)?;

        match &claims.session_token {
            Some(session) => Ok(session == expected_session),
            None => Ok(false),
        }
    }

    pub fn refresh_token(&self, token: &str) -> Result<String, String> {
        let claims = self.validate_token(token)?;

        let new_claims = Claims::new(
            claims.user_id,
            claims.wallet_addr,
            claims.permissions,
            self.token_duration,
        );

        self.generate_token(&new_claims)
    }

    fn sign(&self, message: &str) -> String {
        let mut mac =
            HmacSha256::new_from_slice(&self.secret_key).expect("HMAC can take key of any size");
        mac.update(message.as_bytes());

        let result = mac.finalize();
        BASE64.encode(result.into_bytes())
    }
}

pub struct AuthMiddleware {
    token_manager: TokenManager,
}

impl AuthMiddleware {
    pub fn new(token_manager: TokenManager) -> Self {
        Self { token_manager }
    }

    pub fn authenticate(&self, auth_header: &str) -> Result<Claims, String> {
        if !auth_header.starts_with("Bearer ") {
            return Err("invalid authorization format".to_string());
        }

        let token = &auth_header[7..];
        self.token_manager.validate_token(token)
    }

    pub fn extract_token(&self, auth_header: &str) -> Option<String> {
        if auth_header.starts_with("Bearer ") {
            Some(auth_header[7..].to_string())
        } else {
            None
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_token_generation_and_validation() {
        let manager = TokenManager::new("secret_key_for_testing".to_string());

        let claims = Claims::new(
            "user123".to_string(),
            "0x1234567890abcdef".to_string(),
            vec![Permission::ReadOnly, Permission::ReadWrite],
            Duration::hours(1),
        );

        let token = manager.generate_token(&claims).unwrap();
        assert!(!token.is_empty());

        let validated = manager.validate_token(&token).unwrap();
        assert_eq!(validated.user_id, "user123");
        assert_eq!(validated.wallet_addr, "0x1234567890abcdef");
    }

    #[test]
    fn test_permission_check() {
        let claims = Claims::new(
            "user".to_string(),
            "addr".to_string(),
            vec![Permission::ReadOnly],
            Duration::hours(1),
        );

        assert!(claims.has_permission(&Permission::ReadOnly));
        assert!(!claims.has_permission(&Permission::ReadWrite));
        assert!(!claims.has_permission(&Permission::Admin));

        let admin_claims = Claims::new(
            "admin".to_string(),
            "addr".to_string(),
            vec![Permission::Admin],
            Duration::hours(1),
        );

        assert!(admin_claims.has_permission(&Permission::ReadOnly));
        assert!(admin_claims.has_permission(&Permission::ReadWrite));
        assert!(admin_claims.has_permission(&Permission::Admin));
    }

    #[test]
    fn test_hybrid_token() {
        let manager = TokenManager::new("secret_key_for_testing".to_string());

        let token = manager
            .generate_hybrid_token(
                "user123",
                "0x1234567890abcdef",
                "session_abc123",
                vec![Permission::ReadOnly],
            )
            .unwrap();

        let validated = manager.validate_token(&token).unwrap();
        assert_eq!(validated.session_token, Some("session_abc123".to_string()));

        let session_valid = manager
            .validate_session_token(&token, "session_abc123")
            .unwrap();
        assert!(session_valid);

        let session_invalid = manager
            .validate_session_token(&token, "wrong_session")
            .unwrap();
        assert!(!session_invalid);
    }

    #[test]
    fn test_token_refresh() {
        let manager = TokenManager::new("secret_key_for_testing".to_string());

        let claims = Claims::new(
            "user123".to_string(),
            "0x1234567890abcdef".to_string(),
            vec![Permission::ReadOnly],
            Duration::hours(1),
        );

        let old_token = manager.generate_token(&claims).unwrap();

        std::thread::sleep(std::time::Duration::from_millis(1100));

        let new_token = manager.refresh_token(&old_token).unwrap();

        assert_ne!(old_token, new_token);

        let validated = manager.validate_token(&new_token).unwrap();
        assert_eq!(validated.user_id, "user123");
    }
}
