package database

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// ConnectWithRetry connects the database, retrying once per interval until it
// succeeds. This mirrors the historical server-startup behavior: start early,
// wait for the database to come up rather than crash. Non-connection failures
// (invalid configuration etc.) therefore also keep retrying; a caller that
// wants bounded retries can use ConnectWithRetryBounded.
func ConnectWithRetry(db Database, interval time.Duration) error {
	for {
		err := db.Connect()
		if err == nil {
			return nil
		}
		log.WithFields(log.Fields{"Error": err}).Error("Failed to connect to database, retrying ...")
		time.Sleep(interval)
	}
}

// ConnectWithRetryBounded connects the database with a retry budget. It
// returns the last error after maxAttempts failed attempts.
func ConnectWithRetryBounded(db Database, interval time.Duration, maxAttempts int) error {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := db.Connect()
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt < maxAttempts-1 {
			log.WithFields(log.Fields{
				"Error":   err,
				"Attempt": attempt + 1,
				"Of":      maxAttempts,
			}).Error("Failed to connect to database, retrying ...")
			time.Sleep(interval)
		}
	}
	return lastErr
}
