package database

import (
	"bytes"
	"context"
	"fmt"
	"journal/internal/logger"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-co-op/gocron/v2"
)

const (
	backupBucket = "contourjournal"
	backupPrefix = "backups/"

	// Retention windows: keep every backup for 30 days, then thin to
	// one per week until backups are 6 months old, then one per month.
	dailyRetention = 30 * 24 * time.Hour
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
		return
	}

	if err := pruneBackups(); err != nil {
		logger.Error("prune failed", err)
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

func newS3Client() *s3.Client {
	return s3.NewFromConfig(aws.Config{
		Region:       "auto",
		BaseEndpoint: aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", os.Getenv("R2_ACCOUNT_ID"))),
		Credentials:  credentials.NewStaticCredentialsProvider(os.Getenv("R2_ACCESS_KEY_ID"), os.Getenv("R2_SECRET_ACCESS_KEY"), ""),
	})
}

func upload(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	client := newS3Client()

	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(backupBucket),
		Key:    aws.String(backupPrefix + filepath.Base(filePath)),
		Body:   f,
	})
	return err
}

// backupObject pairs an R2 object key with the timestamp encoded in its
// filename (see backup()'s outPath format).
type backupObject struct {
	key string
	t   time.Time
}

// parseBackupTime extracts the timestamp from a key like
// "backups/backup-2006-01-02-15-04-05.sql".
func parseBackupTime(key string) (time.Time, error) {
	name := strings.TrimSuffix(filepath.Base(key), ".sql")
	name = strings.TrimPrefix(name, "backup-")
	return time.Parse("2006-01-02-15-04-05", name)
}

// pruneBackups lists all backups in R2 and deletes the ones that fall
// outside the retention policy: everything younger than 30 days is kept,
// backups between 30 days and 6 months old are thinned to one per
// (ISO) week, and anything older than 6 months is thinned to one per
// calendar month.
func pruneBackups() error {
	client := newS3Client()
	ctx := context.Background()

	var backups []backupObject
	var continuationToken *string
	for {
		out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(backupBucket),
			Prefix:            aws.String(backupPrefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return fmt.Errorf("list objects failed: %w", err)
		}

		for _, obj := range out.Contents {
			key := aws.ToString(obj.Key)
			t, err := parseBackupTime(key)
			if err != nil {
				continue
			}
			backups = append(backups, backupObject{key: key, t: t})
		}

		if out.IsTruncated == nil || !*out.IsTruncated {
			break
		}
		continuationToken = out.NextContinuationToken
	}

	toDelete := backupsToPrune(backups, time.Now())
	if len(toDelete) == 0 {
		return nil
	}

	objects := make([]types.ObjectIdentifier, 0, len(toDelete))
	for _, key := range toDelete {
		objects = append(objects, types.ObjectIdentifier{Key: aws.String(key)})
	}

	_, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(backupBucket),
		Delete: &types.Delete{Objects: objects},
	})
	if err != nil {
		return fmt.Errorf("delete objects failed: %w", err)
	}

	logger.Info("pruned backups", "count", len(toDelete))
	return nil
}

// backupsToPrune applies the retention policy and returns the keys that
// should be deleted, keeping the most recent backup within each
// week/month bucket once a backup ages out of the daily window.
func backupsToPrune(backups []backupObject, now time.Time) []string {
	sort.Slice(backups, func(i, j int) bool { return backups[i].t.Before(backups[j].t) })

	thirtyDaysAgo := now.Add(-dailyRetention)
	sixMonthsAgo := now.AddDate(0, -6, 0)

	type bucketKey struct{ a, b int }
	weeklyKeep := map[bucketKey]backupObject{}
	monthlyKeep := map[bucketKey]backupObject{}
	keep := map[string]bool{}

	for _, b := range backups {
		switch {
		case b.t.After(thirtyDaysAgo):
			keep[b.key] = true
		case b.t.After(sixMonthsAgo):
			year, week := b.t.ISOWeek()
			k := bucketKey{year, week}
			if cur, ok := weeklyKeep[k]; !ok || b.t.After(cur.t) {
				weeklyKeep[k] = b
			}
		default:
			k := bucketKey{b.t.Year(), int(b.t.Month())}
			if cur, ok := monthlyKeep[k]; !ok || b.t.After(cur.t) {
				monthlyKeep[k] = b
			}
		}
	}
	for _, b := range weeklyKeep {
		keep[b.key] = true
	}
	for _, b := range monthlyKeep {
		keep[b.key] = true
	}

	var toDelete []string
	for _, b := range backups {
		if !keep[b.key] {
			toDelete = append(toDelete, b.key)
		}
	}
	return toDelete
}
