package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hackathon/authsvc/internal/domain"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Repository implements all PostgreSQL-backed ports.
type Repository struct {
	db *sql.DB
}

// Open opens a PostgreSQL connection pool.
func Open(ctx context.Context, databaseURL string) (*Repository, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Repository{db: db}, nil
}

// Close closes the database pool.
func (r *Repository) Close() error {
	return r.db.Close()
}

// DB exposes the underlying pool for health checks.
func (r *Repository) DB() *sql.DB {
	return r.db
}

// CreateUser inserts a user.
func (r *Repository) CreateUser(ctx context.Context, user domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		insert into users (id, tenant_id, email, password_hash, is_active, created_at, updated_at)
		values ($1, $2, $3, $4, $5, $6, $7)
	`, user.ID, user.TenantID, user.Email, user.PasswordHash, user.IsActive, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetUserByEmail returns a user by tenant and email.
func (r *Repository) GetUserByEmail(ctx context.Context, tenantID string, email string) (domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, `
		select id, tenant_id, email, password_hash, is_active, created_at, updated_at
		from users where tenant_id = $1 and email = $2
	`, tenantID, email))
}

// GetUserByID returns a user by tenant and id.
func (r *Repository) GetUserByID(ctx context.Context, tenantID string, userID string) (domain.User, error) {
	return r.scanUser(r.db.QueryRowContext(ctx, `
		select id, tenant_id, email, password_hash, is_active, created_at, updated_at
		from users where tenant_id = $1 and id = $2
	`, tenantID, userID))
}

// UpdatePassword updates a user's password hash.
func (r *Repository) UpdatePassword(ctx context.Context, tenantID string, userID string, passwordHash string, now time.Time) error {
	result, err := r.db.ExecContext(ctx, `
		update users set password_hash = $3, updated_at = $4
		where tenant_id = $1 and id = $2
	`, tenantID, userID, passwordHash, now)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("password rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) scanUser(row *sql.Row) (domain.User, error) {
	var user domain.User
	err := row.Scan(&user.ID, &user.TenantID, &user.Email, &user.PasswordHash, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("scan user: %w", err)
	}
	return user, nil
}

// CreateRefreshToken inserts a hashed refresh token.
func (r *Repository) CreateRefreshToken(ctx context.Context, token domain.RefreshToken) error {
	_, err := r.db.ExecContext(ctx, `
		insert into refresh_tokens (id, tenant_id, user_id, token_hash, family_id, expires_at, created_at)
		values ($1, $2, $3, $4, $5, $6, $7)
	`, token.ID, token.TenantID, token.UserID, token.Hash, token.FamilyID, token.ExpiresAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

// GetRefreshTokenByHash returns a refresh token by hash.
func (r *Repository) GetRefreshTokenByHash(ctx context.Context, hash string) (domain.RefreshToken, error) {
	var token domain.RefreshToken
	err := r.db.QueryRowContext(ctx, `
		select id, tenant_id, user_id, token_hash, family_id, expires_at, created_at, revoked_at, revoked_reason, coalesce(replaced_by_id, '')
		from refresh_tokens where token_hash = $1
	`, hash).Scan(&token.ID, &token.TenantID, &token.UserID, &token.Hash, &token.FamilyID, &token.ExpiresAt, &token.CreatedAt, &token.RevokedAt, &token.RevokedReason, &token.ReplacedByID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.RefreshToken{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("scan refresh token: %w", err)
	}
	return token, nil
}

// RevokeRefreshToken revokes one refresh token.
func (r *Repository) RevokeRefreshToken(ctx context.Context, id string, reason string, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		update refresh_tokens set revoked_at = coalesce(revoked_at, $2), revoked_reason = $3 where id = $1
	`, id, now, reason)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RevokeRefreshTokenFamily revokes all tokens in a refresh-token family.
func (r *Repository) RevokeRefreshTokenFamily(ctx context.Context, familyID string, reason string, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		update refresh_tokens set revoked_at = coalesce(revoked_at, $2), revoked_reason = $3 where family_id = $1
	`, familyID, now, reason)
	if err != nil {
		return fmt.Errorf("revoke refresh token family: %w", err)
	}
	return nil
}

// CreatePasswordResetToken inserts a reset token.
func (r *Repository) CreatePasswordResetToken(ctx context.Context, token domain.PasswordResetToken) error {
	_, err := r.db.ExecContext(ctx, `
		insert into password_reset_tokens (id, tenant_id, user_id, token_hash, expires_at, created_at)
		values ($1, $2, $3, $4, $5, $6)
	`, token.ID, token.TenantID, token.UserID, token.Hash, token.ExpiresAt, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert password reset token: %w", err)
	}
	return nil
}

// GetPasswordResetTokenByHash returns a reset token by hash.
func (r *Repository) GetPasswordResetTokenByHash(ctx context.Context, hash string) (domain.PasswordResetToken, error) {
	var token domain.PasswordResetToken
	err := r.db.QueryRowContext(ctx, `
		select id, tenant_id, user_id, token_hash, expires_at, created_at, consumed_at
		from password_reset_tokens where token_hash = $1
	`, hash).Scan(&token.ID, &token.TenantID, &token.UserID, &token.Hash, &token.ExpiresAt, &token.CreatedAt, &token.ConsumedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PasswordResetToken{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.PasswordResetToken{}, fmt.Errorf("scan password reset token: %w", err)
	}
	return token, nil
}

// ConsumePasswordResetToken marks a reset token as consumed.
func (r *Repository) ConsumePasswordResetToken(ctx context.Context, id string, now time.Time) error {
	_, err := r.db.ExecContext(ctx, `update password_reset_tokens set consumed_at = $2 where id = $1 and consumed_at is null`, id, now)
	if err != nil {
		return fmt.Errorf("consume password reset token: %w", err)
	}
	return nil
}

// GetUserRoles returns roles with permissions for a user.
func (r *Repository) GetUserRoles(ctx context.Context, tenantID string, userID string) ([]domain.Role, error) {
	rows, err := r.db.QueryContext(ctx, `
		select r.id, r.tenant_id, r.name, coalesce(p.id, ''), coalesce(p.value, '')
		from user_roles ur
		join roles r on r.id = ur.role_id and r.tenant_id = ur.tenant_id
		left join role_permissions rp on rp.role_id = r.id and rp.tenant_id = r.tenant_id
		left join permissions p on p.id = rp.permission_id and p.tenant_id = rp.tenant_id
		where ur.tenant_id = $1 and ur.user_id = $2
		order by r.name
	`, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()
	byID := map[string]*domain.Role{}
	for rows.Next() {
		var roleID, roleTenant, roleName, permissionID, permissionValue string
		if err := rows.Scan(&roleID, &roleTenant, &roleName, &permissionID, &permissionValue); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		role := byID[roleID]
		if role == nil {
			role = &domain.Role{ID: roleID, TenantID: roleTenant, Name: roleName}
			byID[roleID] = role
		}
		if permissionID != "" {
			role.Permissions = append(role.Permissions, domain.Permission{ID: permissionID, TenantID: tenantID, Value: permissionValue})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles: %w", err)
	}
	roles := make([]domain.Role, 0, len(byID))
	for _, role := range byID {
		roles = append(roles, *role)
	}
	return roles, nil
}

// UserHasPermission checks whether a user has a permission in a tenant.
func (r *Repository) UserHasPermission(ctx context.Context, tenantID string, userID string, permission string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		select exists (
			select 1
			from user_roles ur
			join role_permissions rp on rp.role_id = ur.role_id and rp.tenant_id = ur.tenant_id
			join permissions p on p.id = rp.permission_id and p.tenant_id = rp.tenant_id
			where ur.tenant_id = $1 and ur.user_id = $2 and p.value = $3
		)
	`, tenantID, userID, permission).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check permission: %w", err)
	}
	return exists, nil
}

// Log writes an audit event into PostgreSQL.
func (r *Repository) Log(ctx context.Context, event domain.AuditEvent) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		insert into audit_logs (tenant_id, user_id, action, outcome, ip, user_agent, metadata, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.TenantID, event.UserID, event.Action, event.Outcome, event.IP, event.UserAgent, metadata, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "SQLSTATE 23505") || strings.Contains(err.Error(), "duplicate key"))
}
