package handlers

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/spburtsev/ex-files-backend/gen/auth/v1"
	"github.com/spburtsev/ex-files-backend/models"
)

func roleToProto(r models.Role) authv1.Role {
	switch r {
	case models.RoleRoot:
		return authv1.Role_ROLE_ROOT
	case models.RoleManager:
		return authv1.Role_ROLE_MANAGER
	case models.RoleEmployee:
		return authv1.Role_ROLE_EMPLOYEE
	default:
		return authv1.Role_ROLE_EMPLOYEE
	}
}

func userToProto(u *models.User) *authv1.User {
	return &authv1.User{
		Id:        uint64(u.ID),
		Email:     u.Email,
		Name:      u.Name,
		AvatarUrl: u.AvatarURL,
		Role:      roleToProto(u.Role),
		CreatedAt: timestamppb.New(u.CreatedAt),
	}
}
