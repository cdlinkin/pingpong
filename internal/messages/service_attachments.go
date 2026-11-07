package messages

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const MaxAttachmentSize = 10 << 20

var allowedFileTypes = map[string]bool{
	"image/jpeg":                true,
	"image/png":                 true,
	"image/gif":                 true,
	"image/webp":                true,
	"application/pdf":           true,
	"application/zip":           true,
	"text/plain; charset=utf-8": true,
}

func (s *Service) SendAttachment(
	ctx context.Context,
	chatID, userID int,
	caption string,
	replyToID *int,
	fileName string,
	file io.Reader,
) (MessageModel, error) {
	caption, err := validateCaption(caption)
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"validate caption: %w",
			err,
		)
	}

	if err := s.ensureMember(ctx, chatID, userID); err != nil {
		return MessageModel{}, err
	}

	if err := s.ensureReplyTarget(ctx, chatID, replyToID); err != nil {
		return MessageModel{}, err
	}

	br := bufio.NewReaderSize(file, 512)
	head, err := br.Peek(512)
	if err != nil && !errors.Is(err, io.EOF) {
		return MessageModel{}, fmt.Errorf(
			"read file: %w",
			err,
		)
	}
	if len(head) == 0 {
		return MessageModel{}, ErrEmptyFile
	}

	contentType := http.DetectContentType(head)
	if !allowedFileTypes[contentType] {
		return MessageModel{}, fmt.Errorf(
			"%s: %w",
			contentType,
			ErrUnsupportedFile,
		)
	}

	key, size, err := s.storage.Save(ctx, io.LimitReader(br, MaxAttachmentSize+1))
	if err != nil {
		return MessageModel{}, fmt.Errorf(
			"save file: %w",
			err,
		)
	}
	if size > MaxAttachmentSize {
		s.storage.Remove(ctx, key)
		return MessageModel{}, ErrFileTooLarge
	}

	msg, err := s.repository.CreateAttachmentMessage(
		ctx,
		chatID,
		userID,
		caption,
		replyToID,
		AttachmentModel{
			FileName:    cleanFileName(fileName),
			ContentType: contentType,
			Size:        size,
			StorageKey:  key,
		},
	)
	if err != nil {
		s.storage.Remove(ctx, key)
		return MessageModel{}, fmt.Errorf(
			"create attachment message: %w",
			err,
		)
	}

	return msg, nil
}

func (s *Service) GetAttachment(
	ctx context.Context,
	userID, attachmentID int,
) (AttachmentModel, io.ReadSeekCloser, error) {
	a, err := s.repository.GetAttachment(ctx, attachmentID)
	if err != nil {
		return AttachmentModel{}, nil, fmt.Errorf(
			"get attachment: %w",
			err,
		)
	}

	if a.Deleted {
		return AttachmentModel{}, nil, ErrAttachmentNotFound
	}

	if err := s.ensureMember(ctx, a.ChatID, userID); err != nil {
		if errors.Is(err, ErrChatNotFound) {
			return AttachmentModel{}, nil, ErrAttachmentNotFound
		}
		return AttachmentModel{}, nil, err
	}

	f, err := s.storage.Open(ctx, a.StorageKey)
	if err != nil {
		return AttachmentModel{}, nil, fmt.Errorf(
			"open file: %w",
			err,
		)
	}

	return a, f, nil
}

func (s *Service) fillAttachments(
	ctx context.Context,
	messages []MessageModel,
	ids []int,
) error {
	attachments, err := s.repository.ListAttachments(ctx, ids)
	if err != nil {
		return fmt.Errorf(
			"list attachments: %w",
			err,
		)
	}

	byMsg := make(map[int]AttachmentModel, len(attachments))
	for _, a := range attachments {
		byMsg[a.MsgID] = a
	}

	for i := range messages {
		if a, ok := byMsg[messages[i].ID]; ok && messages[i].DeletedAt == nil {
			messages[i].Attachment = &a
		}
	}

	return nil
}
