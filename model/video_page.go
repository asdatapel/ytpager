package model

//VideoPage ...
type VideoPage struct {
	Index   int64
	Videos  []Video
	Channel *Channel
}
