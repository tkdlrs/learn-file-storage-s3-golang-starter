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

	//
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	//
	newFile, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer newFile.Close()
	//
	mediaType := header.Header.Get("Content-Type")
	if mediaType == "" {
		respondWithError(w, http.StatusBadRequest, "Missing Content-Type for thumbnail", nil)
		return
	}
	fileExtension := strings.Split(mediaType, "/")[1]
	//
	// data, err := io.ReadAll(newFile)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Error reading file data", err)
	// 	return
	// }

	//
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to update this video", nil)
		return
	}
	//
	fileName := videoIDString + "." + fileExtension
	thumbnailFilePath := filepath.Join(cfg.assetsRoot, fileName)
	thumbnailURL := fmt.Sprintf("http://localhost:%s/%s", cfg.port, thumbnailFilePath)
	//
	fileInFileSystem, err := os.Create(thumbnailFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create thumbnail", err)
		return
	}
	defer fileInFileSystem.Close()
	//
	if _, err := io.Copy(fileInFileSystem, newFile); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Ya dun fucked up.", err)
		return
	}

	//
	video.ThumbnailURL = &thumbnailURL
	//
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update video", err)
		return
	}
	//
	respondWithJSON(w, http.StatusOK, video)
}
