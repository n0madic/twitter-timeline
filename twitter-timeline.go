// Package twittertimeline provides a client for accessing Twitter/X API without authorization
package twittertimeline

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Constants for Twitter API
const (
	BearerToken = "AAAAAAAAAAAAAAAAAAAAAFQODgEAAAAAVHTp76lzh3rFzcHbmHVvQxYYpTw%3DckAlMINMjmCwxUcaXbAN4XqJVdgMJaHqNOFgPMK0zN1qLqLQCF"
	BaseURL     = "https://api.x.com"
	UserAgent   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"

	// GraphQL API endpoints
	UserByScreenNamePath = "/graphql/x3RLKWW1Tl7JgU7YtGxuzw/UserByScreenName"
	UserTweetsPath       = "/graphql/bbmwRjH_roUoWsvbgAJY9g/UserTweets"
)

// Public API structures
type Tweet struct {
	// Basic information
	ID           string // RestID
	Text         string // FullText
	HTML         string // HTML version with links
	CreatedAt    string // Creation date
	PermanentURL string // Permanent link to tweet

	// Author
	Username string // Username (@username)
	UserID   string // User ID

	// Statistics (top level)
	Likes    int // FavoriteCount
	Retweets int // RetweetCount
	Replies  int // ReplyCount

	// Tweet types (boolean flags as is)
	IsPinned  bool // Whether tweet is pinned
	IsRetweet bool // Retweet
	IsQuoted  bool // Quote
	IsReply   bool // Reply

	// Media and links
	Images   []string // Image URLs
	Hashtags []string // Hashtags (text only)
	URLs     []URL    // Links
	Mentions []string // User mentions (username only)
}

type URL struct {
	Short    string // t.co ссылка
	Expanded string // Полная ссылка
	Display  string // Отображаемый текст
}

// Structures for parsing JSON responses
type GuestTokenResponse struct {
	GuestToken string `json:"guest_token"`
}

type UserResponse struct {
	Data struct {
		User struct {
			Result struct {
				RestID string `json:"rest_id"`
				ID     string `json:"id"`
				Legacy struct {
					UserInfo `json:"legacy"`
				} `json:"legacy"`
				Core struct {
					Name       string `json:"name"`
					ScreenName string `json:"screen_name"`
				} `json:"core"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"errors"`
}

type UserInfo struct {
	Name           string `json:"name"`
	ScreenName     string `json:"screen_name"`
	Description    string `json:"description"`
	FollowersCount int    `json:"followers_count"`
	FriendsCount   int    `json:"friends_count"`
	StatusesCount  int    `json:"statuses_count"`
}

type MediaEntity struct {
	MediaURLHTTPS string `json:"media_url_https"`
	Type          string `json:"type"`
}

type TweetResult struct {
	RestID string `json:"rest_id"`
	Core   struct {
		UserResults struct {
			Result struct {
				Core struct {
					ScreenName string `json:"screen_name"`
				} `json:"core"`
			} `json:"result"`
		} `json:"user_results"`
	} `json:"core"`
	Legacy struct {
		FullText             string `json:"full_text"`
		CreatedAt            string `json:"created_at"`
		UserIDStr            string `json:"user_id_str"`
		InReplyToStatusIDStr string `json:"in_reply_to_status_id_str"`
		InReplyToUserIDStr   string `json:"in_reply_to_user_id_str"`
		InReplyToScreenName  string `json:"in_reply_to_screen_name"`
		IsQuoteStatus        bool   `json:"is_quote_status"`
		QuotedStatusIDStr    string `json:"quoted_status_id_str"`
		RetweetedStatusIDStr string `json:"retweeted_status_id_str"`
		Entities             struct {
			Hashtags []struct {
				Text string `json:"text"`
			} `json:"hashtags"`
			Urls []struct {
				URL         string `json:"url"`
				ExpandedURL string `json:"expanded_url"`
				DisplayURL  string `json:"display_url"`
			} `json:"urls"`
			Media []MediaEntity `json:"media"`
		} `json:"entities"`
		ExtendedEntities struct {
			Media []MediaEntity `json:"media"`
		} `json:"extended_entities"`
		FavoriteCount int `json:"favorite_count"`
		RetweetCount  int `json:"retweet_count"`
		ReplyCount    int `json:"reply_count"`
	} `json:"legacy"`
	RetweetedStatusResult struct {
		Result *TweetResult `json:"result"`
	} `json:"retweeted_status_result"`
	IsPinned  bool     `json:"-"` // Not from JSON, set by code
	IsRetweet bool     `json:"-"` // Not from JSON, determined by code
	IsQuoted  bool     `json:"-"` // Not from JSON, determined by code
	IsReply   bool     `json:"-"` // Not from JSON, determined by code
	Images    []string `json:"-"` // Not from JSON, extracted from media
	URL       string   `json:"-"` // Not from JSON, permanent URL to tweet
	HTML      string   `json:"-"` // Not from JSON, HTML formatted content
}

type TimelineEntry struct {
	EntryID string `json:"entryId"`
	Content struct {
		EntryType   string `json:"entryType"`
		ItemContent *struct {
			TweetResults struct {
				Result TweetResult `json:"result"`
			} `json:"tweet_results"`
		} `json:"itemContent"`
		Items *[]struct {
			EntryID string `json:"entryId"`
			Item    struct {
				ItemContent struct {
					TweetResults struct {
						Result TweetResult `json:"result"`
					} `json:"tweet_results"`
				} `json:"itemContent"`
			} `json:"item"`
		} `json:"items"`
	} `json:"content"`
}

type TimelineResponse struct {
	Data struct {
		User struct {
			Result struct {
				Timeline struct {
					Timeline struct {
						Instructions []struct {
							Type    string          `json:"type"`
							Entries []TimelineEntry `json:"entries"`
							Entry   *TimelineEntry  `json:"entry"`
						} `json:"instructions"`
					} `json:"timeline"`
				} `json:"timeline"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
}

// userIDCacheEntry represents a cached user ID entry
type userIDCacheEntry struct {
	UserID    string
	Timestamp time.Time
}

// Client represents a client for working with Twitter API
type Client struct {
	httpClient  *http.Client
	guestToken  string
	bearerToken string
	cacheTTL    time.Duration
}

// Global cache for user IDs to avoid repeated API calls
var userIDCache sync.Map

// NewClient creates a new Twitter client
func NewClient() *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		bearerToken: BearerToken,
		cacheTTL:    24 * time.Hour, // Cache for 24 hours
	}

	// Start cache cleanup goroutine
	go client.cleanupCache()

	return client
}

// cleanupCache periodically removes expired entries from the cache
func (c *Client) cleanupCache() {
	ticker := time.NewTicker(time.Hour) // Run cleanup every hour
	defer ticker.Stop()

	for range ticker.C {
		userIDCache.Range(func(key, value any) bool {
			entry := value.(*userIDCacheEntry)
			if time.Since(entry.Timestamp) > c.cacheTTL {
				userIDCache.Delete(key)
			}
			return true
		})
	}
}

// GetGuestToken gets guest token from Twitter API
func (c *Client) GetGuestToken() error {
	req, err := http.NewRequest("POST", BaseURL+"/1.1/guest/activate.json", nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected response status: %d, body: %s", resp.StatusCode, string(body))
	}

	var tokenResp GuestTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	c.guestToken = tokenResp.GuestToken

	// Reset cookie jar to start fresh with new guest token
	if jar, err := cookiejar.New(nil); err == nil {
		c.httpClient.Jar = jar
	}

	return nil
}

// makeAPICall makes a universal GraphQL API call to Twitter/X
func (c *Client) makeAPICall(endpoint string, variables map[string]any, features map[string]any, fieldToggles map[string]any) (*http.Response, error) {
	if c.guestToken == "" {
		if err := c.GetGuestToken(); err != nil {
			return nil, fmt.Errorf("error getting guest token: %w", err)
		}
	}

	variablesJSON, _ := json.Marshal(variables)
	featuresJSON, _ := json.Marshal(features)
	fieldTogglesJSON, _ := json.Marshal(fieldToggles)

	// Create URL with parameters
	apiURL := BaseURL + endpoint
	params := url.Values{}
	params.Add("variables", string(variablesJSON))
	params.Add("features", string(featuresJSON))
	if fieldToggles != nil {
		params.Add("fieldToggles", string(fieldTogglesJSON))
	}

	fullURL := apiURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set common headers
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://x.com")
	req.Header.Set("Referer", "https://x.com/")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("X-Guest-Token", c.guestToken)
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Client-Language", "en")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %w", err)
	}

	// Check for rate limiting
	if resp.StatusCode == 429 {
		resp.Body.Close()
		return nil, fmt.Errorf("rate limit exceeded. Please wait and try again later")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected response status: %d, body: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// GetUserByScreenName gets user information by screen_name (username)
func (c *Client) GetUserByScreenName(screenName string) (*UserResponse, error) {
	variables := map[string]any{
		"screen_name": screenName,
	}

	features := map[string]any{
		"responsive_web_grok_bio_auto_translation_is_enabled":               false,
		"hidden_profile_subscriptions_enabled":                              true,
		"payments_enabled":                                                  false,
		"profile_label_improvements_pcf_label_in_post_enabled":              true,
		"rweb_tipjar_consumption_enabled":                                   true,
		"verified_phone_label_enabled":                                      false,
		"subscriptions_verification_info_is_identity_verified_enabled":      true,
		"subscriptions_verification_info_verified_since_enabled":            true,
		"highlights_tweets_tab_ui_enabled":                                  true,
		"responsive_web_twitter_article_notes_tab_enabled":                  true,
		"subscriptions_feature_can_gift_premium":                            true,
		"creator_subscriptions_tweet_preview_api_enabled":                   true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled": false,
		"responsive_web_graphql_timeline_navigation_enabled":                true,
	}

	fieldToggles := map[string]any{
		"withAuxiliaryUserLabels": true,
	}

	resp, err := c.makeAPICall(UserByScreenNamePath, variables, features, fieldToggles)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userResp UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Check if user was found
	if userResp.Data.User.Result.RestID == "" {
		return nil, fmt.Errorf("user not found: %s", screenName)
	}

	return &userResp, nil
}

// GetUserID gets user ID by username with caching for frequently requested users
func (c *Client) GetUserID(username string) (string, error) {
	// Normalize username (remove @ if present)
	username = strings.TrimPrefix(username, "@")
	username = strings.ToLower(username)

	// Check cache first
	if value, ok := userIDCache.Load(username); ok {
		entry := value.(*userIDCacheEntry)
		return entry.UserID, nil
	}

	// Try to get user info from API
	userResp, err := c.GetUserByScreenName(username)
	if err != nil {
		return "", fmt.Errorf("failed to get user ID for username '%s': %w", username, err)
	}

	userID := userResp.Data.User.Result.RestID
	if userID == "" {
		return "", fmt.Errorf("user ID not found for username '%s'", username)
	}

	// Cache the result
	userIDCache.Store(username, &userIDCacheEntry{
		UserID:    userID,
		Timestamp: time.Now(),
	})

	return userID, nil
}

// GetUserTweets gets user timeline by user ID and returns a list of tweets
func (c *Client) GetUserTweets(userID string) ([]Tweet, error) {
	variables := map[string]any{
		"userId":                                 userID,
		"count":                                  100,
		"includePromotedContent":                 true,
		"withQuickPromoteEligibilityTweetFields": true,
		"withVoice":                              true,
	}

	features := map[string]any{
		"rweb_video_screen_enabled":                                               false,
		"payments_enabled":                                                        false,
		"profile_label_improvements_pcf_label_in_post_enabled":                    true,
		"rweb_tipjar_consumption_enabled":                                         true,
		"verified_phone_label_enabled":                                            false,
		"creator_subscriptions_tweet_preview_api_enabled":                         true,
		"responsive_web_graphql_timeline_navigation_enabled":                      true,
		"responsive_web_graphql_skip_user_profile_image_extensions_enabled":       false,
		"premium_content_api_read_enabled":                                        false,
		"communities_web_enable_tweet_community_results_fetch":                    true,
		"c9s_tweet_anatomy_moderator_badge_enabled":                               true,
		"responsive_web_grok_analyze_button_fetch_trends_enabled":                 false,
		"responsive_web_grok_analyze_post_followups_enabled":                      false,
		"responsive_web_jetfuel_frame":                                            false,
		"responsive_web_grok_share_attachment_enabled":                            true,
		"articles_preview_enabled":                                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              true,
		"view_counts_everywhere_api_enabled":                                      true,
		"longform_notetweets_consumption_enabled":                                 true,
		"responsive_web_twitter_article_tweet_consumption_enabled":                true,
		"tweet_awards_web_tipping_enabled":                                        false,
		"responsive_web_grok_show_grok_translated_post":                           false,
		"responsive_web_grok_analysis_button_from_backend":                        false,
		"creator_subscriptions_quote_tweet_preview_enabled":                       false,
		"freedom_of_speech_not_reach_fetch_enabled":                               true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": true,
		"longform_notetweets_rich_text_read_enabled":                              true,
		"longform_notetweets_inline_media_enabled":                                true,
		"responsive_web_grok_image_annotation_enabled":                            true,
		"responsive_web_enhance_cards_enabled":                                    false,
	}

	fieldToggles := map[string]any{
		"withArticlePlainText": false,
	}

	resp, err := c.makeAPICall(UserTweetsPath, variables, features, fieldToggles)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var timelineResp TimelineResponse
	if err := json.NewDecoder(resp.Body).Decode(&timelineResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Extract tweets from the timeline response
	tweets := extractTweetsFromTimeline(&timelineResp)
	return tweets, nil
}

// processTweetResult processes a single tweet result by extracting images, setting URL, and generating HTML
func processTweetResult(tweetResult *TweetResult) {
	if tweetResult.Legacy.FullText == "" {
		return
	}

	// Determine tweet type
	tweetResult.IsRetweet = tweetResult.Legacy.RetweetedStatusIDStr != "" || strings.HasPrefix(tweetResult.Legacy.FullText, "RT @") || tweetResult.RetweetedStatusResult.Result != nil
	tweetResult.IsReply = tweetResult.Legacy.InReplyToStatusIDStr != ""
	tweetResult.IsQuoted = tweetResult.Legacy.IsQuoteStatus || tweetResult.Legacy.QuotedStatusIDStr != ""

	// Extract images from tweet media entities
	var images []string
	// First check extended_entities for media (preferred source)
	for _, media := range tweetResult.Legacy.ExtendedEntities.Media {
		if media.Type == "photo" && media.MediaURLHTTPS != "" {
			images = append(images, media.MediaURLHTTPS)
		}
	}
	// If no extended_entities, check regular entities
	if len(images) == 0 {
		for _, media := range tweetResult.Legacy.Entities.Media {
			if media.Type == "photo" && media.MediaURLHTTPS != "" {
				images = append(images, media.MediaURLHTTPS)
			}
		}
	}
	tweetResult.Images = images

	// Set the permanent URL for a tweet
	screenName := tweetResult.Core.UserResults.Result.Core.ScreenName
	if screenName != "" {
		tweetResult.URL = fmt.Sprintf("https://x.com/%s/status/%s", screenName, tweetResult.RestID)
	}

	// Generate HTML content with links and images
	text := html.EscapeString(tweetResult.Legacy.FullText)

	// Replace URLs with HTML links
	for _, url := range tweetResult.Legacy.Entities.Urls {
		expandedURL := url.ExpandedURL
		if expandedURL == "" {
			expandedURL = url.URL
		}
		htmlLink := fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`,
			html.EscapeString(expandedURL),
			html.EscapeString(url.DisplayURL))
		text = strings.ReplaceAll(text, url.URL, htmlLink)
	}

	// Replace hashtags with HTML links
	hashtagRegex := regexp.MustCompile(`#(\w+)`)
	for _, hashtag := range tweetResult.Legacy.Entities.Hashtags {
		hashtagText := "#" + hashtag.Text
		hashtagLink := fmt.Sprintf(`<a href="https://x.com/hashtag/%s" target="_blank">%s</a>`,
			html.EscapeString(hashtag.Text),
			html.EscapeString(hashtagText))
		text = hashtagRegex.ReplaceAllStringFunc(text, func(match string) string {
			if strings.EqualFold(match, hashtagText) {
				return hashtagLink
			}
			return match
		})
	}

	// Replace mentions with HTML links
	mentionRegex := regexp.MustCompile(`@(\w+)`)
	text = mentionRegex.ReplaceAllStringFunc(text, func(match string) string {
		username := strings.TrimPrefix(match, "@")
		return fmt.Sprintf(`<a href="https://x.com/%s" target="_blank">%s</a>`,
			html.EscapeString(username),
			html.EscapeString(match))
	})

	// Add images at the end
	for _, imageURL := range tweetResult.Images {
		text += fmt.Sprintf(`<br><a href="%s" target="_blank"><img src="%s" alt="Tweet image" style="max-width: 500px; height: auto;"></a>`,
			html.EscapeString(imageURL),
			html.EscapeString(imageURL))
	}

	tweetResult.HTML = text
}

// convertTweetResult converts TweetResult to public Tweet structure
func convertTweetResult(tweetResult *TweetResult) Tweet {
	// Store original retweet flag
	originalIsRetweet := tweetResult.IsRetweet

	// Check if this is a retweet and replace with original tweet if available
	if tweetResult.Legacy.RetweetedStatusIDStr != "" || tweetResult.RetweetedStatusResult.Result != nil {
		originalIsRetweet = true
		if tweetResult.RetweetedStatusResult.Result != nil {
			// Process the retweeted status to ensure it has all necessary fields
			processTweetResult(tweetResult.RetweetedStatusResult.Result)
			// Replace the current tweet with the retweeted one
			tweetResult = tweetResult.RetweetedStatusResult.Result
		}
	}

	// Extract hashtags as strings
	var hashtags []string
	for _, hashtag := range tweetResult.Legacy.Entities.Hashtags {
		hashtags = append(hashtags, hashtag.Text)
	}

	// Extract URLs
	var urls []URL
	for _, url := range tweetResult.Legacy.Entities.Urls {
		urls = append(urls, URL{
			Short:    url.URL,
			Expanded: url.ExpandedURL,
			Display:  url.DisplayURL,
		})
	}

	// Extract mentions from text using regex
	var mentions []string
	mentionRegex := regexp.MustCompile(`@(\w+)`)
	matches := mentionRegex.FindAllStringSubmatch(tweetResult.Legacy.FullText, -1)
	for _, match := range matches {
		if len(match) > 1 {
			mentions = append(mentions, match[1])
		}
	}

	return Tweet{
		ID:           tweetResult.RestID,
		Text:         tweetResult.Legacy.FullText,
		HTML:         tweetResult.HTML,
		CreatedAt:    tweetResult.Legacy.CreatedAt,
		PermanentURL: tweetResult.URL,
		Username:     tweetResult.Core.UserResults.Result.Core.ScreenName,
		UserID:       tweetResult.Legacy.UserIDStr,
		Likes:        tweetResult.Legacy.FavoriteCount,
		Retweets:     tweetResult.Legacy.RetweetCount,
		Replies:      tweetResult.Legacy.ReplyCount,
		IsPinned:     tweetResult.IsPinned,
		IsRetweet:    originalIsRetweet,
		IsQuoted:     tweetResult.IsQuoted,
		IsReply:      tweetResult.IsReply,
		Images:       tweetResult.Images,
		Hashtags:     hashtags,
		URLs:         urls,
		Mentions:     mentions,
	}
}

// extractTweetsFromTimeline extracts tweets from timeline response
func extractTweetsFromTimeline(timeline *TimelineResponse) []Tweet {
	var tweetResults []TweetResult

	for _, instruction := range timeline.Data.User.Result.Timeline.Timeline.Instructions {
		if instruction.Type == "TimelineAddEntries" {
			for _, entry := range instruction.Entries {
				// Process regular tweets
				if strings.Contains(entry.EntryID, "tweet-") && entry.Content.ItemContent != nil {
					tweetResult := entry.Content.ItemContent.TweetResults.Result
					processTweetResult(&tweetResult)
					if tweetResult.Legacy.FullText != "" {
						tweetResults = append(tweetResults, tweetResult)
					}
				}

				// Process profile-conversation entries
				if strings.Contains(entry.EntryID, "profile-conversation-") &&
					entry.Content.EntryType == "TimelineTimelineModule" &&
					entry.Content.Items != nil {

					for _, item := range *entry.Content.Items {
						if strings.Contains(item.EntryID, "tweet-") {
							tweetResult := item.Item.ItemContent.TweetResults.Result
							processTweetResult(&tweetResult)
							if tweetResult.Legacy.FullText != "" {
								tweetResults = append(tweetResults, tweetResult)
							}
						}
					}
				}
			}
		} else if instruction.Type == "TimelinePinEntry" && instruction.Entry != nil {
			if strings.Contains(instruction.Entry.EntryID, "tweet-") && instruction.Entry.Content.ItemContent != nil {
				tweetResult := instruction.Entry.Content.ItemContent.TweetResults.Result
				tweetResult.IsPinned = true
				processTweetResult(&tweetResult)
				if tweetResult.Legacy.FullText != "" {
					tweetResults = append(tweetResults, tweetResult)
				}
			}
		}
	}

	// Convert TweetResults to public Tweet structures
	var tweets []Tweet
	for _, tweetResult := range tweetResults {
		tweets = append(tweets, convertTweetResult(&tweetResult))
	}

	return tweets
}
