package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	file, header, err := r.FormFile("thumbnail")
	contentType := header.Header.Get("Content-Type")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video information", err)
		return
	}
	if userID != video.UserID {
		respondWithError(w, http.StatusUnauthorized, "User is not the owner", nil)
		return
	}

	fileExtension := strings.Split(contentType, "/")[1]
	targetPath := filepath.Join(cfg.assetsRoot, videoIDString+"."+fileExtension)
	fp, err := os.Create(targetPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to start saving thumbnail", err)
		return
	}
	defer fp.Close()
	_, err = io.Copy(fp, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save thumbnail", err)
		return
	}
	thumbnailUrl := fmt.Sprintf("http://localhost:%s/assets/%s.%s", cfg.port, videoIDString, fileExtension)
	video.ThumbnailURL = &thumbnailUrl
	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save video information", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
