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
		feed_id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		title VARCHAR(32) NOT NULL,
		author VARCHAR(32) NOT NULL,
		language VARCHAR(16) NOT NULL,
		description TEXT NOT NULL,
		link VARCHAR(63) NOT NULL,
		feed_link VARCHAR(63) NOT NULL,
		image_url VARCHAR(63) NOT NULL,
		last_updated DATETIME
	)`)
	if err != nil {
		return nil, err
	}

	_, err = sqlDB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS items (
		author VARCHAR(32) NOT NULL,
		title VARCHAR(32) NOT NULL,
		description TEXT NOT NULL,
		content TEXT NOT NULL,
		link VARCHAR(63) NOT NULL,
		image_url VARCHAR(63) NOT NULL,
		added DATETIME NOT NULL,
		feed INTEGER NOT NULL,
		FOREIGN KEY(feed) REFERENCES feeds(feed_id)
	)`)
	if err != nil {
		return nil, err
	}

	return &DB{sqlDB}, nil
}

func (sqlDB *DB) Newest(ctx context.Context, n uint) ([]db.Item, error) {
	rows, err := sqlDB.QueryContext(ctx, `SELECT _rowid_, feed, author, title, description,
		content, link, image_url, added
		FROM items
		ORDER BY added DESC
		LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]db.Item, 0)
	for rows.Next() {
		var item db.Item
		err := rows.Scan(&item.ID, &item.FeedID, &item.Author, &item.Title,
			&item.Description, &item.Content, &item.Link, &item.ImageURL, &item.Added)
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
			WHERE feed_link=?`,
			feed.Link).Scan(&count)
		if err != nil {
			return err
		}
		if count > 0 {
			return nil
		}

		res, err := sqlDB.ExecContext(ctx, `INSERT INTO
			feeds(author, title, language, description,
			link, feed_link, image_url, last_updated)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
			feed.Author, feed.Title, feed.Language, feed.Description,
			feed.Link, feed.FeedLink, feed.ImageURL, feed.LastUpdated)
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
				items(feed, author, title, description, content, link, image_url, added)
				VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
				item.FeedID, item.Author, item.Title,
				item.Description, item.Content, item.Link, item.ImageURL, item.Added)
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
