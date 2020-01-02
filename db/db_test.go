package db

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestDB(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "feeder-db-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// d, err := Init("ctr.csv", "feeds.csv", "items.csv")
	path := func(path string) string {
		return filepath.Join(dir, path)
	}
	d, err := Open(path("ctr.csv"), path("feeds.csv"), path("items.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	if _, err := d.AllFeeds(); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Newest(0, 30); err != sql.ErrNoRows {
		t.Fatal(err)
	}

	feeds := make([]Feed, 0)
	for i := 1; i < 4; i++ {
		s := strconv.Itoa(i)
		f := Feed{
			ID:      i,
			Host:    "host" + s,
			FeedURL: "url" + s,
		}

		if _, err := d.AddFeed(f.Host, f.FeedURL); err != nil {
			t.Fatal(err)
		}

		feeds = append(feeds, f)
	}

	feeds2, err := d.AllFeeds()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(feeds, feeds2) {
		t.Fatalf("feeds don't match after store")
	}

	newHost := "blubber"
	feeds[1].Host = newHost
	if err := d.EditFeedHost(feeds[1].ID, newHost); err != nil {
		t.Fatal(err)
	}
	feeds2, err = d.AllFeeds()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(feeds, feeds2) {
		t.Fatalf("feeds don't match after store")
	}

	if err := d.AddItems(999, nil); err == nil {
		t.Fatal("expected an err, got nil")
	}

	timeNow = func() time.Time {
		return time.Date(2019, time.December, 31, 12, 12, 12, 0, time.Local)
	}

	items := make([]Item, 0)
	iwh := make([]ItemWithHost, 0)
	for i := 0; i < 3; i++ {
		s := strconv.Itoa(i)
		item := Item{
			FeedID: feeds[1].ID,
			Title:  "title" + s,
			URL:    "url" + s,
			Added:  timeNow(),
		}
		items = append(items, item)
		iwh = append(iwh, ItemWithHost{Item: item, Host: feeds[1].Host})
	}

	if err := d.AddItems(feeds[1].ID, items); err != nil {
		t.Fatal(err)
	}
	iwh1, err := d.Newest(0, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(iwh, iwh1) {
		t.Fatal("items don't match after store")
	}

	newItem := Item{FeedID: feeds[2].ID, Title: "some", URL: "thing", Added: timeNow()}
	newIWH := ItemWithHost{Item: newItem, Host: feeds[2].Host}
	iwh = append(iwh, newIWH)
	if err := d.AddItems(feeds[2].ID, []Item{newItem}); err != nil {
		t.Fatal(err)
	}
	nullNow := func() sql.NullTime {
		return sql.NullTime{
			Valid: true,
			Time:  timeNow(),
		}
	}
	feeds[2].LastChecked = nullNow()
	feeds[2].LastUpdated = nullNow()

	iwh1, err = d.Newest(0, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(iwh, iwh1) {
		t.Fatal("items don't match after store")
	}

	if err := d.RemoveFeed(feeds[1].ID); err != nil {
		t.Fatal(err)
	}
	feeds = append(make([]Feed, 0), feeds[2], feeds[0])
	feeds2, err = d.AllFeeds()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(feeds, feeds2) {
		t.Logf("\n%+v\n----\n%+v", feeds, feeds2)
		t.Fatalf("feeds don't match with stored feeds after remove")
	}

	iwh = []ItemWithHost{iwh[3]}
	iwh1, err = d.Newest(0, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(iwh, iwh1) {
		t.Logf("\n%+v\n----\n%+v", iwh, iwh1)
		t.Fatalf("items don't match with stored items after remove")
	}

	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
}
