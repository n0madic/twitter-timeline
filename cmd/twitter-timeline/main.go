package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	twittertimeline "github.com/n0madic/twitter-timeline"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: twitter-timeline <user_id_or_username>")
		fmt.Println("Examples:")
		fmt.Println("  twitter-timeline 1624051836033421317     # Poe platform (User ID)")
		fmt.Println("  twitter-timeline elonmusk                # Elon Musk (Username)")
		os.Exit(1)
	}

	userID := os.Args[1]
	client := twittertimeline.NewClient()

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
