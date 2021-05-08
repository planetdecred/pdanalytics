// Copyright (c) 2018-2019 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package commstats

import (
	"context"
	"net/http"
	"time"

	"github.com/planetdecred/pdanalytics/web"
)

type CommStat struct {
	Date               time.Time         `json:"date"`
	RedditStats        map[string]Reddit `json:"reddit_stats"`
	TwitterFollowers   int               `json:"twitter_followers"`
	YoutubeSubscribers int               `json:"youtube_subscribers"`
	GithubStars        int               `json:"github_stars"`
	GithubFolks        int               `json:"github_folks"`
}

type RedditResponse struct {
	Kind string `json:"kind"`
	Data Reddit `json:"data"`
}

type Reddit struct {
	Date           time.Time `json:"date"`
	Subscribers    int       `json:"subscribers"`
	AccountsActive int       `json:"active_user_count"`
	Subreddit      string    `json:"subreddit"`
}

type Github struct {
	Date       time.Time `json:"date"`
	Stars      int       `json:"stars"`
	Folks      int       `json:"folks"`
	Repository string    `json:"repository"`
}

type Youtube struct {
	Date        time.Time `json:"date"`
	Subscribers int       `json:"subscribers"`
	Channel     string    `json:"channel"`
	ViewCount   int       `json:"view_count"`
}

type Twitter struct {
	Date      time.Time `json:"date"`
	Followers int       `json:"followers"`
	Handle    string    `json:"handle"`
}

type ChartData struct {
	Date   time.Time `json:"date"`
	Record int64     `json:"record"`
}

type DataStore interface {
	StoreRedditStat(context.Context, Reddit) error
	LastCommStatEntry() (time time.Time)
	StoreTwitterStat(ctx context.Context, twitter Twitter) error
	StoreYoutubeStat(ctx context.Context, youtube Youtube) error
	StoreGithubStat(ctx context.Context, github Github) error

	LastEntry(ctx context.Context, tableName string, receiver interface{}) error
	CountRedditStat(ctx context.Context, subreddit string) (int64, error)
	RedditStats(ctx context.Context, subreddit string, offset int, limit int) ([]Reddit, error)
	CountTwitterStat(ctx context.Context, handle string) (int64, error)
	TwitterStats(ctx context.Context, handle string, offset int, limit int) ([]Twitter, error)
	CountYoutubeStat(ctx context.Context, channel string) (int64, error)
	YoutubeStat(ctx context.Context, channel string, offset int, limit int) ([]Youtube, error)
	CountGithubStat(ctx context.Context, repository string) (int64, error)
	GithubStat(ctx context.Context, repository string, offset int, limit int) ([]Github, error)
	CommunityChart(ctx context.Context, platform string, dataType string, filters map[string]string) ([]ChartData, error)
}

type Collector struct {
	client    http.Client
	server    *web.Server
	dataStore DataStore
	options   *CommunityStatOptions
}

type CommunityStatOptions struct {
	// Community stat
	CommunityStat       bool     `long:"commstat" description:"Disable/Enable the periodic community stat collection"`
	CommunityStatHttp   bool     `long:"commstat-http" description:"Disable/Enable the http endpoints for community stat"`
	RedditStatInterval  int      `long:"redditstatinterval" description:"Collection interval for Reddit community stat"`
	Subreddit           []string `long:"subreddit" description:"List of subreddit for community stat collection"`
	TwitterHandles      []string `long:"twitterhandle" description:"List of twitter handles community stat collection"`
	TwitterStatInterval int      `long:"twitterstatinterval" description:"Number of minutes between Twitter stat collection"`
	GithubRepositories  []string `long:"githubrepository" description:"List of Github repositories to track"`
	GithubStatInterval  int      `long:"githubstatinterval" description:"Number of minutes between Github stat collection"`
	YoutubeChannelName  []string `long:"youtubechannelname" description:"List of Youtube channel names to be tracked"`
	YoutubeChannelId    []string `long:"youtubechannelid" description:"List of Youtube channel ID to be tracked"`
	YoutubeStatInterval int      `long:"youtubestatinterval" description:"Number of minutes between Youtube stat collection"`
	YoutubeDataApiKey   string   `long:"youtubedataapikey" description:"Youtube data API key gotten from google developer console"`
}
