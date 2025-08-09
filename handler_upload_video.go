package main

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	//
	const maxMemory = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)
	//
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	//
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	//
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
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
	// Pare upload file from form data
	r.ParseMultipartForm(maxMemory)
	videoFile, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer videoFile.Close()
	// Validate uploaded file
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", err)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Incorrect Content-Type provided for video", nil)
		return
	}
	//
	// assetPath := getAssetPath(mediaType)
	// assetDiskPath := cfg.getAssetDiskPath(assetPath)
	//
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create temp video file on server", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close() // Defer is LIFO. This ensures it is closed before attempting the remove above.
	// Rest tempFile's file pointer to beginning
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to rewind video", err)
		return
	}
	//
	// if _, err := io.Copy(tempFile, videoFile); err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Error saving the video file", err)
	// 	return
	// }
	//
	videoAssetPath := getAssetPath(mediaType)
	// Prepare PutObjectInput
	putObjectInput := &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &videoAssetPath,
		Body:        tempFile,
		ContentType: &mediaType,
	}

	// Perform the PutObject operation
	_, err = cfg.s3Client.PutObject(context.TODO(), putObjectInput)
	if err != nil {
		fmt.Printf("Error uploading object to S3: %v\n", err)
		return
	}
	// //
	videoAwsS3URL := fmt.Sprintf("https://%v.s3.%v.amazonaws.com/%v", cfg.s3Bucket, cfg.s3Region, videoAssetPath)
	video.VideoURL = &videoAwsS3URL
	// //
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update video", err)
		return
	}
	//
	respondWithJSON(w, http.StatusOK, video)
}
