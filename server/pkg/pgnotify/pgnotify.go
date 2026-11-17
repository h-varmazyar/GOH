// Package pgnotify wakes processes with Postgres LISTEN and NOTIFY. A job
// queue notifies its channel when a job may have become runnable, and the job
// runners of every process listen to it, so they need not poll the job table.
//
// A notification is only a wake-up call: it carries no payload, may be lost
// while a listener reconnects and may be sent for a job another runner already
// took. Listeners therefore treat it as "look at the table now", and also look
// whenever they connect, so a notification missed while disconnected is
// covered.
package pgnotify

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Notify sends a notification on channel. Inside a transaction it is sent when
// the transaction commits, and not at all if it rolls back.
func Notify(db *gorm.DB, channel string) error {
	if channel == "" {
		return nil
	}
	if err := db.Exec("SELECT pg_notify(?, '')", channel).Error; err != nil {
		return fmt.Errorf("notify %s: %w", channel, err)
	}
	return nil
}

// Listener listens to Postgres channels on a connection of its own.
type Listener struct {
	db       *sql.DB
	channels []string
	retry    time.Duration
	logger   *logrus.Entry
}

func NewListener(db *sql.DB, channels ...string) *Listener {
	return &Listener{
		db:       db,
		channels: channels,
		retry:    5 * time.Second,
		logger:   logrus.WithField("component", "pgnotify"),
	}
}

// Run calls wake once the channels are listened to, and again on every
// notification, until ctx is done. A lost connection is opened again, and wake
// is called once it is, since notifications sent meanwhile were lost.
func (l *Listener) Run(ctx context.Context, wake func()) {
	for ctx.Err() == nil {
		err := l.listen(ctx, wake)
		if ctx.Err() != nil {
			return
		}
		l.logger.WithError(err).WithField("channels", l.channels).Warn("listen to Postgres notifications; retrying")
		timer := time.NewTimer(l.retry)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (l *Listener) listen(ctx context.Context, wake func()) error {
	conn, err := l.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(driverConn any) error {
		stdConn, ok := driverConn.(*stdlib.Conn)
		if !ok {
			return fmt.Errorf("listening needs the pgx driver, got %T", driverConn)
		}
		pgConn := stdConn.Conn()
		for _, channel := range l.channels {
			if _, err := pgConn.Exec(ctx, "LISTEN "+pgx.Identifier{channel}.Sanitize()); err != nil {
				// The connection is dropped rather than returned to the pool
				// with some channels listened to.
				return errors.Join(err, driver.ErrBadConn)
			}
		}
		wake()
		for {
			if _, err := pgConn.WaitForNotification(ctx); err != nil {
				return errors.Join(err, driver.ErrBadConn)
			}
			wake()
		}
	})
}
