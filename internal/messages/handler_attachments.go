package messages

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/cdlinkin/pingpong/internal/auth"
	"github.com/cdlinkin/pingpong/internal/httpx"
)

// SendAttachment godoc
//
//	@Summary		Отправить файл
//	@Description	Загружает файл в чат как новое сообщение (multipart/form-data). Максимум 10 МБ. Разрешены картинки (JPEG, PNG, GIF, WebP), PDF, ZIP и обычный текст; тип определяется по содержимому файла, а не по расширению. Подпись `message` необязательна (до 500 символов). Участники чата онлайн получают новое сообщение по WebSocket так же, как обычное.
//	@Tags			messages
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		int			true	"ID чата"
//	@Param			file		formData	file		true	"Файл до 10 МБ"
//	@Param			message		formData	string		false	"Подпись (до 500 символов)"
//	@Param			reply_to_id	formData	int			false	"ID сообщения, на которое это ответ"
//	@Success		201			{object}	msgResponse	"Сообщение с вложением"
//	@Failure		400			{string}	string		"file is required / file is empty / caption must be up to 500 symbols"
//	@Failure		401			{string}	string		"invalid token"
//	@Failure		404			{string}	string		"chat not found / reply message not found"
//	@Failure		413			{string}	string		"file too large"
//	@Failure		415			{string}	string		"unsupported file type"
//	@Failure		500			{string}	string		"internal error"
//	@Router			/chats/{id}/attachments [post]
func (h *Handler) SendAttachment(rw http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(
			rw,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}
	chatID, err := httpx.PathInt(r, "id")
	if err != nil {
		http.Error(
			rw,
			"invalid chat id",
			http.StatusBadRequest,
		)
		return
	}

	r.Body = http.MaxBytesReader(rw, r.Body, MaxAttachmentSize+1<<20)

	file, header, err := r.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(
				rw,
				"file too large",
				http.StatusRequestEntityTooLarge,
			)
			return
		}
		http.Error(
			rw,
			"file is required",
			http.StatusBadRequest,
		)
		return
	}
	defer file.Close()

	var replyToID *int
	if s := r.FormValue("reply_to_id"); s != "" {
		id, err := strconv.Atoi(s)
		if err != nil {
			http.Error(
				rw,
				"invalid reply_to_id",
				http.StatusBadRequest,
			)
			return
		}
		replyToID = &id
	}

	msg, err := h.service.SendAttachment(
		r.Context(),
		chatID,
		userID,
		r.FormValue("message"),
		replyToID,
		header.Filename,
		file,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmptyFile):
			http.Error(rw, "file is empty", http.StatusBadRequest)
		case errors.Is(err, ErrFileTooLarge):
			http.Error(rw, "file too large", http.StatusRequestEntityTooLarge)
		case errors.Is(err, ErrUnsupportedFile):
			http.Error(rw, "unsupported file type", http.StatusUnsupportedMediaType)
		case errors.Is(err, ErrMessageNotFound):
			http.Error(rw, "reply message not found", http.StatusNotFound)
		default:
			h.writeError(rw, "send attachment", err, "caption must be up to 500 symbols")
		}
		return
	}

	response := toResponse(msg)

	h.broadcast(r.Context(), chatID, response)

	if err := httpx.WriteJSON(rw, http.StatusCreated, response); err != nil {
		h.logger.Warn("send attachment: encode response", "err", err)
	}
}

// GetAttachment godoc
//
//	@Summary		Скачать файл
//	@Description	Отдаёт содержимое вложения. Доступно только участникам чата. Картинки открываются в браузере (`inline`), остальные файлы скачиваются (`attachment`). Если сообщение удалено или вы не участник чата, ответ 404.
//	@Tags			messages
//	@Produce		octet-stream
//	@Security		BearerAuth
//	@Param			id	path		int		true	"ID вложения"
//	@Success		200	{file}		file	"Содержимое файла"
//	@Failure		400	{string}	string	"invalid attachment id"
//	@Failure		401	{string}	string	"invalid token"
//	@Failure		404	{string}	string	"attachment not found"
//	@Failure		500	{string}	string	"internal error"
//	@Router			/attachments/{id} [get]
func (h *Handler) GetAttachment(rw http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(
			rw,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}
	attachmentID, err := httpx.PathInt(r, "id")
	if err != nil {
		http.Error(
			rw,
			"invalid attachment id",
			http.StatusBadRequest,
		)
		return
	}

	a, file, err := h.service.GetAttachment(r.Context(), userID, attachmentID)
	if err != nil {
		if errors.Is(err, ErrAttachmentNotFound) {
			http.Error(
				rw,
				"attachment not found",
				http.StatusNotFound,
			)
			return
		}
		h.writeError(rw, "get attachment", err, "invalid attachment id")
		return
	}
	defer file.Close()

	disposition := "attachment"
	if strings.HasPrefix(a.ContentType, "image/") {
		disposition = "inline"
	}

	rw.Header().Set("Content-Type", a.ContentType)
	rw.Header().Set("X-Content-Type-Options", "nosniff")
	rw.Header().Set("Cache-Control", "private, max-age=86400")
	rw.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("%s; filename*=UTF-8''%s", disposition, url.PathEscape(a.FileName)),
	)

	http.ServeContent(rw, r, "", a.CreatedAt, file)
}
