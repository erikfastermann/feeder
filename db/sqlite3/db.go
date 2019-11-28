package sqlite3

import (
	"context"
	"database/sql"

	"github.com/erikfastermann/feeder/db"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func Open(ctx context.Context, path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	err = sqlDB.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	_, err = sqlDB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS feeds (
		id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		url TEXT NOT NULL,
		feed_url TEXT NOT NULL,
		last_checked DATETIME
		last_updated DATETIME
	)`)
	if err != nil {
		return nil, err
	}

	_, err = sqlDB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		feed_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		url TEXT NOT NULL,
		added DATETIME NOT NULL,
		FOREIGN KEY(feed_id) REFERENCES feeds(id)
	)`)
	if err != nil {
		return nil, err
	}

	return &DB{sqlDB}, nil
}

func (sqlDB *DB) AllFeeds(ctx context.Context) ([]db.Feed, error) {
	rows, err := sqlDB.QueryContext(ctx, `SELECT id, url, feed_url, last_checked, last_updated
		FROM feeds`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feeds := make([]db.Feed, 0)
	for rows.Next() {
		var feed db.Feed
		err := rows.Scan(&feed.ID, &feed.URL, &feed.FeedURL, &feed.LastChecked, &feed.LastUpdated)
		if err != nil {
			return feeds, err
		}
		feeds = append(feeds, feed)
	}

	return feeds, rows.Err()
}

func (sqlDB *DB) Newest(ctx context.Context, offset, limit uint) ([]db.Item, error) {
	rows, err := sqlDB.QueryContext(ctx, `SELECT id, feed_id, title, url, added
		FROM items
		ORDER BY added DESC
		LIMIT ?
		OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]db.Item, 0)
	for rows.Next() {
		var item db.Item
		err := rows.Scan(&item.ID, &item.FeedID, &item.Title, &item.URL, &item.Added)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (sqlDB *DB) AddFeed(ctx context.Context, feed db.Feed) (int64, error) {
	id := int64(-1)
	err := sqlDB.asTx(ctx, func(tx *sql.Tx) error {
		var count int
		err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*)
			FROM feeds
			WHERE feed_url=?`,
			feed.URL).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}

		res, err := sqlDB.ExecContext(ctx, `INSERT INTO
			feeds(url, feed_url, last_checked, last_updated)
			VALUES(?, ?, ?, ?)`,
			feed.URL, feed.FeedURL, feed.LastChecked, feed.LastUpdated)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		return err
	})
	if err != nil {
		return -1, err
	}
	return id, nil
}

func (sqlDB *DB) AddItems(ctx context.Context, items []db.Item) ([]db.Item, error) {
	addedItems := make([]db.Item, 0)

	for _, item := range items {
		err := sqlDB.asTx(ctx, func(tx *sql.Tx) error {
			var count int
			err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*)
				FROM items
				WHERE title=? AND added=?`,
				item.Title, item.Added).Scan(&count)
			if err != nil {
				return err
			}
			if count > 0 {
				return nil
			}

			res, err := sqlDB.ExecContext(ctx, `INSERT INTO
				items(feed_id, title, url, added)
				VALUES(?, ?, ?, ?)`,
				item.FeedID, item.Title, item.URL, item.Added)
			if err != nil {
				return err
			}
			item.ID, err = res.LastInsertId()
			if err != nil {
				return err
			}
			addedItems = append(addedItems, item)
			return nil
		})
		if err != nil {
			return addedItems, err
		}
	}
	return addedItems, nil
}

func (sqlDB *DB) asTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		if err := tx.Rollback(); err != nil {
			return err
		}
		return err
	}
	return tx.Commit()
}
