package main

// Copyright Biko Pougala, 2020. All rights reserved.
// Created on 28 May, 2020

// Creates a web server to serve videos stored in Google Cloud Storage buckets
import (
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"context"
	firebase "firebase.google.com/go"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	//"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func handlers() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/", indexPage).Methods("GET")
	router.HandleFunc("/serveVideos/{classID}/{mId}/", streamHandler).Methods("GET")
	router.HandleFunc("/serveVideos/{classID}/{mId}/{segName}", streamHandler).Methods("GET")
	router.HandleFunc("/S3/", indexPageS3).Methods("GET")
	router.HandleFunc("/serveVideosS3/{classID}/{mId}/{quality}/", streamHandlerS3).Methods("GET")
	router.HandleFunc("/serveVideosS3/{classID}/{mId}/{quality}/{segName}", streamHandlerS3).Methods("GET")
	return router
}

func accessNewAWSCredentials() []string {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Printf("[ERROR] 1 %s", err)
	}
	idName := os.Getenv("AWS_ID_LOC")
	if idName == "" {
		idName = "projects/768996125988/secrets/AWS_ACCESS_KEY_ID/versions/latest"
	}
	secretName := os.Getenv("AWS_SECRET_LOC")
	if secretName == "" {
		secretName = "projects/768996125988/secrets/AWS_SECRET_ACCESS_KEY/versions/latest"
	}
	req := &secretmanagerpb.AccessSecretVersionRequest{Name: idName}
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		log.Printf("[ERROR] 2 %s", err)
	}
	var results []string
	resultStr := string(result.Payload.Data)
	results = append(results, resultStr)
	req = &secretmanagerpb.AccessSecretVersionRequest{Name: secretName}
	result, err = client.AccessSecretVersion(ctx, req)
	if err != nil {
		log.Printf("[ERROR] 2 %s", err)
	}
	resultStr = string(result.Payload.Data)
	results = append(results, resultStr)
	return results
}
func main() {

	http.Handle("/", handlers())
	port := os.Getenv("PORT")
	log.Fatal(http.ListenAndServe(":"+port, nil))
	//receiveAccountEndPoint()
}
func S3handlers() *mux.Router {
	router := mux.NewRouter()

	return router
}

func indexPageS3(w http.ResponseWriter, r *http.Request) {
	fmt.Println("content loaded")
	creds := accessNewAWSCredentials()
	region := aws.String("eu-west-1")
	bucket := aws.String(os.Getenv("AWS_VIDEO_BUCKET"))
	sess, err := session.NewSession(&aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentials(creds[0], creds[1], ""),
	})
	if err != nil {
		log.Fatalln("[ERROR] ", err)
	}
	downloader := s3manager.NewDownloader(sess)
	fileName := "temp-" + string(rand.Intn(100))
	tempFile, err := os.Create(fileName)
	if err != nil {
		log.Printf("err: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("ERROR"))
	}
	_, err = downloader.Download(tempFile,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    aws.String("videoindex.html"),
		})
	if err != nil {
		log.Printf("err: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("ERROR"))
	}

	http.ServeFile(w, r, fileName)

	_ = os.Remove(fileName)
}
func fetchVideoM3U8FileFromS3(fileName, classID, quality string) ([]byte, error) {
	creds := accessNewAWSCredentials()
	region := aws.String("eu-west-1")
	bucket := aws.String(os.Getenv("AWS_VIDEO_BUCKET_1"))
	sess, err := session.NewSession(&aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentials(creds[0], creds[1], ""),
	})
	if err != nil {
		log.Fatalln("[ERROR] ", err)
	}
	re := regexp.MustCompile("[0-9]")
	num := re.FindAllString(fileName, -1)
	digits := strings.Join(num, "")
	digit, err := strconv.Atoi(digits)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var episodeNumber string
	if digit < 10 {
		episodeNumber = fmt.Sprintf("0%v", digit)
	} else {
		episodeNumber = strconv.Itoa(digit)
	}
	log.Printf("episodeNumber: %v\n", episodeNumber)
	downloader := s3manager.NewDownloader(sess)
	tempBuffer := aws.NewWriteAtBuffer([]byte{})
	numBytes, err := downloader.Download(tempBuffer,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    aws.String(classID + "/" + fileName + "/" + "lovevictor" + episodeNumber + ".mkv-" + quality + ".m3u8"),
		})
	if err != nil {
		log.Printf("[DOWNLOAD ERROR] Manifest file %v\n", err)
		log.Printf(classID + "/" + fileName + "/" + "lovevictor08.mkv-" + quality + ".m3u8")
		return nil, err
	}
	if numBytes > 0 {
		return tempBuffer.Bytes(), nil
	} else {
		log.Println("Error getting downloaded bytes")
		return nil, err
	}

}

func fetchVideoChunkS3(classID, fileName, segName string) ([]byte, error) {
	region := aws.String("eu-west-1")
	bucket := aws.String(os.Getenv("AWS_VIDEO_BUCKET_1"))
	creds := accessNewAWSCredentials()
	sess, err := session.NewSession(&aws.Config{
		Region: region,
		Credentials: credentials.NewStaticCredentials(creds[0], creds[1], ""),
	})
	if err != nil {
		log.Fatalln("[ERROR] ", err)
	}
	downloader := s3manager.NewDownloader(sess)
	tempBuffer := aws.NewWriteAtBuffer([]byte{})
	numBytes, err := downloader.Download(tempBuffer,
		&s3.GetObjectInput{
			Bucket: bucket,
			Key:    aws.String(classID + "/" + fileName + "/" + segName),
		})
	if err != nil {
		log.Printf("[DOWNLOAD ERROR] %v\n", err)
		return nil, err
	}
	if numBytes > 0 {
		return tempBuffer.Bytes(), nil
	} else {
		log.Println("Error getting downloaded bytes")
		return nil, err
	}
}

func streamHandlerS3(response http.ResponseWriter, request *http.Request) {
	log.Println("starting streaming")
	log.Printf("request headers: %s\n", request.Header.Get("Access-Control-Allow-Origin"))
	vars := mux.Vars(request)
	m3u8Name := vars["classID"]
	mId := vars["mId"]
	quality := vars["quality"]

	segName, ok := vars["segName"]
	if !ok {
		serveHlsM3u8FileFromS3(response, request, m3u8Name, mId, quality)
	} else {
		log.Printf("not ok")
		serveHlsTsFromS3(response, request, m3u8Name, mId, segName)
	}
}

func serveHlsM3u8FileFromS3(w http.ResponseWriter, r *http.Request, fileName, fileID, quality string) {
	fileData, err := fetchVideoM3U8FileFromS3(fileID, fileName, quality)
	if err != nil {
		return
	}
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("[FILE CREATION ERROR] %v\n", err)
		return
	}
	_, err = file.Write(fileData)
	if err != nil {
		log.Printf("[FILE WRITE ERROR] %v\n", err)
		return
	}
	w.Header().Set("Content-Type", "application/w-mpegURL")
	enableCors(&w)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	http.ServeFile(w, r, fileName)
	_ = os.Remove(fileName)

}

func serveHlsTsFromS3(w http.ResponseWriter, r *http.Request, classID, fileName, segName string) {
	fileData, err := fetchVideoChunkS3(classID, fileName, segName)
	if err != nil {
		return
	}
	file, err := os.Create(fileName)
	if err != nil {
		return
	}
	_, err = file.Write(fileData)
	if err != nil {
		return
	}
	http.ServeFile(w, r, fileName)
	w.Header().Set("Content-Type", "video/MP2T")
	enableCors(&w)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	_ = os.Remove(fileName)
}


func indexPage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("content loaded")
	projectID := os.Getenv("PROJECTID")
	bucket := "classrooms-media-eu"
	config := &firebase.Config{
		ProjectID:     projectID,
		StorageBucket: bucket,
	}
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		log.Printf("[ERROR INITIALISING FIREBASE] %s\n", err)
		_, _ = w.Write([]byte("ERROR"))
		return
	}
	client, err := app.Storage(ctx)
	if err != nil {
		w.Write([]byte("ERROR HAPPENED!"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	bck, err := client.Bucket(bucket)
	objPath := "videoReader.html"
	objReader, err := bck.Object(objPath).NewReader(ctx)
	if err != nil {
		log.Printf("err: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("ERROR"))
	}
	data, err := ioutil.ReadAll(objReader)
	if err != nil {
		log.Printf("err: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("ERROR"))
	}
	file, err := os.Create(objPath)
	if err != nil {
		log.Printf("err: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("ERROR"))
	}
	_, _ = file.Write(data)
	http.ServeFile(w, r, objPath)
	os.Remove(objPath)
}
func fetchVideoM3U8File(fileName, classID string) ([]byte, error) {
	projectID := os.Getenv("PROJECTID")
	bucket := "classrooms-media-eu"
	if projectID == "" {
		projectID = "classrooms-5f5e6"
	}
	config := &firebase.Config{
		ProjectID:     projectID,
		StorageBucket: bucket,
	}
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		log.Printf("[ERROR INITIALISING FIREBASE] %s\n", err)
		return nil, err
	}
	client, err := app.Storage(ctx)
	if err != nil {
		log.Printf("[FIREBASE STORAGE ERROR] %v\n", err)
		return nil, err
	}
	bck, err := client.Bucket(bucket)
	objPath := classID + "/" + fileName + "/index.m3u8"
	if err != nil {
		log.Printf("[BUCKET ERROR] %v\n", err)
		return nil, err
	}
	objReader, err := bck.Object(objPath).NewReader(ctx)
	if err != nil {
		log.Printf("objPath: %s\n", objPath)
		log.Printf("[OBJECT ACCESS ERROR] %v\n", err)
		return nil, err
	}
	data, err := ioutil.ReadAll(objReader)
	if err != nil {
		log.Printf("[OBJECT READING ERROR] %v\n", err)
		return nil, err
	}
	return data, nil
}

/*func setupAWSTranscodingSession() {
	region := os.Getenv("AWS_REGION")
	if region == "" { region = "eu-south-1" }
	creds := accessNewAWSCredentials()
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(creds[0], creds[1], ""),
	})
	if err != nil { log.Fatalln("[ERROR] ", err) }
	transcodeURL := os.Getenv("AWS_MEDIACONVERT_URL")
	if transcodeURL == "" { transcodeURL = "https://r1eeew44a.mediaconvert.eu-west-1.amazonaws.com" }


}*/
func fetchVideoChunk(classID, fileName, segName string) ([]byte, error) {
	projectID := os.Getenv("PROJECTID")
	bucket := "classrooms-media-eu"
	if projectID == "" {
		projectID = "classrooms-5f5e6"
	}
	config := &firebase.Config{
		ProjectID:     projectID,
		StorageBucket: bucket,
	}
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		log.Printf("[ERROR INITIALISING FIREBASE] %s\n", err)
		return nil, err
	}
	client, err := app.Storage(ctx)
	if err != nil {
		log.Printf("[FIREBASE STORAGE ERROR] %v\n", err)
		return nil, err
	}
	bck, err := client.Bucket(bucket)
	objPath := classID + "/" + fileName + "/" + segName
	if err != nil {
		log.Printf("[BUCKET ERROR] %v\n", err)
		log.Printf("objPath: %s\n", objPath)
		return nil, err
	}
	objReader, err := bck.Object(objPath).NewReader(ctx)
	if err != nil {
		log.Printf("[OBJECT ACCESS ERROR] %v\n", err)
		log.Printf("objPath: %s\n", objPath)
		return nil, err
	}
	data, err := ioutil.ReadAll(objReader)
	if err != nil {
		log.Printf("[OBJECT READING ERROR] %v\n", err)
		return nil, err
	}
	return data, nil
}

func streamHandler(response http.ResponseWriter, request *http.Request) {
	fmt.Println("starting streaming")
	fmt.Printf("request headers: %s\n", request.Header.Get("Access-Control-Allow-Origin"))
	vars := mux.Vars(request)
	m3u8Name := vars["classID"]
	mId := vars["mId"]

	segName, ok := vars["segName"]
	if !ok {
		serveHlsM3u8(response, request, m3u8Name, mId)
	} else {
		log.Printf("not ok")
		serveHlsTs(response, request, m3u8Name, mId, segName)
	}
}

func getMediaBase(mId int) string {
	fmt.Println("getting media base")
	mediaRoot := "videos"
	return fmt.Sprintf("%s/%d", mediaRoot, mId)
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
func serveHlsM3u8(w http.ResponseWriter, r *http.Request, fileName, fileID string) {
	fileData, err := fetchVideoM3U8File(fileID, fileName)
	if err != nil {
		return
	}
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("[FILE CREATION ERROR] %v\n", err)
		return
	}
	_, err = file.Write(fileData)
	if err != nil {
		log.Printf("[FILE WRITE ERROR] %v\n", err)
		return
	}
	w.Header().Set("Content-Type", "application/w-mpegURL")
	enableCors(&w)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	http.ServeFile(w, r, fileName)
	_ = os.Remove(fileName)

}

func serveHlsTs(w http.ResponseWriter, r *http.Request, classID, fileName, segName string) {
	fileData, err := fetchVideoChunk(classID, fileName, segName)
	if err != nil {
		return
	}
	file, err := os.Create(fileName)
	if err != nil {
		return
	}
	_, err = file.Write(fileData)
	if err != nil {
		return
	}
	http.ServeFile(w, r, fileName)
	w.Header().Set("Content-Type", "video/MP2T")
	enableCors(&w)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	_ = os.Remove(fileName)
}
