package parser

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/erikfastermann/feeder/db"
)

type item struct {
	Title     string `xml:"title"`
	Updated   string `xml:"updated"`
	PubDate   string `xml:"pubDate"`
	Published string `xml:"published"`
	Link      struct {
		Text string `xml:",chardata"`
		Href string `xml:"href,attr"`
	} `xml:"link"`
}

func Items(url string) ([]db.Item, error) {
	c := &http.Client{Timeout: 10 * time.Second}
	res, err := c.Get(url)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	items, err := parse(data)
	if err != nil {
		return nil, err
	}

	finals := make([]db.Item, 0)
	for _, i := range items {
		final := db.Item{
			ID:     -1,
			FeedID: -1,
			Title:  i.Title,
		}

		switch {
		case i.Link.Text != "":
			final.URL = i.Link.Text
		case i.Link.Href != "":
			final.URL = i.Link.Href
		default:
			return nil, errors.New("post without a link")
		}

		var dateStr string
		switch {
		case i.Updated != "":
			dateStr = i.Updated
		case i.PubDate != "":
			dateStr = i.PubDate
		case i.Published != "":
			dateStr = i.Published
		default:
			return nil, errors.New("post without a date")
		}
		final.Added, err = parseDate(dateStr)
		if err != nil {
			return nil, err
		}

		finals = append(finals, final)
	}
	return finals, nil
}

func parse(data []byte) ([]item, error) {
	feed := struct {
		XMLName xml.Name `xml:"feed"`
		Entries []item   `xml:"entry"`
	}{}
	if err := xml.Unmarshal(data, &feed); err != nil {
		rss := struct {
			XMLName xml.Name `xml:"rss"`
			Channel struct {
				Items []item `xml:"item"`
			} `xml:"channel"`
		}{}
		if err := xml.Unmarshal(data, &rss); err != nil {
			return nil, err
		}
		return rss.Channel.Items, nil
	}
	return feed.Entries, nil
}

func parseDate(str string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC1123,     // "Mon, 02 Jan 2006 15:04:05 MST"
		time.RFC1123Z,    // "Mon, 02 Jan 2006 15:04:05 -0700"
		time.RFC3339,     // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano, // "2006-01-02T15:04:05.999999999Z07:00"
	} {
		t, err := time.Parse(layout, str)
		if err != nil {
			continue
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("time %s is invalid", str)
}
