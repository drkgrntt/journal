package database

import (
	"bytes"
	"context"
	"fmt"
	"journal/internal/logger"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-co-op/gocron/v2"
)

func init() {
	s, err := gocron.NewScheduler()
	if err != nil {
		logger.Error("Error starting scheduler", "error", err.Error())
		return
	}

	_, err = s.NewJob(
		gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(20, 0, 30))),
		gocron.NewTask(dailyBackup),
	)

	s.Start()
}

func dailyBackup() {
	if os.Getenv("APP_ENV") != "production" {
		return
	}

	path, err := backup(connectionStr())
	if err != nil {
		logger.Error("err", "backup failed", err)
		return
	}
	defer os.Remove(path)
	if err := upload(path); err != nil {
		logger.Error("upload failed", err)
	}
}

func backup(connStr string) (string, error) {
	outPath := fmt.Sprintf("/tmp/backup-%s.sql", time.Now().Format("2006-01-02-15-04-05"))
	cmd := exec.Command("pg_dump", connStr, "-f", outPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pg_dump failed: %w, stderr: %s", err, stderr.String())
	}
	return outPath, nil
}

func upload(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	client := s3.NewFromConfig(aws.Config{
		Region:       "auto",
		BaseEndpoint: aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", os.Getenv("R2_ACCOUNT_ID"))),
		Credentials:  credentials.NewStaticCredentialsProvider(os.Getenv("R2_ACCESS_KEY_ID"), os.Getenv("R2_SECRET_ACCESS_KEY"), ""),
	})

	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String("contourjournal"),
		Key:    aws.String(fmt.Sprintf("backups/%s", filepath.Base(filePath))),
		Body:   f,
	})
	return err
}
