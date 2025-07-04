# Twitter Timeline Library

Go library for loading Twitter/X user timeline without authorization.

## 🚀 Features

- Get public timeline of any Twitter/X user
- Works **without authorization** (automatic guest token acquisition)
- **User ID support** - works with numeric Twitter User IDs
- **Username support** - automatic resolution and caching
- **Rich content processing** - HTML generation with clickable links
- **Complete metadata** - pinned tweets, types (retweet/reply/quote), statistics
- **Media extraction** - automatic image URL extraction
- **Entity parsing** - hashtags, URLs, mentions automatically extracted
- **Library and CLI interface** with simple, intuitive usage
- **Error handling** and rate limit detection

## 📦 Installation

### As a library
```bash
go get github.com/n0madic/twitter-timeline
```

### CLI tool
```bash
git clone https://github.com/n0madic/twitter-timeline
cd twitter-timeline
go build -o twitter-timeline cmd/twitter-timeline/main.go
```

## 📖 Usage

### Library Usage

```go
package main

import (
    "fmt"
    "log"

    twittertimeline "github.com/n0madic/twitter-timeline"
)

func main() {
    // Create a new Twitter client
    client := twittertimeline.NewClient()

    // Option 1: Get user timeline by User ID
    tweets, err := client.GetUserTweets("1624051836033421317")
    if err != nil {
        log.Fatalf("Error getting timeline: %v", err)
    }

    // Option 2: Get user ID from username first (with caching)
    userID, err := client.GetUserID("elonmusk")
    if err != nil {
        log.Fatalf("Error getting user ID: %v", err)
    }

    tweets, err = client.GetUserTweets(userID)
    if err != nil {
        log.Fatalf("Error getting timeline: %v", err)
    }

    // Process tweets - simple flat structure!
    for _, tweet := range tweets {
        fmt.Printf("@%s: %s\n", tweet.Username, tweet.Text)
        fmt.Printf("Stats: %d likes, %d retweets, %d replies\n",
            tweet.Likes, tweet.Retweets, tweet.Replies)
        fmt.Printf("URL: %s\n", tweet.PermanentURL)

        // Tweet type indicators
        if tweet.IsPinned {
            fmt.Println("📌 Pinned")
        }
        if tweet.IsRetweet {
            fmt.Println("🔄 Retweet")
        }
        if tweet.IsReply {
            fmt.Println("💬 Reply")
        }

        // Rich content
        if len(tweet.Images) > 0 {
            fmt.Printf("Images: %v\n", tweet.Images)
        }
        if len(tweet.Hashtags) > 0 {
            fmt.Printf("Hashtags: %v\n", tweet.Hashtags)
        }
        if len(tweet.Mentions) > 0 {
            fmt.Printf("Mentions: %v\n", tweet.Mentions)
        }

        fmt.Println("---")
    }
}
```

### CLI Usage

```bash
./twitter-timeline <user_id_or_username>
```

#### Parameters

- `user_id_or_username` - Twitter user ID (numeric) or username (@handle without @)

#### Examples

```bash
# Load tweets from Elon Musk
./twitter-timeline 44196397

# Load tweets using username
./twitter-timeline elonmusk
```

## 🔍 How to find User ID

To use the library or CLI, you can use either the numeric User ID or username of the Twitter account.

### Getting User ID from username:
```go
client := twittertimeline.NewClient()

// Using GetUserID (returns user ID with caching)
userID, err := client.GetUserID("elonmusk")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("User ID: %s\n", userID)

// The library automatically caches user IDs for 24 hours
// to improve performance on subsequent requests
```

### Alternative methods to find User IDs:
- Twitter's web interface (inspect profile elements)
- Third-party services like [tweeterid.com](https://tweeterid.com)

## 📊 Tweet Structure

The library returns a clean, flat `Tweet` structure with all data easily accessible:

```go
type Tweet struct {
    // Basic Information
    ID           string   // Tweet ID
    Text         string   // Tweet text content
    HTML         string   // HTML formatted content with clickable links
    CreatedAt    string   // Creation timestamp
    PermanentURL string   // Direct link to tweet (https://x.com/user/status/id)

    // Author Information
    Username     string   // @username
    UserID       string   // User ID

    // Statistics
    Likes        int      // Like count
    Retweets     int      // Retweet count
    Replies      int      // Reply count

    // Tweet Types
    IsPinned     bool     // Pinned to profile
    IsRetweet    bool     // Is a retweet
    IsQuoted     bool     // Is a quote tweet
    IsReply      bool     // Is a reply

    // Rich Content
    Images       []string // Image URLs
    Hashtags     []string // Hashtag texts (without #)
    URLs         []URL    // Expanded URL information
    Mentions     []string // Mentioned usernames (without @)
}
```

### Key Benefits:
- **No nested structures** - direct field access like `tweet.Text` instead of `tweet.Legacy.FullText`
- **Rich HTML content** - automatically generated with clickable links for URLs, hashtags, mentions
- **Complete metadata** - all useful information extracted and easily accessible
- **Type detection** - automatic identification of retweets, replies, quotes, pinned tweets

## ⚙️ How it works

1. **Guest Token Acquisition**: Automatically requests guest token from Twitter API
2. **GraphQL Request**: Creates properly formatted request with required parameters and headers
3. **JSON Parsing**: Parses complex Twitter API response into internal TweetResult structures
4. **Content Processing**: Extracts images, generates permanent URLs, creates HTML with clickable links
5. **Structure Conversion**: Converts internal TweetResult to clean, flat Tweet structures
6. **Result Delivery**: Returns simple `[]Tweet` array with all useful data easily accessible

## ⚠️ Limitations

- Works only with **public accounts**
- Guest token has **rate limiting**
- **Username resolution** may have occasional issues (use User ID when possible)
- Twitter may **change API** without notice
- Package may require updates when API changes

## 🔧 Technical details

### API Endpoints
- **UserTweets**: `https://api.x.com/graphql/***/UserTweets`
- **UserByScreenName**: `https://api.x.com/graphql/***/UserByScreenName`
- **Guest Token**: `https://api.x.com/1.1/guest/activate.json`

### Headers
- Browser simulation (User-Agent)
- Authorization via Bearer token
- Guest token for unauthenticated access

### Error handling
- HTTP timeout (30 seconds)
- JSON response validation
- Status code checking

## 🛠️ Requirements

- **Go 1.18** or higher
- **Internet connection**
- **Public access** to target account

## 📝 License

MIT License - see LICENSE file for details.

This project is intended for educational purposes and public data analysis. Use responsibly and in accordance with Twitter/X terms of service.
