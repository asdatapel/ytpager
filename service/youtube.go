package service

import (
	"asdatapel/ytpager/model"
	"context"
	"fmt"
	"log"
	"net/http"

	"google.golang.org/api/youtube/v3"
)

type Youtube struct {
	ytservice *youtube.Service
}

func NewYoutube(ctx context.Context, client *http.Client) Youtube {
	ytservice, err := youtube.New(client)
	if err != nil {
		log.Fatal(err)
	}

	return Youtube{ytservice}
}

func (yt Youtube) GetChannel(name string) model.Channel {
	channelRequest := yt.ytservice.Channels.List("id, snippet, contentDetails").ForUsername(name)
	channelResponse, err := channelRequest.Do()
	if err != nil {
		log.Fatalf("30 %v", err)
	}

	firstChannel := channelResponse.Items[0]
	playlistId := firstChannel.ContentDetails.RelatedPlaylists.Uploads

	playlistRequest := yt.ytservice.Playlists.List("id, contentDetails").Id(playlistId)
	playlistResponse, err := playlistRequest.Do()
	if err != nil {
		log.Fatalf("39 %v", err)
	}

	return model.Channel{
		Id:              firstChannel.Id,
		Name:            firstChannel.Snippet.Title,
		UploadsPlaylist: playlistId,
		NumVideos:       playlistResponse.Items[0].ContentDetails.ItemCount,
		NumPages:        playlistResponse.Items[0].ContentDetails.ItemCount/20 + 1,
	}
}

func (yt Youtube) ListVideos(channel *model.Channel, page int64) ([]model.Video, string) {
	itemsRequest := yt.ytservice.PlaylistItems.List("id,snippet,contentDetails").
		PlaylistId(channel.UploadsPlaylist).MaxResults(20)
	itemsResponse, err := itemsRequest.Do()
	if err != nil {
		log.Fatalf("56 %v", err)
	}

	var currentPage int64 = 0
	for currentPage < page {
		itemsRequest := yt.ytservice.PlaylistItems.List("id,snippet,contentDetails").
			PlaylistId(channel.UploadsPlaylist).MaxResults(20).PageToken(itemsResponse.NextPageToken)
		itemsResponse, err = itemsRequest.Do()
		if err != nil {
			log.Fatalf("56 %v", err)
		}

		currentPage += 1
	}

	Youtube := make([]model.Video, len(itemsResponse.Items))
	for i, item := range itemsResponse.Items {
		Youtube[i] = model.Video{
			Id:           item.ContentDetails.VideoId,
			Title:        item.Snippet.Title,
			PublishedAt:  item.Snippet.PublishedAt,
			ThumbnailUrl: item.Snippet.Thumbnails.Medium.Url,
		}
	}
	return Youtube, ""
}

func getWeights(bases []int64, number int64) []int64 {
	n := number
	var weights []int64
	for _, item := range bases {
		q := n / item
		n = n % item
		weights = append(weights, q)
	}
	return weights
}

func getPageToken(page int64) string {
	fmt.Println("asdasd")
	fmt.Println(page)
	b64 := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	x := getWeights([]int64{65536, 16384, 8192, 128, 16, 1}, page)
	weight_1 := x[5]
	weight_16 := x[4]
	weight_128 := x[3]
	weight_8192 := x[2]
	weight_16384 := x[1]
	weight_65536 := x[0]

	var w16_offset int64
	if page < 128 {
		w16_offset = 0
	} else {
		w16_offset = 8
	}
	w1_offset := (weight_8192 % 2) + 2

	// converts range(0, 3) into 'BRhx'
	suffix_pos := (weight_16384*16 + 1) % 64
	char_16384 := string(b64[suffix_pos])

	if page < 16384 {
		char_16384 = "E"
		w1_offset = 1
	}
	if page < 8192 {
		w1_offset = 0
	}
	if page < 128 {
		char_16384 = "Q"
	}

	char_1 := string(b64[weight_1*4+w1_offset])  // converts range(0, 15) into  'AEIMQUYcgkosw048', 'BFJNRVZdhlptx159', 'CGKOSWaeimquy26-', 'DHLPTXbfjnrvz37_'
	char_16 := string(b64[weight_16+w16_offset]) // converts range(0 ,7) into 'ABCDEFGH' and 'IJKLMNOP'

	var char_128 string
	if page >= 128 {
		char_128 = string(b64[weight_128])
	} else {
		char_128 = ""
	}
	var char_65536 string
	if page >= 16384 {
		char_65536 = string(b64[weight_65536])
	} else {
		char_65536 = ""
	}

	return fmt.Sprintf("%s%s%s%s%s%s%s", "C", char_16, char_1, char_128, char_65536, char_16384, "AA")
}
