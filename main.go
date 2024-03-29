package main

import (
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/net/publicsuffix"

	"github.com/anhdowastaken/fantasypl-crawler/configuration"
	"github.com/anhdowastaken/fantasypl-crawler/logger"
)

type GameAPIResult struct {
	Events []struct {
		ID        int  `json:"id"`
		IsCurrent bool `json:"is_current"`
	} `json:"events"`
}

type LeagueAPIResult struct {
	League struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"league`
	Standings struct {
		HasNext bool `json:"has_next"`
		Page    int  `json:"page"`
		Results []struct {
			ID         int    `json:"id"`
			Entry      int    `json:"entry"`
			EntryName  string `json:"entry_name"`
			PlayerName string `json:"player_name"`
			EventTotal int    `json:"event_total"`
			Total      int    `json:"total"`
			Rank       int    `json:"rank"`
		} `json:"results`
	} `json:"standings`
}

type EntryAPIResult struct {
	Current []struct {
		Event              int `json:event`
		Points             int `json:points`
		EventTransfersCost int `json:event_transfers_cost`
	} `json:currentWeek`
}

type Entry struct {
	ID         int
	EntryNum   int
	EntryName  string
	PlayerName string
	Point      map[int]int
	Total      int
	Rank       int
}

type League struct {
	ID      int
	Name    string
	Entries []Entry
}

const instanceName = "FANTASY-CRAWLER"
const defaultConfigFile = "fantasypl-crawler.conf"

func main() {
	log := logger.New()

	confPath := flag.String("c", "", "Config file of an instance")
	flag.Parse()

	log.SetStreamSingle(os.Stdout)

	log.SetPrefix(strings.ToUpper(instanceName))
	log.Info.Printf("Start %s", strings.ToUpper(instanceName))

	if *confPath == "" {
		log.Critical.Printf("Can not found config path in command line. Use default path instead: %s\n", defaultConfigFile)
		*confPath = defaultConfigFile
	}

	cm := configuration.New()
	if cm.Load(*confPath) != nil {
		log.Critical.Printf("Can not load config file %s\n", *confPath)
		os.Exit(1)
	}

	log.Info.Printf("Log level: %s\n", logger.LOGLEVEL[cm.AppCfg.LogLevel])

	var urlStr string

	// Login
	urlStr = "https://users.premierleague.com/accounts/login/"
	log.Debug.Printf("Login...")
	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	jar, err := cookiejar.New(&options)
	if err != nil {
		log.Critical.Printf("%#v", err)
		os.Exit(1)
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{
		Jar:       jar,
		Transport: tr,
	}
	_, err = client.PostForm(urlStr, url.Values{
		"login":        {cm.FplCfg.Username},
		"password":     {cm.FplCfg.Password},
		"app":          {"plfpl-web"},
		"redirect_uri": {"https://fantasy.premierleague.com/"},
	})
	if err != nil {
		log.Critical.Printf("%#v", err)
		os.Exit(1)
	}

	// Get currentWeek event
	urlStr = "https://fantasy.premierleague.com/api/bootstrap-static/"
	log.Debug.Printf("Fetch %s", urlStr)
	bytes, err := fetch(&client, urlStr)
	if err != nil {
		log.Critical.Printf("%#v", err)
		os.Exit(1)
	}
	var gameAPIResult GameAPIResult
	err = json.Unmarshal(bytes, &gameAPIResult)
	if err != nil {
		log.Critical.Printf("%#v", err)
	}

	currentWeek := 1
	for _, e := range gameAPIResult.Events {
		if e.IsCurrent == true {
			currentWeek = e.ID
			break
		}
	}

	for _, id := range cm.FplCfg.LeagueIDs {
		weeklyReportStr := ""
		finalReportStr := ""

		urlStr := fmt.Sprintf("https://fantasy.premierleague.com/api/leagues-classic/%s/standings/?page_new_entries=1&page_standings=1&phase=1", id)
		log.Debug.Printf("Fetch %s", urlStr)
		bytes, err := fetch(&client, urlStr)
		if err != nil {
			log.Critical.Printf("%#v", err)
			os.Exit(1)
		}
		var leagueAPIResult LeagueAPIResult
		err = json.Unmarshal(bytes, &leagueAPIResult)
		if err != nil {
			log.Critical.Printf("%#v", err)
			os.Exit(1)
		}

		var l League
		l.ID = leagueAPIResult.League.ID
		l.Name = leagueAPIResult.League.Name
		l.Entries = make([]Entry, 0)

		hasNext := leagueAPIResult.Standings.HasNext
		page := leagueAPIResult.Standings.Page + 1
		for hasNext == true {
			urlStr := fmt.Sprintf("https://fantasy.premierleague.com/api/leagues-classic/%s/standings/?page_new_entries=1&page_standings=%d&phase=1", id, page)
			log.Debug.Printf("Fetch %s", urlStr)
			bytes, err := fetch(&client, urlStr)
			if err != nil {
				log.Critical.Printf("%#v", err)
				os.Exit(1)
			}
			var _leagueAPIResult LeagueAPIResult
			err = json.Unmarshal(bytes, &_leagueAPIResult)
			if err != nil {
				log.Critical.Printf("%#v", err)
				os.Exit(1)
			}
			hasNext = _leagueAPIResult.Standings.HasNext
			page = _leagueAPIResult.Standings.Page + 1
		}

		var wg sync.WaitGroup
		var mutex = &sync.Mutex{}
		for _, r := range leagueAPIResult.Standings.Results {
			wg.Add(1)
			go func(id int, entry int, entryName string, playerName string, total int, rank int) {
				defer wg.Done()

				urlStr := fmt.Sprintf("https://fantasy.premierleague.com/api/entry/%d/history/", entry)
				log.Debug.Printf("Fetch %s", urlStr)
				bytes, err := fetch(&client, urlStr)
				if err != nil {
					log.Critical.Printf("%#v", err)
					os.Exit(1)
				}
				var entryAPIResult EntryAPIResult
				err = json.Unmarshal(bytes, &entryAPIResult)
				if err != nil {
					log.Critical.Printf("%#v", err)
					os.Exit(1)
				}

				ignored := false
				for _, e := range cm.FplCfg.IgnoreEntries {
					if entry == e {
						ignored = true
						break
					}
				}
				if ignored {
					return
				}

				var e Entry
				e.ID = id
				e.EntryNum = entry
				e.EntryName = entryName
				e.PlayerName = playerName
				e.Total = total
				e.Rank = rank
				e.Point = make(map[int]int)

				for _, c := range entryAPIResult.Current {
					e.Point[c.Event] = c.Points - c.EventTransfersCost
				}

				mutex.Lock()
				l.Entries = append(l.Entries, e)
				mutex.Unlock()
			}(r.ID, r.Entry, r.EntryName, r.PlayerName, r.Total, r.Rank)
		}
		wg.Wait()

		if len(l.Entries) == 0 {
			continue
		}

		weeklyReportStr = fmt.Sprintf("\n[%d] %s", l.ID, l.Name)
		finalReportStr = fmt.Sprintf("\n[%d] %s", l.ID, l.Name)

		weeklyReportStr = fmt.Sprintf("%s\nCurrent week: %d", weeklyReportStr, currentWeek)
		finalReportStr = fmt.Sprintf("%s\nCurrent week: %d", finalReportStr, currentWeek)

		// Find highest point of each week
		for w := 1; w <= currentWeek; w++ {
			weeklyReportStr = fmt.Sprintf("%s\n- Week %d", weeklyReportStr, w)
			highestPoint := 0
			for _, e := range l.Entries {
				p, ok := e.Point[w]
				if ok {
					if p > highestPoint {
						highestPoint = p
					}
				}
			}

			weeklyReportStr = fmt.Sprintf("%s\n\tHighest point: %d", weeklyReportStr, highestPoint)

			// Find entry whose point is equal to the highest point
			for _, e := range l.Entries {
				p, ok := e.Point[w]
				if ok {
					if p == highestPoint {
						weeklyReportStr = fmt.Sprintf("%s\n\t+ [%s] %s", weeklyReportStr, e.EntryName, e.PlayerName)
					}
				}
			}
		}

		for i := len(l.Entries); i > 0; i-- {
			//The inner loop will first iterate through the full length
			//the next iteration will be through n-1
			// the next will be through n-2 and so on
			for j := 1; j < i; j++ {
				if l.Entries[j-1].Total < l.Entries[j].Total {
					tmp := l.Entries[j]
					l.Entries[j] = l.Entries[j-1]
					l.Entries[j-1] = tmp
				}
			}
		}

		lastRank := -1
		lastPoint := -1
		for i := range l.Entries {
			if lastRank == -1 {
				l.Entries[i].Rank = i + 1
			} else {
				if l.Entries[i].Total == lastPoint {
					l.Entries[i].Rank = lastRank
				} else {
					l.Entries[i].Rank = i + 1
				}
			}

			lastRank = l.Entries[i].Rank
			lastPoint = l.Entries[i].Total
		}

		for i := range l.Entries {
			finalReportStr = fmt.Sprintf("%s\n+ Top %2d: %d: [%s] %s", finalReportStr, l.Entries[i].Rank, l.Entries[i].Total, l.Entries[i].EntryName, l.Entries[i].PlayerName)
		}

		log.Info.Printf("%s", weeklyReportStr)
		log.Info.Printf("%s", finalReportStr)

		if cm.AppCfg.DirectoryToExport != "" {
			err = os.MkdirAll(cm.AppCfg.DirectoryToExport, 0755)
			if err != nil {
				log.Critical.Printf("%+v", err)
			} else {
				var f *os.File

				f, err = os.Create(filepath.Join(cm.AppCfg.DirectoryToExport, fmt.Sprintf("%s-weekly.txt", id)))
				if err != nil {
					log.Critical.Printf("%+v", err)
				} else {
					_, err = f.Write([]byte(weeklyReportStr))
					if err != nil {
						log.Critical.Printf("%+v", err)
					}
					f.Close()
				}

				f, err = os.Create(filepath.Join(cm.AppCfg.DirectoryToExport, fmt.Sprintf("%s-final.txt", id)))
				if err != nil {
					log.Critical.Printf("%+v", err)
				} else {
					_, err = f.Write([]byte(finalReportStr))
					if err != nil {
						log.Critical.Printf("%+v", err)
					}
					f.Close()
				}
			}
		}
	}

	os.Exit(0)
}

func fetch(client *http.Client, urlStr string) ([]byte, error) {
	var reader io.ReadCloser

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return make([]byte, 0), err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/76.0.3809.100 Safari/537.36")

	res, err := client.Do(req)
	if err != nil {
		return make([]byte, 0), err
	}
	defer res.Body.Close()

	// Check that the server actual sent compressed data
	switch res.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(res.Body)
		if err != nil {
			return make([]byte, 0), err
		}
		defer reader.Close()
	default:
		reader = res.Body
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return make([]byte, 0), err
	}

	return body, nil
}
