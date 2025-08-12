package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	twittertimeline "github.com/n0madic/twitter-timeline"
)

func main() {
	// Define command line flags
	proxy := flag.String("proxy", "", "HTTP/HTTPS proxy URL (e.g., http://proxy:8080 or https://user:pass@proxy:8080)")
	help := flag.Bool("help", false, "Show help message")
	h := flag.Bool("h", false, "Show help message")

	// Parse flags
	flag.Parse()

	// Show help if requested or no arguments
	if *help || *h || flag.NArg() < 1 {
		fmt.Println("Usage: twitter-timeline [OPTIONS] <user_id_or_username>")
		fmt.Println("\nOptions:")
		fmt.Println("  -proxy string    HTTP/HTTPS proxy URL (e.g., http://proxy:8080)")
		fmt.Println("  -help, -h        Show this help message")
		fmt.Println("\nExamples:")
		fmt.Println("  twitter-timeline 1624051836033421317                          # Poe platform (User ID)")
		fmt.Println("  twitter-timeline elonmusk                                     # Elon Musk (Username)")
		fmt.Println("  twitter-timeline -proxy http://proxy:8080 elonmusk            # With proxy")
		fmt.Println("  twitter-timeline -proxy https://user:pass@proxy:8080 elonmusk # With authenticated proxy")
		fmt.Println("\nProxy can also be set via environment variables:")
		fmt.Println("  HTTP_PROXY=http://proxy:8080 twitter-timeline elonmusk")
		fmt.Println("  HTTPS_PROXY=https://proxy:8080 twitter-timeline elonmusk")
		os.Exit(0)
	}

	userID := flag.Arg(0)
	client := twittertimeline.NewClient()

	// Configure proxy if provided
	if *proxy != "" {
		fmt.Printf("Using proxy: %s\n", *proxy)
		if err := client.SetProxy(*proxy); err != nil {
			fmt.Printf("Error setting proxy: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Check environment variables for proxy
		envProxy := os.Getenv("HTTP_PROXY")
		if envProxy == "" {
			envProxy = os.Getenv("HTTPS_PROXY")
		}
		if envProxy == "" {
			envProxy = os.Getenv("http_proxy")
		}
		if envProxy == "" {
			envProxy = os.Getenv("https_proxy")
		}

		if envProxy != "" {
			fmt.Printf("Using proxy from environment: %s\n", envProxy)
			if err := client.SetProxy(envProxy); err != nil {
				fmt.Printf("Error setting proxy from environment: %v\n", err)
				os.Exit(1)
			}
		}
	}

	// Resolve User ID from input parameter
	IsUserID, _ := regexp.MatchString(`^\d{1,19}$`, userID)
	if !IsUserID {
		// Otherwise consider it username and try to get User ID
		resolvedUserID, err := client.GetUserID(userID)
		if err != nil {
			fmt.Printf("failed to find user '%s': %v\n", userID, err)
			os.Exit(1)
		}
		userID = resolvedUserID
	}

	fmt.Printf("Loading timeline for user %s...\n", userID)

	tweets, err := client.GetUserTweets(userID)
	if err != nil {
		fmt.Printf("Error getting timeline: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== TIMELINE ===")

	for _, tweet := range tweets {
		if tweet.Text != "" {
			if tweet.IsPinned {
				fmt.Println("\n🔝 PINNED TWEET:")
			}

			// Show tweet type
			var tweetType []string
			if tweet.IsRetweet {
				tweetType = append(tweetType, "RETWEET")
			}
			if tweet.IsReply {
				tweetType = append(tweetType, "REPLY")
			}
			if tweet.IsQuoted {
				tweetType = append(tweetType, "QUOTED")
			}
			if len(tweetType) == 0 {
				tweetType = append(tweetType, "ORIGINAL")
			}

			fmt.Printf("\n--- Tweet ID: %s [%s] ---\n", tweet.ID, strings.Join(tweetType, ", "))
			fmt.Printf("Author: @%s (ID: %s)\n", tweet.Username, tweet.UserID)
			fmt.Printf("Text: %s\n", tweet.Text)
			fmt.Printf("Created: %s\n", tweet.CreatedAt)
			fmt.Printf("Stats: %d likes | %d retweets | %d replies\n",
				tweet.Likes,
				tweet.Retweets,
				tweet.Replies)
			if tweet.PermanentURL != "" {
				fmt.Printf("URL: %s\n", tweet.PermanentURL)
			}
			if tweet.HTML != "" {
				fmt.Printf("HTML: %s\n", tweet.HTML)
			}

			if len(tweet.Hashtags) > 0 {
				fmt.Print("Hashtags: ")
				for _, hashtag := range tweet.Hashtags {
					fmt.Printf("#%s ", hashtag)
				}
				fmt.Println()
			}

			if len(tweet.URLs) > 0 {
				fmt.Println("URLs:")
				for _, url := range tweet.URLs {
					fmt.Printf("  %s (%s) -> %s\n", url.Display, url.Short, url.Expanded)
				}
			}

			if len(tweet.Mentions) > 0 {
				fmt.Print("Mentions: ")
				for _, mention := range tweet.Mentions {
					fmt.Printf("@%s ", mention)
				}
				fmt.Println()
			}

			if len(tweet.Images) > 0 {
				fmt.Println("Images:")
				for _, imageURL := range tweet.Images {
					fmt.Printf("  %s\n", imageURL)
				}
			}
		}
	}
}
