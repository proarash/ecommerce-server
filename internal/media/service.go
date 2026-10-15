package media

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

const MaxImageSize = 2 << 20

var (
	ErrFileTooLarge     = errors.New("file exceeds the 2MB limit")
	ErrUnsupportedImage = errors.New("only webp and svg images are allowed")
	ErrInvalidFileName  = errors.New("invalid media file name")
	ErrSingleImage      = errors.New("a category or banner accepts exactly one image")
	ErrEmptyMeta        = errors.New("title and alt cannot be empty")
)

var imageExtensions = map[string]string{
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
}

var fileNamePattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\.(webp|svg)$`)

func detectImageType(fileName string, data []byte) string {
	if strings.EqualFold(filepath.Ext(fileName), ".svg") {
		return "image/svg+xml"
	}
	return http.DetectContentType(data)
}

type Service struct {
	store Store
	dir   string
}

func NewService(store Store, dir string) *Service {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	return &Service{store: store, dir: dir}
}

type storedFile struct {
	name string
	mime string
	size int64
}

func (s *Service) writeImage(fh *multipart.FileHeader) (storedFile, error) {
	if fh.Size > MaxImageSize {
		return storedFile{}, ErrFileTooLarge
	}
	f, err := fh.Open()
	if err != nil {
		return storedFile{}, err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, MaxImageSize+1))
	if err != nil {
		return storedFile{}, err
	}
	if len(data) > MaxImageSize {
		return storedFile{}, ErrFileTooLarge
	}
	mime := detectImageType(fh.Filename, data)
	ext, ok := imageExtensions[mime]
	if !ok {
		return storedFile{}, ErrUnsupportedImage
	}
	name := uuid.NewString() + ext
	if err := os.WriteFile(s.path(name), data, 0o644); err != nil {
		return storedFile{}, err
	}
	return storedFile{name: name, mime: mime, size: int64(len(data))}, nil
}

func (s *Service) path(name string) string {
	return filepath.Join(s.dir, name)
}

func (meta Meta) resolve(fileName string) Meta {
	meta.Title = strings.TrimSpace(meta.Title)
	meta.Alt = strings.TrimSpace(meta.Alt)
	if meta.Title == "" {
		meta.Title = strings.TrimSuffix(fileName, filepath.Ext(fileName))
	}
	if meta.Alt == "" {
		meta.Alt = meta.Title
	}
	return meta
}

func (sf storedFile) apply(m *Media) {
	m.URL = "/media/" + sf.name
	m.MediaType = TypeImage
	m.MimeType = sf.mime
	m.FileSize = sf.size
	m.FileName = sf.name
}

func (s *Service) saveImage(ctx context.Context, fh *multipart.FileHeader, meta Meta) (Media, error) {
	sf, err := s.writeImage(fh)
	if err != nil {
		return Media{}, err
	}
	meta = meta.resolve(fh.Filename)
	m := Media{Title: meta.Title, Alt: meta.Alt}
	sf.apply(&m)
	if err := s.store.Create(ctx, &m); err != nil {
		os.Remove(s.path(sf.name))
		return Media{}, err
	}
	return m, nil
}

func (s *Service) discard(ctx context.Context, items []Media) {
	for _, m := range items {
		s.store.Delete(ctx, m.ID)
		os.Remove(s.path(m.FileName))
	}
}

func (s *Service) List(ctx context.Context, q ListQuery) ([]Media, int64, error) {
	return s.store.List(ctx, q)
}

func (s *Service) Upload(ctx context.Context, target int, targetID uint, files []*multipart.FileHeader, metas []Meta) ([]Media, error) {
	if target != 0 && target != TargetProduct && len(files) != 1 {
		return nil, ErrSingleImage
	}
	if target != 0 {
		if err := s.store.TargetExists(ctx, target, targetID); err != nil {
			return nil, err
		}
	}
	saved := make([]Media, 0, len(files))
	for i, fh := range files {
		var meta Meta
		if i < len(metas) {
			meta = metas[i]
		}
		m, err := s.saveImage(ctx, fh, meta)
		if err != nil {
			s.discard(ctx, saved)
			return nil, err
		}
		saved = append(saved, m)
	}
	if target == 0 {
		return saved, nil
	}
	if err := s.store.AssignToTarget(ctx, target, targetID, saved); err != nil {
		s.discard(ctx, saved)
		return nil, err
	}
	return saved, nil
}

func (s *Service) Update(ctx context.Context, id uint, dto UpdateDto) (Media, error) {
	m, err := s.store.FindByID(ctx, id)
	if err != nil {
		return Media{}, err
	}
	if dto.Title != nil {
		if m.Title = strings.TrimSpace(*dto.Title); m.Title == "" {
			return Media{}, ErrEmptyMeta
		}
	}
	if dto.Alt != nil {
		if m.Alt = strings.TrimSpace(*dto.Alt); m.Alt == "" {
			return Media{}, ErrEmptyMeta
		}
	}
	if err := s.store.Update(ctx, &m); err != nil {
		return Media{}, err
	}
	return m, nil
}

func (s *Service) Replace(ctx context.Context, id uint, fh *multipart.FileHeader, meta Meta) (Media, error) {
	m, err := s.store.FindByID(ctx, id)
	if err != nil {
		return Media{}, err
	}
	sf, err := s.writeImage(fh)
	if err != nil {
		return Media{}, err
	}
	old := m.FileName
	sf.apply(&m)
	if title := strings.TrimSpace(meta.Title); title != "" {
		m.Title = title
	}
	if alt := strings.TrimSpace(meta.Alt); alt != "" {
		m.Alt = alt
	}
	if err := s.store.Update(ctx, &m); err != nil {
		os.Remove(s.path(sf.name))
		return Media{}, err
	}
	if fileNamePattern.MatchString(old) {
		os.Remove(s.path(old))
	}
	return m, nil
}

func (s *Service) Delete(ctx context.Context, id uint) error {
	m, err := s.store.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.store.Delete(ctx, id); err != nil {
		return err
	}
	if fileNamePattern.MatchString(m.FileName) {
		if err := os.Remove(s.path(m.FileName)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (s *Service) Resolve(name string) (string, error) {
	name = strings.ToLower(name)
	if !fileNamePattern.MatchString(name) {
		return "", ErrInvalidFileName
	}
	p := s.path(name)
	info, err := os.Stat(p)
	if err != nil || !info.Mode().IsRegular() {
		return "", ErrInvalidFileName
	}
	return p, nil
}
