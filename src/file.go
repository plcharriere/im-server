package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"

	"github.com/go-pg/pg/v10"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type File struct {
	Uuid     string
	UserUuid string
	Name     string
	Type     string
	Data     []byte
}

func (s *Server) HttpPostFile(ctx *fasthttp.RequestCtx) {
	token := string(ctx.Request.Header.Peek("token"))

	userUuid, err := s.GetUserUuidByToken(token)
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusUnauthorized)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	fileType := fileHeader.Header.Get("Content-Type")

	fileM, err := fileHeader.Open()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	var buf bytes.Buffer
	io.Copy(&buf, fileM)

	fileM.Close()

	file := &File{
		Uuid:     uuid.New().String(),
		UserUuid: userUuid,
		Name:     fileHeader.Filename,
		Type:     fileType,
		Data:     buf.Bytes(),
	}

	_, err = s.Db.Model(file).Insert()
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	ctx.WriteString(file.Uuid)
}

func (s *Server) HttpGetFile(ctx *fasthttp.RequestCtx) {
	fileUuid := ctx.UserValue("uuid")
	if fileUuid == nil {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	file := &File{
		Uuid: fileUuid.(string),
	}
	err := s.Db.Model(file).WherePK().Select()
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusNotFound)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	ctx.Success(file.Type, file.Data)
}

func (s *Server) HttpGetFileInfos(ctx *fasthttp.RequestCtx) {
	fileUuid := ctx.UserValue("uuid")
	if fileUuid == nil {
		ctx.Error("", fasthttp.StatusBadRequest)
		return
	}

	file := &File{
		Uuid: fileUuid.(string),
	}
	err := s.Db.Model(file).WherePK().Select()
	if err != nil {
		if err == pg.ErrNoRows {
			ctx.Error("", fasthttp.StatusNotFound)
		} else {
			HttpInternalServerError(ctx, err)
		}
		return
	}

	size := strconv.Itoa(len(file.Data))

	infos := []string{
		file.Name, size,
	}

	json, err := json.Marshal(infos)
	if err != nil {
		HttpInternalServerError(ctx, err)
		return
	}

	ctx.Write(json)
}
