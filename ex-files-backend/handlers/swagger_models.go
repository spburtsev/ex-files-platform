package handlers

import "time"

// NOTE: These types exist solely for Swagger documentation.
// The actual API responses use protobuf binary encoding (application/x-protobuf).
// The JSON shapes below represent the protojson equivalent of each proto message.

// --- Auth ---

type swagUser struct {
	ID        uint64    `json:"id" example:"1"`
	Email     string    `json:"email" example:"user@example.com"`
	Name      string    `json:"name" example:"John Doe"`
	AvatarURL string    `json:"avatarUrl"`
	Role      string    `json:"role" example:"ROLE_EMPLOYEE" enums:"ROLE_UNSPECIFIED,ROLE_ROOT,ROLE_MANAGER,ROLE_EMPLOYEE"`
	CreatedAt time.Time `json:"createdAt"`
}

type swagRegisterRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"secret123"`
	Name     string `json:"name" example:"John Doe"`
}

type swagLoginRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"secret123"`
}

type swagAuthResponse struct {
	User  swagUser `json:"user"`
	Token string   `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
}

type swagMeResponse struct {
	User swagUser `json:"user"`
}

type swagGetUsersResponse struct {
	Users []swagUser `json:"users"`
}

// --- Workspaces ---

type swagWorkspace struct {
	ID        uint64    `json:"id" example:"1"`
	Name      string    `json:"name" example:"Engineering"`
	ManagerID uint64    `json:"managerId" example:"1"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type swagWorkspaceDetail struct {
	Workspace swagWorkspace `json:"workspace"`
	Manager   swagUser      `json:"manager"`
	Members   []swagUser    `json:"members"`
}

type swagWorkspaceMember struct {
	ID          uint64    `json:"id" example:"1"`
	WorkspaceID uint64    `json:"workspaceId" example:"1"`
	UserID      uint64    `json:"userId" example:"2"`
	CreatedAt   time.Time `json:"createdAt"`
}

type swagCreateWorkspaceRequest struct {
	Name string `json:"name" example:"Engineering"`
}

type swagCreateWorkspaceResponse struct {
	Workspace swagWorkspace `json:"workspace"`
}

type swagGetWorkspacesResponse struct {
	Workspaces []swagWorkspace `json:"workspaces"`
}

type swagGetWorkspaceResponse struct {
	Workspace swagWorkspaceDetail `json:"workspace"`
}

type swagUpdateWorkspaceRequest struct {
	Name string `json:"name" example:"Engineering v2"`
}

type swagUpdateWorkspaceResponse struct {
	Workspace swagWorkspace `json:"workspace"`
}

type swagAddMemberRequest struct {
	UserID uint64 `json:"userId" example:"2"`
}

type swagAddMemberResponse struct {
	Member swagWorkspaceMember `json:"member"`
}

type swagAssignableMembersResponse struct {
	Users []swagUser `json:"users"`
}

// --- Issues ---

type swagIssue struct {
	ID            string     `json:"id" example:"1"`
	WorkspaceID   string     `json:"workspaceId" example:"1"`
	CreatorID     string     `json:"creatorId" example:"1"`
	AssigneeID    string     `json:"assigneeId" example:"2"`
	Title         string     `json:"title" example:"Review Q4 report"`
	Description   string     `json:"description"`
	Deadline      *time.Time `json:"deadline"`
	Resolved      bool       `json:"resolved"`
	CommentsCount int32      `json:"commentsCount"`
	VersionsCount int32      `json:"versionsCount"`
}

type swagCreateIssueRequest struct {
	Title       string `json:"title" example:"Review Q4 report"`
	Description string `json:"description"`
	AssigneeID  uint   `json:"assignee_id" example:"2"`
	Deadline    string `json:"deadline" example:"2026-12-31T23:59:59Z"`
}

type swagGetIssuesResponse struct {
	Issues []swagIssue `json:"issues"`
}

type swagGetIssueResponse struct {
	Issue swagIssue `json:"issue"`
	User  swagUser  `json:"user"`
}

type swagCreateIssueResponse struct {
	Issue swagIssue `json:"issue"`
}

// --- Documents ---

type swagDocument struct {
	ID           uint64    `json:"id" example:"1"`
	Name         string    `json:"name" example:"report.pdf"`
	MimeType     string    `json:"mimeType" example:"application/pdf"`
	Size         int64     `json:"size" example:"102400"`
	Hash         string    `json:"hash" example:"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"`
	Status       string    `json:"status" example:"pending" enums:"pending,in_review,approved,rejected,changes_requested"`
	UploaderID   uint64    `json:"uploaderId" example:"1"`
	UploaderName string    `json:"uploaderName" example:"John Doe"`
	IssueID      uint64    `json:"issueId" example:"1"`
	ReviewerID   uint64    `json:"reviewerId"`
	ReviewerName string    `json:"reviewerName"`
	ReviewerNote string    `json:"reviewerNote"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type swagDocumentVersion struct {
	ID           uint64    `json:"id" example:"1"`
	DocumentID   uint64    `json:"documentId" example:"1"`
	Version      int32     `json:"version" example:"1"`
	Hash         string    `json:"hash"`
	Size         int64     `json:"size" example:"102400"`
	StorageKey   string    `json:"storageKey"`
	UploaderID   uint64    `json:"uploaderId" example:"1"`
	UploaderName string    `json:"uploaderName" example:"John Doe"`
	CreatedAt    time.Time `json:"createdAt"`
}

type swagDocumentDetail struct {
	Document swagDocument          `json:"document"`
	Versions []swagDocumentVersion `json:"versions"`
}

type swagUploadDocumentResponse struct {
	Document swagDocument        `json:"document"`
	Version  swagDocumentVersion `json:"version"`
}

type swagGetDocumentResponse struct {
	Document swagDocumentDetail `json:"document"`
}

type swagListDocumentsResponse struct {
	Documents []swagDocument `json:"documents"`
}

type swagUpdateDocumentResponse struct {
	Document swagDocument `json:"document"`
}

type swagDownloadURLResponse struct {
	URL string `json:"url" example:"https://minio.example.com/documents/..."`
}

type swagReviewNoteRequest struct {
	Note string `json:"note" example:"Please fix section 3"`
}

type swagAssignReviewerRequest struct {
	ReviewerID uint `json:"reviewer_id" example:"2"`
}

// --- Comments ---

type swagComment struct {
	ID         uint64    `json:"id" example:"1"`
	DocumentID uint64    `json:"documentId" example:"1"`
	AuthorID   uint64    `json:"authorId" example:"1"`
	AuthorName string    `json:"authorName" example:"John Doe"`
	Body       string    `json:"body" example:"Looks good to me"`
	CreatedAt  time.Time `json:"createdAt"`
}

type swagCreateCommentRequest struct {
	Body string `json:"body" example:"Looks good to me"`
}

type swagCreateCommentResponse struct {
	Comment swagComment `json:"comment"`
}

type swagListCommentsResponse struct {
	Comments []swagComment `json:"comments"`
}

// --- Audit ---

type swagAuditEntry struct {
	ID         uint64    `json:"id" example:"1"`
	Action     string    `json:"action" example:"document.uploaded"`
	ActorID    uint64    `json:"actorId" example:"1"`
	ActorName  string    `json:"actorName" example:"John Doe"`
	TargetID   *uint64   `json:"targetId" example:"1"`
	TargetType string    `json:"targetType" example:"document"`
	Metadata   any       `json:"metadata"`
	CreatedAt  time.Time `json:"createdAt"`
}

type swagGetAuditLogResponse struct {
	Entries []swagAuditEntry `json:"entries"`
}

// --- Common ---

type swagErrorResponse struct {
	Error string `json:"error" example:"invalid credentials"`
}

type swagMessageResponse struct {
	Message string `json:"message" example:"ok"`
}

type swagVerifyResponse struct {
	Verified     bool      `json:"verified" example:"true"`
	DocumentName string    `json:"document_name,omitempty" example:"report.pdf"`
	Status       string    `json:"status,omitempty" example:"approved"`
	NotarizedAt  time.Time `json:"notarized_at,omitempty"`
	Hash         string    `json:"hash,omitempty"`
}
