// Copyright 2011 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package gophertv

import (
  "encoding/json"
  "fmt"
  "html/template"
  "io/ioutil"
  "log"
  "net/http"
  "strconv"
  "time"

  "code.google.com/p/google-api-go-client/youtube/v3"
  "github.com/gorilla/mux"

  "appengine"
  "appengine/datastore"
  "appengine/taskqueue"
)

func init() {
  r := mux.NewRouter().StrictSlash(false)

  // all API routes starts here
  r.HandleFunc("/", homePageHandler)
  r.HandleFunc("/t/{tag}", tagPageHandler)
  r.HandleFunc("/v/{id}", videoPageHandler)

  // curation pages
  curateRoutes := r.PathPrefix("/curate").Subrouter()
  curateRoutes.Path("/list").HandlerFunc(curateListHandler)
  curateRoutes.Path("/delete_all").Methods("POST").HandlerFunc(deleteAllHandler)
  // curateRoutes.Path("/video/{id}").HandlerFunc(curateVideoHandler)

  // videos collection
  videos := r.Path("/videos").Subrouter()
  videos.Methods("GET").HandlerFunc(VideoIndexHandler)

  // videos singular
  video := r.PathPrefix("/videos/{id}").Subrouter()
  // video.Methods("GET").Path("/edit").HandlerFunc(videoEditHandler)
  video.Methods("GET").HandlerFunc(videoShowHandler)
  video.Methods("PUT", "POST").HandlerFunc(videoUpdateHandler)
  video.Methods("DELETE").HandlerFunc(videoDeleteHandler)

  // Youtube crawler APIs

  // discover videos with a given query q
  ytCrawl := r.PathPrefix("/crawler/yt").Subrouter()
  ytCrawl.Path("/crawl").HandlerFunc(crawlVideos)
  ytCrawl.Path("/fetch_playlists").HandlerFunc(fetchPlaylistHandler)
  ytCrawl.Path("/fetch_videos").HandlerFunc(fetchVideosHandler)

  http.Handle("/", r)
}

func homePageHandler(w http.ResponseWriter, r *http.Request) {
  // TODO (jacob): replace with actual content
  fmt.Fprintf(w, "Home page .. coming soon")
}

func tagPageHandler(w http.ResponseWriter, r *http.Request) {
  // TODO (jacob): replace with actual content
  vars := mux.Vars(r)
  tag := vars["tag"]
  fmt.Fprintf(w, "videos for tag %v to be displayed here", tag)
}

func videoPageHandler(w http.ResponseWriter, r *http.Request) {
  // TODO (jacob): replace with actual content
  vars := mux.Vars(r)
  id := vars["id"]
  fmt.Fprintf(w, "video with %v to be displayed here", id)
}

func deleteAllHandler(w http.ResponseWriter, r *http.Request) {
  q := datastore.NewQuery("Video").Limit(400).KeysOnly()
  var videos []Video
  c := appengine.NewContext(r)
  keys, err := q.GetAll(c, &videos)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  err = datastore.DeleteMulti(c, keys)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  w.WriteHeader(http.StatusOK)
}
func VideoIndexHandler(w http.ResponseWriter, r *http.Request) {
  q := datastore.NewQuery("Video")
  tag := r.URL.Query().Get("tag")
  if tag != "" {
    q = q.Filter("Tags =", tag)
  }
  isCurated := r.URL.Query().Get("is_curated")
  if isCurated == "" {
    // by default, return curated videos
    q = q.Filter("IsCurated =", true)
  } else {
    q = q.Filter("IsCurated =", false)
  }
  order := r.URL.Query().Get("order")
  if order == "" {
    q = q.Order("-ViewCount")
  } else {
    q = q.Order(order)
  }
  limit := r.URL.Query().Get("limit")
  if limit != "" {
    n, err := strconv.ParseInt(limit, 10, 64)
    if err != nil {
      http.Error(w, "internal server error", http.StatusInternalServerError)
      return
    }
    q = q.Limit(int(n))
  }
  offset := r.URL.Query().Get("offset")
  if offset != "" {
    n, err := strconv.ParseInt(offset, 10, 64)
    if err != nil {
      http.Error(w, "internal server error", http.StatusInternalServerError)
      return
    }
    q = q.Offset(int(n))
  }
  c := appengine.NewContext(r)
  var videos []Video
  _, err := q.GetAll(c, &videos)
  if err != nil {
    http.Error(w, "internal server error", http.StatusInternalServerError)
    return
  }

  if len(videos) == 0 {
    videos = []Video{}
  }

  jsn, err := json.Marshal(&videos)
  if err != nil {
    http.Error(w, "internal server error", http.StatusInternalServerError)
    return
  }
  w.Header().Set("Content-Type", "application/json")
  w.Write(jsn)
}

func curateListHandler(w http.ResponseWriter, r *http.Request) {
  q := datastore.NewQuery("Video").
    Filter("IsCurated =", false).Order("-ViewCount").Limit(10)

  var n int64
  var err error
  offset := r.URL.Query().Get("offset")
  if offset != "" {
    n, err = strconv.ParseInt(offset, 10, 64)
    if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }
    log.Printf("current offset: %d", n)
    q = q.Offset(int(n))
  }
  c := appengine.NewContext(r)
  var videos []Video
  _, err = q.GetAll(c, &videos)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  t, err := template.ParseFiles("public/templates/curation/list.html")
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  log.Printf("executing the template, new offset: n: %d len: %d", n, len(videos))
  err = t.Execute(w, struct {
    Videos []Video
    Offset int
  }{
    Videos: videos,
    Offset: int(n) + len(videos),
  })
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
}

// Video resource REST functions

// get Handler
func videoShowHandler(w http.ResponseWriter, r *http.Request) {
  // TODO (sunil): pick it from the URL
  vars := mux.Vars(r)
  id := vars["id"]
  var video Video
  c := appengine.NewContext(r)
  key := datastore.NewKey(c, "Video", id, 0, nil)
  err := datastore.Get(c, key, &video)
  if err != nil {
    if err == datastore.ErrNoSuchEntity {
      http.Error(w, "video not found", http.StatusNotFound)
      return
    }
    http.Error(w, "internal server error", http.StatusInternalServerError)
    return
  }
  jsn, err := json.Marshal(&video)
  if err != nil {
    http.Error(w, "internal server error", http.StatusInternalServerError)
    return
  }
  w.Header().Set("Content-Type", "application/json")
  w.Write(jsn)
}

// videoDeleteHandler deletes a video.
func videoDeleteHandler(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  id := vars["id"]
  c := appengine.NewContext(r)
  key := datastore.NewKey(c, "Video", id, 0, nil)
  err := datastore.Delete(c, key)
  if err != nil {
    http.Error(w, "internal server error", http.StatusInternalServerError)
    return
  }
  w.WriteHeader(http.StatusOK)
}

type VideoUpdateRequest struct {
  ID        string   `json:"id"`
  Tags      []string `json:"tags"`
  IsCurated bool     `json:"is_curated"`
}

func videoUpdateHandler(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  id := vars["id"]
  body, err := ioutil.ReadAll(r.Body)
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  updateReq := VideoUpdateRequest{}
  err = json.Unmarshal(body, &updateReq)
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }

  if updateReq.ID == "" || len(updateReq.Tags) == 0 || id != updateReq.ID {
    http.Error(w, "invalid request", http.StatusBadRequest)
    return
  }

  var video Video
  c := appengine.NewContext(r)
  key := datastore.NewKey(c, "Video", updateReq.ID, 0, nil)
  err = datastore.Get(c, key, &video)
  if err != nil {
    if err == datastore.ErrNoSuchEntity {
      http.Error(w, "video not found", http.StatusNotFound)
      return
    }
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  video.Tags = updateReq.Tags
  video.IsCurated = true
  _, err = datastore.Put(c, key, &video)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  w.WriteHeader(http.StatusOK)
}

func crawlVideos(w http.ResponseWriter, r *http.Request) {
  query := r.URL.Query().Get("q")
  if query == "" {
    log.Printf("query missing in crawl request")
    http.Error(w, "query missing", http.StatusBadRequest)
    return
  }
  c := appengine.NewContext(r)

  playlists, err := searchPlaylists(c, query)
  if err != nil {
    log.Printf("error in searching playlist for query: %s :: %v", query, err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  log.Printf("fetch %d playlists for query: %s", len(playlists), query)
  for _, pl := range playlists {
    log.Printf("adding task for fetching video for playlist: %s", pl)
    task := taskqueue.NewPOSTTask("/crawler/yt/fetch_playlists", nil)
    payload, _ := json.Marshal(&FetchRequest{IDs: []string{pl}})
    task.Payload = payload
    _, err = taskqueue.Add(c, task, "")
    if err != nil {
      log.Printf("error scheduling tasks for fetching videos for pl: %s :: %v",
        pl, err)
      continue
    }
  }
  log.Printf("jobs scheduled for fetching videos for %d playlists for query: %s",
    len(playlists), query)
  fmt.Fprintf(w, "job scheduled")
}

func fetchPlaylistHandler(w http.ResponseWriter, r *http.Request) {
  fetchReq, err := readFetchRequest(r)
  if err != nil {
    log.Printf("error parsing fetch request :: %v", err)
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  log.Printf("new playlist fetch request received : %v", fetchReq.IDs)
  c := appengine.NewContext(r)
  for _, pl := range fetchReq.IDs {
    log.Printf("fetching video IDs for playlist: %s", pl)
    videosIDs, err := getVideoIDs(c, pl)
    if err != nil {
      log.Printf("error fetching video IDs for playlist: %s", pl)
      continue
    }
    for _, vid := range videosIDs {
      //schedule tasks
      task := taskqueue.NewPOSTTask("/crawler/yt/fetch_videos", nil)
      payload, _ := json.Marshal(&FetchRequest{IDs: []string{vid}})
      task.Payload = payload
      _, err = taskqueue.Add(c, task, "")
      if err != nil {
        log.Printf("error scheduling tasks for fetching video %s for pl: %s :: %v",
          vid, pl, err)
        continue
      }
    }
  }
  fmt.Fprintf(w, "job scheduled")
}

type FetchRequest struct {
  Kind string   `json:"kind"`
  IDs  []string `json:"ids"`
}

// readFetchRequest parses incoming fetch request and returns an instance of
// FetchRequest.
func readFetchRequest(r *http.Request) (*FetchRequest, error) {
  body, err := ioutil.ReadAll(r.Body)
  if err != nil {
    return nil, err
  }
  fr := FetchRequest{}
  err = json.Unmarshal(body, &fr)
  if err != nil {
    return nil, err
  }
  return &fr, nil
}

func fetchVideosHandler(w http.ResponseWriter, r *http.Request) {
  fetchReq, err := readFetchRequest(r)
  if err != nil {
    log.Printf("error parsing fetch request :: %v", err)
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  c := appengine.NewContext(r)
  videos, err := fetchVideos(c, fetchReq.IDs...)
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
    return
  }
  printVideos(videos...)
  SaveVideos(c, videos...)
  fmt.Fprintf(w, "fetched %d videos", len(videos))
}

const (
  VideoSourceYoutube = "yt"
  VideoSourceVimeo   = "vimeo"
)

type Video struct {
  ID                string    `json:"id"`
  VideoSource       string    `json:"video_source"`
  IsCurated         bool      `json:"is_curated"`
  PublishedAt       time.Time `json:"published_at"`
  CreatedAt         time.Time `json:"created_at"`
  Title             string    `datastore:",noindex",json:"title"`
  Desc              string    `datastore:",noindex",json:"desc"`
  Tags              []string  `json:"tags"`
  CommentCount      int64     `json:"comment_count"`
  DislikeCount      int64     `json:"dislike_count"`
  FavoriteCount     int64     `json:"favorite_count"`
  LikeCount         int64     `json:"like_count"`
  ViewCount         int64     `json:"view_count"`
  DefaultThumbnail  Thumbnail `json:"default_thumbnail"`
  HighresThumbnail  Thumbnail `json:"highres_thumbnail"`
  MaxresThumbnail   Thumbnail `json:"maxres_thumbnail"`
  MediumThumbnail   Thumbnail `json:"medium_thumbnail"`
  StandardThumbnail Thumbnail `json:"standard_thumbnail"`
  ContentDuration   string    `json:"content_duration"`
  ContentDimension  string    `json:"content_dimension"`
  ContentDefinition string    `json:"content_definition"`
  ChannelID         string    `json:"channel_id"`
  ChannelTitle      string    `json:"channel_title"`
}

type Thumbnail struct {
  URL    string
  Width  int64
  Height int64
}

func NewThumbnail(t *youtube.Thumbnail) Thumbnail {
  return Thumbnail{
    URL:    t.Url,
    Width:  t.Width,
    Height: t.Height,
  }
}

func SaveVideos(c appengine.Context, videos ...*youtube.Video) error {
  for i, v := range videos {
    log.Printf("saving video %d", i)
    err := SaveVideo(c, v)
    if err != nil {
      log.Printf("error in saving video: %v :: %v", v.Id, err)
    }
  }
  return nil
}

func SaveVideo(c appengine.Context, video *youtube.Video) error {
  var videoInDB Video
  videoKey := datastore.NewKey(c, "Video", video.Id, 0, nil)
  err := datastore.Get(c, videoKey, &videoInDB)
  if err == nil {
    // video already exist, so ignore
    return nil
  }
  if err != datastore.ErrNoSuchEntity {
    // some error occured
    return err
  }

  // video doesn't exist, so lets save it
  // lets save the video now
  publishedAt, err := time.Parse(time.RFC3339, video.Snippet.PublishedAt)
  if err != nil {
    log.Printf("error parsing publishedAt field for the video :: %v", err)
    publishedAt = time.Now().UTC()
  }
  videoToSave := Video{
    VideoSource:       VideoSourceYoutube,
    ID:                video.Id,
    IsCurated:         false,
    Title:             video.Snippet.Title,
    Desc:              video.Snippet.Description,
    Tags:              []string{""},
    CommentCount:      int64(video.Statistics.CommentCount),
    DislikeCount:      int64(video.Statistics.DislikeCount),
    FavoriteCount:     int64(video.Statistics.FavoriteCount),
    LikeCount:         int64(video.Statistics.LikeCount),
    ViewCount:         int64(video.Statistics.ViewCount),
    ContentDuration:   video.ContentDetails.Duration,
    ContentDimension:  video.ContentDetails.Dimension,
    ContentDefinition: video.ContentDetails.Definition,
    ChannelID:         video.Snippet.ChannelId,
    ChannelTitle:      video.Snippet.ChannelTitle,
    CreatedAt:         time.Now().UTC(),
    PublishedAt:       publishedAt,
  }
  thumbnails := video.Snippet.Thumbnails
  if thumbnails != nil {
    if thumbnails.Default != nil {
      videoToSave.DefaultThumbnail = NewThumbnail(thumbnails.Default)
    }
    if thumbnails.High != nil {
      videoToSave.HighresThumbnail = NewThumbnail(thumbnails.High)
    }
    if thumbnails.Medium != nil {
      videoToSave.MediumThumbnail = NewThumbnail(thumbnails.Medium)
    }
    if thumbnails.Maxres != nil {
      videoToSave.MaxresThumbnail = NewThumbnail(thumbnails.Maxres)
    }

    if thumbnails.Standard != nil {
      videoToSave.StandardThumbnail = NewThumbnail(thumbnails.Standard)
    }
  }
  _, err = datastore.Put(c, videoKey, &videoToSave)
  if err != nil {
    log.Printf("error saving video :: %v", err)
  }
  return err
}
