package artifacts

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/wyw14/cry-082/internal/domain/artifact"
	"github.com/wyw14/cry-082/internal/domain/audit"
	"github.com/wyw14/cry-082/internal/domain/site"
	"github.com/wyw14/cry-082/internal/platform/transaction"
)

var ErrFileSiteMismatch = errors.New("file does not belong to site")

type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string }
type Repository interface {
	SaveFile(context.Context, artifact.File) error
	FindFile(context.Context, string) (artifact.File, error)
}
type AccessRepository interface {
	Membership(context.Context, string, string) (site.Membership, error)
}
type AuditRepository interface {
	AppendAudit(context.Context, audit.Entry) error
}
type FileStore interface {
	Put(context.Context, string, string, []byte) (string, string, error)
	Open(context.Context, string) (io.ReadCloser, error)
}

type Service struct {
	repository Repository
	access     AccessRepository
	audits     AuditRepository
	files      FileStore
	tx         transaction.Manager
	clock      Clock
	ids        IDGenerator
}

func New(repository Repository, access AccessRepository, audits AuditRepository, files FileStore, tx transaction.Manager, clock Clock, ids IDGenerator) *Service {
	return &Service{repository: repository, access: access, audits: audits, files: files, tx: tx, clock: clock, ids: ids}
}

type UploadInput struct {
	SiteID, ActorID, RequestID string
	Name, MediaType            string
	Purpose                    artifact.Purpose
	Payload                    []byte
}

func (s *Service) Upload(ctx context.Context, input UploadInput) (artifact.File, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	membership, err := s.access.Membership(ctx, input.ActorID, input.SiteID)
	if err != nil {
		return artifact.File{}, err
	}
	if err := site.Require(membership, input.SiteID, site.PermissionMaintenance); err != nil {
		return artifact.File{}, err
	}
	fileID, checksum, err := s.files.Put(ctx, input.Name, input.MediaType, input.Payload)
	if err != nil {
		return artifact.File{}, err
	}
	stored, err := artifact.NewFile(fileID, input.SiteID, input.Name, input.MediaType, input.Purpose, checksum, int64(len(input.Payload)), input.ActorID, s.clock.Now())
	if err != nil {
		return artifact.File{}, err
	}
	err = transaction.Execute(ctx, s.tx, func(txctx context.Context) error {
		if err := s.repository.SaveFile(txctx, stored); err != nil {
			return err
		}
		entry, err := audit.New(s.ids.NewID(), input.SiteID, input.ActorID, "api", "file.uploaded", "file", stored.ID, "upload controlled artifact", input.RequestID, nil, map[string]string{"purpose": string(stored.Purpose), "checksum": stored.Checksum}, s.clock.Now())
		if err != nil {
			return err
		}
		return s.audits.AppendAudit(txctx, entry)
	})
	if err != nil {
		return artifact.File{}, err
	}
	return stored, nil
}

func (s *Service) Download(ctx context.Context, siteID, fileID, actorID string) (artifact.File, io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	membership, err := s.access.Membership(ctx, actorID, siteID)
	if err != nil {
		return artifact.File{}, nil, err
	}
	if err := site.Require(membership, siteID, site.PermissionSiteRead); err != nil {
		return artifact.File{}, nil, err
	}
	stored, err := s.repository.FindFile(ctx, fileID)
	if err != nil {
		return artifact.File{}, nil, err
	}
	if stored.SiteID != siteID {
		return artifact.File{}, nil, ErrFileSiteMismatch
	}
	reader, err := s.files.Open(ctx, stored.ID)
	if err != nil {
		return artifact.File{}, nil, err
	}
	return stored, reader, nil
}
