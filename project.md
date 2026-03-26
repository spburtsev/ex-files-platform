# Multi-Tier application proposal: Document Notarization & Approval Pipeline

by Sergei Burtsev 71336

## **Problem**

Organizations handling contracts, invoices, and compliance documents need a reliable way to prove that a document existed at a specific point in time, was reviewed by authorized staff, and was never silently altered.

**Proposed Solution**

Notarization platform where documents are uploaded, routed through a configurable approval workflow, versioned, and auditable. As an optional extension, approved documents can be connected to a public blockchain to provide third-party-verifiable records without relying on the application server.

## **Features**

**1 Document Upload & Management**

Users can upload PDF documents in common formats through a drag-and-drop interface or file picker. Each upload is hashed with SHA-256 to produce a unique fingerprint. The system stores the original file in S3-compatible object storage and records metadata such as file name, size, uploader, and timestamp in a relational DB. Uploads are validated for size limits and files type.

Documents are organized into workspaces. Within each workspace, managers can create issues — discrete review tasks or requests that group related documents together. Documents are uploaded to a specific issue, not directly to the workspace. Each issue has a title, description, and tracks its own lifecycle. Users can search and filter documents by name, date, status and uploader within an issue. A document detail view shows the current version, full version history, approval status and comments.

**2 Role-Based Access Control**

The platform enforces two primary roles: Manager and Employee. Managers can create workspaces, create issues within workspaces, manage workspace members, assign reviewers to documents, and perform approve/reject/request-changes actions. Employees can view workspaces they belong to, upload documents to issues, and view their own submissions. Both roles can view documents and comment on them. Managers have additional access to the global audit log and user management.

Permissions are checked at the API layer on every request. Each endpoint validates the caller’s JWT token, extracts their role, and enforces access rules before processing. Role assignments are stored in a DB and can be updated by administrators through a management UI.

**3 Issues**

Within a workspace, managers create issues to organize review work. An issue represents a specific task, request, or review topic (e.g., "Q2 Contract Review", "Invoice Batch #42"). Each issue has a title, description, status, and a deadline. Employees upload documents to an issue, and the review/approval workflow happens at the document level within that issue. Issues provide context and grouping so that related documents are reviewed together.

**4 Real-Time Commenting, Versioning and Status changes**

Every document has a comment section where reviewers and submitters can post messages. Comments are attached to a specific document version.

Approval events, new comments, and status transitions are displayed in real time.

**5 Document Versioning**

Every time a submitter uploads a revised document, the system creates a new version linked to the same document record. The previous version remains immutable in object storage, preserving a complete history. Each version stores its own SHA-256 hash, upload timestamp, uploader identity, and the approval state at the time of creation.

Users can browse the full version history in a timeline view, download any past version, and compare metadata between versions.

**6 Audit Log**

The platform maintains an audit log that records every significant action: document uploads, version creations, approval decisions (with reviewer ID and timestamp), rejections with reasons, comments and role changes. Each entry includes the acting user and the target resource.

Administrators can view the full audit log through a dedicated UI with filtering by date range, event type, user, or document. The log is append-only by design—entries cannot be modified or deleted, even by administrators ensuring a trustworthy record for compliance and dispute resolution.

**7 Public Verification Endpoint**

A publicly accessible REST endpoint allows anyone to verify a document’s integrity without logging in. The verifier uploads or provides the SHA-256 hash of a document, and the system responds with whether a matching hash exists and the timestamp of its notarization. This enables third parties (auditors, regulators, counterparties) to independently confirm that a document was notarized and approved without needing an account on the platform.

**8 Email Notifications**

The platform sends email notifications for key events to keep all participants informed without requiring them to be actively logged in. Reviewers receive an email when a new document is assigned to them for approval, including a direct link to the review page. Submitters are notified when their document is approved, rejected, or when a reviewer requests changes, with the rejection or change-request reason included in the email body.

## **9 Optional Extension: Blockchain Audit Trail**

As an optional extension, the platform can record document fingerprints and approval events on a public blockchain testnet. When a document is approved, its SHA-256 hash and the list of approver identities are submitted to a smart contract that permanently records the notarization on-chain.

Each approver’s signature is cryptographically tied to their unique wallet address, making it impossible to forge or deny approval after the fact.

## **Tech Stack**


| Tier                      | Technology     | Responsibilities                                                                                   |
| ------------------------- | -------------- | -------------------------------------------------------------------------------------------------- |
| **Web Frontend**          | Svelte         | Document upload & preview, approval workflow UI, real-time event feed, audit log viewer            |
| **Backend**               | Go             | File workflows, SHA-256 hashing, approval state machine, events streaming.                         |
| **File Storage**          | S3 (MinIO)     | Raw document BLOBs, versioned artifact storage, pre-signed URL serving                             |
| **Database**              | PostgreSQL     | Users, workspaces, issues, documents metadata, workflow state, approval history, audit metadata    |
| **Blockchain (optional)** | Public testnet | Document hash registry, on-chain approval records, smart contract enforcing N-of-M approval policy |


