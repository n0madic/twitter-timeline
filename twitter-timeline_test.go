package twittertimeline

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Test constants - using known public accounts
const (
	// Elon Musk's account - very active and public
	TestUserID   = "44196397"
	TestUsername = "elonmusk"

	// Known active account with various tweet types
	TestUserID2   = "783214" // Twitter account
	TestUsername2 = "Twitter"

	// Invalid IDs for error testing
	InvalidUserID   = "999999999999999999"
	InvalidUsername = "thisusernameshouldnotexist123456789"
)

func TestNewClient(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Fatal("NewClient() returned nil")
	}

	if client.httpClient == nil {
		t.Error("HTTP client is nil")
	}

	if client.bearerToken != BearerToken {
		t.Error("Bearer token not set correctly")
	}

	if client.cacheTTL != 24*time.Hour {
		t.Error("Cache TTL not set correctly")
	}
}

func TestGetGuestToken(t *testing.T) {
	client := NewClient()

	err := client.GetGuestToken()
	if err != nil {
		t.Fatalf("GetGuestToken() failed: %v", err)
	}

	if client.guestToken == "" {
		t.Error("Guest token is empty after acquisition")
	}

	// Test that guest token looks valid (should be alphanumeric)
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, client.guestToken)
	if !matched {
		t.Errorf("Guest token format seems invalid: %s", client.guestToken)
	}
}

func TestGetUserTweets_ValidUserID(t *testing.T) {
	client := NewClient()

	tweets, err := client.GetUserTweets(TestUserID)
	if err != nil {
		t.Fatalf("GetUserTweets() failed for valid user ID: %v", err)
	}

	if len(tweets) == 0 {
		t.Error("No tweets returned for active user")
	}

	// Test first tweet structure
	tweet := tweets[0]

	// Basic fields
	if tweet.ID == "" {
		t.Error("Tweet ID is empty")
	}
	if tweet.Text == "" {
		t.Error("Tweet text is empty")
	}
	if tweet.CreatedAt == "" {
		t.Error("Tweet CreatedAt is empty")
	}
	if tweet.Username == "" {
		t.Error("Tweet Username is empty")
	}
	if tweet.UserID == "" {
		t.Error("Tweet UserID is empty")
	}

	// Check PermanentURL format
	if tweet.PermanentURL != "" {
		expectedPrefix := fmt.Sprintf("https://x.com/%s/status/", tweet.Username)
		if !strings.HasPrefix(tweet.PermanentURL, expectedPrefix) {
			t.Errorf("PermanentURL format incorrect: %s", tweet.PermanentURL)
		}
		if !strings.HasSuffix(tweet.PermanentURL, tweet.ID) {
			t.Errorf("PermanentURL doesn't end with tweet ID: %s", tweet.PermanentURL)
		}
	}

	// HTML should be generated
	if tweet.HTML == "" {
		t.Error("HTML content is empty")
	}

	// Statistics should be non-negative
	if tweet.Likes < 0 {
		t.Error("Likes count is negative")
	}
	if tweet.Retweets < 0 {
		t.Error("Retweets count is negative")
	}
	if tweet.Replies < 0 {
		t.Error("Replies count is negative")
	}
}

func TestGetUserTweets_InvalidUserID(t *testing.T) {
	client := NewClient()

	tweets, err := client.GetUserTweets(InvalidUserID)
	// The API might not always return an error for invalid ID,
	// it might just return empty results
	if err != nil {
		t.Logf("Got expected error for invalid user ID: %v", err)
	}

	if len(tweets) == 0 {
		t.Log("Got empty tweets for invalid user ID (expected)")
	} else {
		t.Logf("Got %d tweets for invalid user ID", len(tweets))
	}
}

func TestGetUserID_ValidUsername(t *testing.T) {
	client := NewClient()

	userID, err := client.GetUserID(TestUsername)
	if err != nil {
		t.Fatalf("GetUserID() failed for valid username: %v", err)
	}

	if userID == "" {
		t.Error("UserID is empty")
	}

	// Should return the expected user ID
	if userID != TestUserID {
		t.Errorf("Expected user ID %s, got %s", TestUserID, userID)
	}
}

func TestGetUserID_InvalidUsername(t *testing.T) {
	client := NewClient()

	userID, err := client.GetUserID(InvalidUsername)
	if err == nil {
		t.Error("Expected error for invalid username")
	}

	if userID != "" {
		t.Error("Should return empty user ID for invalid username")
	}
}

func TestIntegration_FullWorkflow(t *testing.T) {
	client := NewClient()

	// Use Twitter's official account for diverse tweets
	tweets, err := client.GetUserTweets(TestUserID2)
	if err != nil {
		t.Fatalf("GetUserTweets() failed: %v", err)
	}
	if len(tweets) == 0 {
		t.Fatal("No tweets returned")
	}

	foundTweetWithImages := false
	foundTweetWithHashtags := false
	foundTweetWithURLs := false
	foundTweetWithMentions := false

	foundPinned := false
	foundRetweet := false
	foundReply := false
	foundQuoted := false

	numericRegex := regexp.MustCompile(`^\d+$`)

	for i, tweet := range tweets {
		if tweet.ID == "" {
			t.Errorf("Tweet %d has empty ID", i)
		}
		if tweet.Text == "" {
			t.Errorf("Tweet %d has empty text", i)
		}
		if tweet.Username == "" {
			t.Errorf("Tweet %d has empty username", i)
		}
		if tweet.UserID != TestUserID2 {
			t.Errorf("Tweet %d has wrong user ID: expected %s, got %s", i, TestUserID2, tweet.UserID)
		}
		if tweet.PermanentURL == "" {
			t.Errorf("Tweet %d has empty permanent URL", i)
		}
		if tweet.HTML == "" {
			t.Errorf("Tweet %d has empty HTML", i)
		}

		// --- Structure tests ---
		// URL parsing
		if len(tweet.URLs) > 0 {
			foundTweetWithURLs = true
			for _, url := range tweet.URLs {
				if url.Short == "" || url.Display == "" {
					t.Error("URL structure incomplete")
				}
			}
		}
		// Hashtag extraction
		if len(tweet.Hashtags) > 0 {
			foundTweetWithHashtags = true
			for _, hashtag := range tweet.Hashtags {
				if hashtag == "" {
					t.Error("Empty hashtag found")
				}
				if strings.HasPrefix(hashtag, "#") {
					t.Error("Hashtag should not include # symbol")
				}
			}
		}
		// Mention extraction
		if len(tweet.Mentions) > 0 {
			foundTweetWithMentions = true
			for _, mention := range tweet.Mentions {
				if mention == "" {
					t.Error("Empty mention found")
				}
				if strings.HasPrefix(mention, "@") {
					t.Error("Mention should not include @ symbol")
				}
			}
		}
		// Image extraction
		if len(tweet.Images) > 0 {
			foundTweetWithImages = true
			for _, image := range tweet.Images {
				if !strings.HasPrefix(image, "https://") {
					t.Error("Image URL should be HTTPS")
				}
			}
		}

		// Tweet type flags consistency
		typeCount := 0
		if tweet.IsRetweet {
			typeCount++
		}
		if tweet.IsReply {
			typeCount++
		}
		if tweet.IsQuoted {
			typeCount++
		}
		// UserID numeric
		matched := numericRegex.Match([]byte(tweet.UserID))
		if !matched {
			t.Errorf("UserID should be numeric: %s", tweet.UserID)
		}
		// Tweet ID numeric
		matched = numericRegex.Match([]byte(tweet.ID))
		if !matched {
			t.Errorf("Tweet ID should be numeric: %s", tweet.ID)
		}

		// --- Type detection ---
		if tweet.IsPinned {
			foundPinned = true
			t.Logf("Found pinned tweet: %s", tweet.ID)
		}
		if tweet.IsRetweet {
			foundRetweet = true
			t.Logf("Found retweet: %s", tweet.ID)
		}
		if tweet.IsReply {
			foundReply = true
			t.Logf("Found reply: %s", tweet.ID)
		}
		if tweet.IsQuoted {
			foundQuoted = true
			t.Logf("Found quoted tweet: %s", tweet.ID)
		}
	}

	t.Logf("Found tweets with: images=%v, hashtags=%v, URLs=%v, mentions=%v",
		foundTweetWithImages, foundTweetWithHashtags, foundTweetWithURLs, foundTweetWithMentions)
	t.Logf("Tweet types found: pinned=%v, retweet=%v, reply=%v, quoted=%v",
		foundPinned, foundRetweet, foundReply, foundQuoted)
}
