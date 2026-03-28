package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrEmployeeInactive = errors.New("员工已停用")

type AppService struct {
	repo *Repository
	jwt  *JWTManager
}

func NewAppService(repo *Repository, jwt *JWTManager) *AppService {
	return &AppService{repo: repo, jwt: jwt}
}

func (s *AppService) Login(ctx context.Context, input AppLoginInput) (*AppLoginResponse, error) {
	emp, err := s.repo.FindEmployeeByLoginID(ctx, strings.TrimSpace(input.LoginID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		if errors.Is(err, ErrAppLoginIDAmbiguous) {
			return nil, err
		}
		return nil, fmt.Errorf("查询员工失败: %w", err)
	}
	if emp.Status != "active" {
		return nil, ErrEmployeeInactive
	}
	if emp.AppPasswordHash == nil || strings.TrimSpace(*emp.AppPasswordHash) == "" {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*emp.AppPasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.buildLoginResponse(emp)
}

func (s *AppService) RefreshToken(ctx context.Context, refreshToken string) (*AppLoginResponse, error) {
	claims, err := s.jwt.ParseToken(refreshToken)
	if err != nil {
		return nil, err
	}
	emp, err := s.repo.GetEmployeeByIDForAppAuth(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("查询员工失败: %w", err)
	}
	if emp.Status != "active" {
		return nil, ErrEmployeeInactive
	}
	return s.buildLoginResponse(emp)
}

func (s *AppService) ResetPassword(ctx context.Context, employeeID string, input ResetPasswordInput) error {
	if !isStrongPassword(input.NewPassword) {
		return ErrWeakPassword
	}
	emp, err := s.repo.GetEmployeeByIDForAppAuth(ctx, employeeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if emp.AppPasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*emp.AppPasswordHash), []byte(input.OldPassword)) != nil {
		return ErrOldPasswordMismatch
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}
	return s.repo.UpdateEmployeeAppCredentials(ctx, employeeID, string(hashed), false)
}

func (s *AppService) ForceResetPassword(ctx context.Context, employeeID string, input ForceResetPasswordInput) error {
	if !isStrongPassword(input.NewPassword) {
		return ErrWeakPassword
	}
	emp, err := s.repo.GetEmployeeByIDForAppAuth(ctx, employeeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if !emp.AppMustResetPwd {
		return ErrNotRequireReset
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码哈希失败: %w", err)
	}
	return s.repo.UpdateEmployeeAppCredentials(ctx, employeeID, string(hashed), false)
}

func (s *AppService) GetMe(ctx context.Context, claims *Claims) (*AppMeResponse, error) {
	emp, err := s.repo.GetEmployeeByIDForAppAuth(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if emp.Status != "active" {
		return nil, ErrEmployeeInactive
	}
	return &AppMeResponse{
		Employee:    emp.toInfo(),
		CurrentNode: emp.currentNode(),
	}, nil
}

func (s *AppService) buildLoginResponse(emp *appEmployeeRecord) (*AppLoginResponse, error) {
	accessToken, err := s.jwt.GenerateAccessToken(emp.ID, emp.OrgNodeID, emp.OrgNodePath, appRoleName)
	if err != nil {
		return nil, fmt.Errorf("签发 Access Token 失败: %w", err)
	}
	refreshToken, err := s.jwt.GenerateRefreshToken(emp.ID, emp.OrgNodeID, emp.OrgNodePath, appRoleName)
	if err != nil {
		return nil, fmt.Errorf("签发 Refresh Token 失败: %w", err)
	}
	return &AppLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.jwt.AccessTokenTTLSeconds(),
		Employee:     emp.toInfo(),
		CurrentNode:  emp.currentNode(),
		MustResetPwd: emp.AppMustResetPwd,
	}, nil
}
