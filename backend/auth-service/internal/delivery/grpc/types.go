package grpc

// RegisterRequest registers a tenant-scoped user.
type RegisterRequest struct {
	TenantId string `json:"tenant_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Ip       string `json:"ip"`
}

type LoginRequest struct {
	TenantId string `json:"tenant_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Ip       string `json:"ip"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	TenantId string `json:"tenant_id"`
	Email    string `json:"email"`
	Ip       string `json:"ip"`
}

type ResetPasswordRequest struct {
	TenantId    string `json:"tenant_id"`
	ResetToken  string `json:"reset_token"`
	NewPassword string `json:"new_password"`
	Ip          string `json:"ip"`
}

type ValidateTokenRequest struct {
	AccessToken string `json:"access_token"`
}

type GetUserInfoRequest struct {
	AccessToken string `json:"access_token"`
}

type AuthorizeRequest struct {
	AccessToken string `json:"access_token"`
	Permission  string `json:"permission"`
}

type AuthResponse struct {
	User         *User  `json:"user,omitempty"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
}

type EmptyResponse struct{}

type ValidateTokenResponse struct {
	Valid       bool     `json:"valid"`
	Subject     string   `json:"subject"`
	TenantId    string   `json:"tenant_id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	ExpiresAt   string   `json:"expires_at"`
	TokenId     string   `json:"token_id"`
}

type GetUserInfoResponse struct {
	User *User `json:"user,omitempty"`
}

type AuthorizeResponse struct {
	Allowed bool                   `json:"allowed"`
	Claims  *ValidateTokenResponse `json:"claims,omitempty"`
}

type User struct {
	Id        string `json:"id"`
	TenantId  string `json:"tenant_id"`
	Email     string `json:"email"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
