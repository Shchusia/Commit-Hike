package app

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Shchusia/commit-hike/core/internal/protocol"
)

// Hiker icons are small sprites drawn on the trail, so the limits are tight.
const (
	maxAvatarBytes = 256 << 10 // 256 KiB
	maxAvatarSide  = 512       // px
	avatarFile     = "avatar.png"
)

func (s *Service) avatarPath() string { return filepath.Join(s.st.Dir, avatarFile) }

// Avatar returns the custom hiker icon, if the user set one.
func (s *Service) Avatar() (*protocol.Avatar, error) {
	data, err := os.ReadFile(s.avatarPath())
	if errors.Is(err, fs.ErrNotExist) {
		return &protocol.Avatar{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &protocol.Avatar{Custom: true, DataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)}, nil
}

// SetAvatar validates a PNG (ideally with a transparent background) and
// stores it as the hiker icon.
func (s *Service) SetAvatar(src string) (*protocol.Avatar, error) {
	fi, err := os.Stat(src)
	if err != nil {
		return nil, fail(protocol.CodeInvalidImage, "%s", err)
	}
	if fi.Size() > maxAvatarBytes {
		return nil, fail(protocol.CodeInvalidImage, "the image is too large (max %d KiB)", maxAvatarBytes>>10)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return nil, err
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fail(protocol.CodeInvalidImage, "not a PNG image: %s", err)
	}
	if cfg.Width > maxAvatarSide || cfg.Height > maxAvatarSide || cfg.Width < 8 || cfg.Height < 8 {
		return nil, fail(protocol.CodeInvalidImage, "the image must be between 8 and %d px on each side, got %dx%d",
			maxAvatarSide, cfg.Width, cfg.Height)
	}
	unlock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := s.st.WriteFile(avatarFile, data); err != nil {
		return nil, err
	}
	return s.Avatar()
}

// ResetAvatar goes back to the default hiker.
func (s *Service) ResetAvatar() error {
	err := os.Remove(s.avatarPath())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
