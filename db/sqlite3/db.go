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

	_, err = sqlDB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS items (
		feed_title VARCHAR(32) NOT NULL,
		author VARCHAR(32) NOT NULL,
		title VARCHAR(32) NOT NULL,
		description TEXT NOT NULL,
		link VARCHAR(63) NOT NULL,
		image_url VARCHAR(63) NOT NULL,
		updated DATETIME NOT NULL
	)`)
	if err != nil {
		return nil, err
	}

	return &DB{sqlDB}, nil
}

func (sqlDB *DB) Newest(ctx context.Context, n uint) ([]db.Item, error) {
	rows, err := sqlDB.QueryContext(ctx, `SELECT feed_title, author, title,
		description, link, image_url, updated
		FROM items
		ORDER BY updated DESC
		LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]db.Item, 0)
	for rows.Next() {
		var item db.Item
		err := rows.Scan(&item.FeedTitle, &item.Author, &item.Title,
			&item.Description, &item.Link, &item.ImageURL, &item.Updated)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (sqlDB *DB) AddItems(ctx context.Context, items []db.Item) ([]db.Item, error) {
	addedItems := make([]db.Item, 0)

	for _, item := range items {
		err := sqlDB.asTx(ctx, func(tx *sql.Tx) error {
			var count int
			err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*)
				FROM items
				WHERE title=? AND updated=?`,
				item.Title, item.Updated).Scan(&count)
			if err != nil {
				return err
			}
			if count > 0 {
				return nil
			}

			_, err = sqlDB.ExecContext(ctx, `INSERT INTO
				items(feed_title, author, title, description, link, image_url, updated)
				VALUES(?, ?, ?, ?, ?, ?, ?)`,
				item.FeedTitle, item.Author, item.Title,
				item.Description, item.Link, item.ImageURL, item.Updated)
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
