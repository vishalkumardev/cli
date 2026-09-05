package api

// ── Auth types ────────────────────────────────────────────────────────────────

type LoginRequest struct {
	Email string `json:"email"`
}

type VerifyOTPRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
	Name  string `json:"name,omitempty"`
}

type VerifyAPIKeyRequest struct {
	APIKey string `json:"apiKey"`
}

type AuthResult struct {
	Token    string `json:"token"`
	UserID   int    `json:"userId"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

type APIKeyVerifyResult struct {
	AccessToken string     `json:"accessToken"`
	User        AuthResult `json:"user"`
}

// ── User types ────────────────────────────────────────────────────────────────

type UserProfile struct {
	UserID   int    `json:"userId"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	IsAdmin  bool   `json:"isAdmin"`
}

// ── App types ─────────────────────────────────────────────────────────────────

type CreateAppRequest struct {
	Name        string `json:"name"`
	PackageName string `json:"packageName"`
}

type App struct {
	AppID          string `json:"appId"`
	Name           string `json:"name"`
	PackageName    string `json:"packageName"`
	OrganizationID string `json:"organizationId"`
	CreatedAt      string `json:"createdAt"`
}

type AppListResponse struct {
	Records    []App `json:"records"`
	Data       []App `json:"data"`
	Total      int   `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalPages int   `json:"totalPages"`
}

func (r AppListResponse) Items() []App {
	if len(r.Records) > 0 {
		return r.Records
	}
	return r.Data
}

type PaginationRequest struct {
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
	Search   string `json:"search,omitempty"`
}

// ── Build types ───────────────────────────────────────────────────────────────

type Build struct {
	BuildID       string  `json:"buildId"`
	AppID         string  `json:"appId"`
	AppName       string  `json:"appName"`
	PackageName   string  `json:"packageName"`
	VersionName   string  `json:"versionName"`
	VersionCode   int     `json:"versionCode"`
	Platform      string  `json:"platform"`
	FileSize      float64 `json:"fileSize"`
	Changelog     string  `json:"changelog"`
	DownloadCount int     `json:"downloadCount"`
	InstallURL    string  `json:"installUrl,omitempty"`
	CreatedAt     string  `json:"createdAt"`
}

type BuildListResponse struct {
	Records    []Build `json:"records"`
	Data       []Build `json:"data"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"pageSize"`
	TotalPages int     `json:"totalPages"`
}

func (r BuildListResponse) Items() []Build {
	if len(r.Records) > 0 {
		return r.Records
	}
	return r.Data
}

type DownloadResponse struct {
	DownloadURL string `json:"downloadUrl"`
}

// ── Organization types ────────────────────────────────────────────────────────

type Organization struct {
	OrganizationID string `json:"organizationId"`
	Name           string `json:"name"`
	OwnerID        int    `json:"ownerId"`
	CreatedAt      string `json:"createdAt"`
}
