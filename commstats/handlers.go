package commstats

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	cache "github.com/planetdecred/pdanalytics/chart"
	"github.com/planetdecred/pdanalytics/postgres/models"
	"github.com/planetdecred/pdanalytics/web"
)

const (
	redditPlatform  = "Reddit"
	twitterPlatform = "Twitter"
	githubPlatform  = "GitHub"
	youtubePlatform = "YouTube"
)

var (
	commStatPlatforms = []string{redditPlatform, twitterPlatform, githubPlatform, youtubePlatform}
)

// /community
func (s *Collector) community(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	pageStr := r.FormValue("page")
	viewOption := r.FormValue("view-option")
	selectedNumStr := r.FormValue("records-per-page")
	platform := r.FormValue("platform")
	subreddit := r.FormValue("subreddit")
	dataType := r.FormValue("data-type")
	twitterHandle := r.FormValue("twitter-handle")
	repository := r.FormValue("repository")
	channel := r.FormValue("channel")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	if viewOption == "" {
		viewOption = "chart"
	}

	if platform == "" {
		platform = commStatPlatforms[0]
	}

	if subreddit == "" && len(Subreddits()) > 0 {
		subreddit = Subreddits()[0]
	}

	if twitterHandle == "" && len(TwitterHandles()) > 0 {
		twitterHandle = TwitterHandles()[0]
	}

	if repository == "" && len(Repositories()) > 0 {
		repository = Repositories()[0]
	}

	if channel == "" && len(YoutubeChannels()) > 0 {
		channel = YoutubeChannels()[0]
	}

	selectedNum, _ := strconv.Atoi(selectedNumStr)
	if selectedNum == 0 {
		selectedNum = 20
	}

	var previousPage, nextPage int
	if page > 1 {
		previousPage = page - 1
	} else {
		previousPage = 1
	}

	nextPage = page + 1

	data := map[string]interface{}{
		"page":             page,
		"viewOption":       viewOption,
		"platforms":        commStatPlatforms,
		"platform":         platform,
		"subreddits":       Subreddits(),
		"subreddit":        subreddit,
		"twitterHandles":   TwitterHandles(),
		"twitterHandle":    twitterHandle,
		"repositories":     Repositories(),
		"repository":       repository,
		"channels":         YoutubeChannels(),
		"channel":          channel,
		"dataType":         dataType,
		"currentPage":      page,
		"pageSizeSelector": web.PageSizeSelector,
		"selectedNum":      selectedNum,
		"previousPage":     previousPage,
		"nextPage":         nextPage,
	}

	str, err := s.server.Templates.ExecTemplateToString("community", struct {
		*web.CommonPageData
		BreadcrumbItems []web.BreadcrumbItem
		Data            map[string]interface{}
	}{
		CommonPageData: s.server.CommonData(r),
		BreadcrumbItems: []web.BreadcrumbItem{
			{
				HyperText: "Historic exchange rate data",
				Active:    true,
			},
		},
		Data: data,
	})

	if err != nil {
		log.Errorf("Template execute failure: %v", err)
		s.server.StatusPage(w, r, web.DefaultErrorCode, web.DefaultErrorMessage, err.Error(), web.ExpStatusError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err = io.WriteString(w, str); err != nil {
		log.Error(err)
	}
}

// getCommunityStat
func (s *Collector) getCommunityStat(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	plarform := r.FormValue("platform")
	pageStr := r.FormValue("page")
	pageSizeStr := r.FormValue("records-per-page")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(pageSizeStr)
	if pageSize < 1 {
		pageSize = 20
	}

	var stats interface{}
	var columnHeaders []string
	var totalCount int64
	var err error

	offset := (page - 1) * pageSize

	switch plarform {
	case redditPlatform:
		subreddit := r.FormValue("subreddit")
		stats, err = s.db.RedditStats(r.Context(), subreddit, offset, pageSize)
		if err != nil {
			web.RenderErrorfJSON(w, "cannot fetch Reddit stat, %s", err.Error())
			return
		}

		totalCount, err = s.dataStore.CountRedditStat(r.Context(), subreddit)
		if err != nil {
			web.RenderErrorfJSON(w, "cannot fetch Reddit stat, %s", err.Error())
			return
		}

		columnHeaders = append(columnHeaders, "Date", "Subscribers", "Accounts Active")
	case twitterPlatform:
		handle := r.FormValue("twitter-handle")
		stats, err = s.dataStore.TwitterStats(r.Context(), handle, offset, pageSize)
		if err != nil {
			web.RenderErrorfJSON(w, "cannot fetch Twitter stat, %s", err.Error())
			return
		}

		totalCount, err = s.dataStore.CountTwitterStat(r.Context(), handle)
		if err != nil {
			web.RenderErrorfJSON(w, "cannot fetch Twitter stat, %s", err.Error())
			return
		}

		columnHeaders = append(columnHeaders, "Date", "Followers")
	case githubPlatform:
		repository := r.FormValue("repository")
		stats, err = s.dataStore.GithubStat(r.Context(), repository, offset, pageSize)
		if err != nil {
			web.RenderErrorfJSON(w, "cannot fetch Github stat, %s", err.Error())
			return
		}

		totalCount, err = s.dataStore.CountGithubStat(r.Context(), repository)
		if err != nil {
			web.RenderErrorfJSON(w, "cannot fetch Github stat, %s", err.Error())
			return
		}

		columnHeaders = append(columnHeaders, "Date", "Stars", "Forks")
	case youtubePlatform:
		channel := r.FormValue("channel")
		stats, err = s.db.YoutubeStat(r.Context(), channel, offset, pageSize)
		if err != nil {
			web.RenderErrorfJSON(w, fmt.Sprintf("cannot fetch Youtbue stat, %s", err.Error())
			return
		}

		totalCount, err = s.dataStore.CountYoutubeStat(r.Context(), channel)
		if err != nil {
			web.RenderErrorfJSON(w, "cannot fetch Youtbue stat, %s", err.Error())
			return
		}

		columnHeaders = append(columnHeaders, "Date", "Subscribers", "View Count")
	}

	totalPages := totalCount / int64(pageSize)
	if totalCount > totalPages*int64(pageSize) {
		totalPages += 1
	}

	web.RenderJSON(w, map[string]interface{}{
		"stats":       stats,
		"columns":     columnHeaders,
		"total":       totalCount,
		"totalPages":  totalPages,
		"currentPage": page,
	})
}

// /communitychat
func (s *Collector) communityChat(resp http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	platform := req.FormValue("platform")
	dataType := req.FormValue("data-type")

	filters := map[string]string{}
	yLabel := ""
	switch platform {
	case githubPlatform:
		if dataType == models.GithubColumns.Folks {
			yLabel = "Forks"
		} else {
			yLabel = "Stars"
		}
		platform = models.TableNames.Github
		filters[models.GithubColumns.Repository] = fmt.Sprintf("'%s'", req.FormValue("repository"))
	case twitterPlatform:
		yLabel = "Followers"
		dataType = models.TwitterColumns.Followers
		platform = models.TableNames.Twitter
	case redditPlatform:
		if dataType == models.RedditColumns.ActiveAccounts {
			yLabel = "Active Accounts"
		} else if dataType == models.RedditColumns.Subscribers {
			yLabel = "Subscribers"
		}
		platform = models.TableNames.Reddit
		filters[models.RedditColumns.Subreddit] = fmt.Sprintf("'%s'", req.FormValue("subreddit"))
	case youtubePlatform:
		platform = models.TableNames.Youtube
		if dataType == models.YoutubeColumns.ViewCount {
			yLabel = "View Count"
		} else if dataType == models.YoutubeColumns.Subscribers {
			yLabel = "Subscribers"
		}
		filters[models.YoutubeColumns.Channel] = fmt.Sprintf("'%s'", req.FormValue("channel"))
	}

	if dataType == "" {
		s.renderErrorJSON("Data type cannot be empty", resp)
		return
	}

	data, err := s.db.CommunityChart(req.Context(), platform, dataType, filters)
	if err != nil {
		s.renderErrorJSON(fmt.Sprintf("Cannot fetch chart data, %s", err.Error()), resp)
		return
	}
	var dates, records cache.ChartUints
	for _, record := range data {
		dates = append(dates, uint64(record.Date.Unix()))
		records = append(records, uint64(record.Record))
	}

	s.renderJSON(map[string]interface{}{
		"x":      dates,
		"y":      records,
		"ylabel": yLabel,
	}, resp)
}
