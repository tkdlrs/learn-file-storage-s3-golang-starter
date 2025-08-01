package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
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

	// TODO: implement the upload here
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)
	//
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()
	//
	mediaType := header.Header.Get("Content-Type")
	//
	imageData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get image data", err)
	}
	//
	videoInDatabase, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not get video", err)
		return
	}

	thumbnailURLTemplate := fmt.Sprintf("http://localhost:%v/api/thumbnails/%v ", cfg.port, videoID.String())
	videoInDatabase.ThumbnailURL = &thumbnailURLTemplate
	newVideoThumbnail := thumbnail{
		data:      imageData,
		mediaType: mediaType,
	}

	videoThumbnails[videoID] = newVideoThumbnail
	cfg.db.UpdateVideo(videoInDatabase)
	//
	updatedVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not get video", err)
		return
	}

	//
	respondWithJSON(w, http.StatusOK, database.Video{
		ID:           updatedVideo.ID,
		CreatedAt:    updatedVideo.CreatedAt,
		UpdatedAt:    updatedVideo.UpdatedAt,
		ThumbnailURL: updatedVideo.ThumbnailURL,
		VideoURL:     updatedVideo.VideoURL,
	})
}
