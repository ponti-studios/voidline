package social

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"voidline/internal/domain/social"
	"voidline/internal/infrastructure/persistence/sqlite"
)

type Service struct {
	repo *sqlite.SocialRepository
}

func NewService(repo *sqlite.SocialRepository) *Service {
	return &Service{repo: repo}
}

type ImportOptions struct {
	DryRun bool
	Force  bool
}

func (s *Service) ImportAll(ctx context.Context, sourceDir string, options ImportOptions) (*social.ImportResult, error) {
	result := &social.ImportResult{}

	if !options.Force {
		fmt.Println("Clearing existing social data...")
		if err := s.repo.ClearTables(ctx); err != nil {
			return result, fmt.Errorf("failed to clear tables: %w", err)
		}
	}

	start := time.Now()

	s.importInstagram(ctx, sourceDir)
	s.importFacebook(ctx, sourceDir)
	s.importBumble(ctx, sourceDir)
	s.importHinge(ctx, sourceDir)
	s.importOpenAI(ctx, sourceDir)

	result.Duration = time.Since(start)
	fmt.Println("Social data import complete.")
	return result, nil
}

func (s *Service) importInstagram(ctx context.Context, sourceDir string) {
	fmt.Println("Importing Instagram data...")

	mediaPath := sourceDir + "/instagram/media.json"
	if fileData, err := os.ReadFile(mediaPath); err == nil {
		var data map[string]interface{}
		if json.Unmarshal(fileData, &data) == nil {
			for _, postType := range []string{"photos", "videos", "stories", "profile", "direct"} {
				if items, ok := data[postType].([]interface{}); ok {
					for _, item := range items {
						m := item.(map[string]interface{})
						caption := toString(m["caption"])
						location := toString(m["location"])
						timestamp := toString(m["taken_at"])
						path := toString(m["path"])
						pt := strings.TrimSuffix(postType, "s")
						s.repo.InsertPost(ctx, "instagram", pt, caption, location, timestamp, path, "", "")
					}
				}
			}
		}
	}

	likesPath := sourceDir + "/instagram/likes.json"
	if fileData, err := os.ReadFile(likesPath); err == nil {
		var data map[string]interface{}
		if json.Unmarshal(fileData, &data) == nil {
			for _, likeType := range []string{"media_likes", "comment_likes"} {
				if items, ok := data[likeType].([]interface{}); ok {
					for _, item := range items {
						arr := item.([]interface{})
						if len(arr) >= 2 {
							timestamp := toString(arr[0])
							username := toString(arr[1])
							target := "media"
							if likeType == "comment_likes" {
								target = "comment"
							}
							s.repo.InsertLike(ctx, "instagram", timestamp, username, target, "LIKE")
						}
					}
				}
			}
		}
	}

	connPath := sourceDir + "/instagram/connections.json"
	if fileData, err := os.ReadFile(connPath); err == nil {
		var data map[string]interface{}
		if json.Unmarshal(fileData, &data) == nil {
			for _, connType := range []string{"followers", "following", "blocked_users", "follow_requests_sent"} {
				if items, ok := data[connType].(map[string]interface{}); ok {
					for username, timestamp := range items {
						cType := strings.TrimSuffix(connType, "s")
						cType = strings.ReplaceAll(cType, "_user", "")
						if cType == "follow_requests_sent" {
							cType = "follow_request_sent"
						}
						s.repo.InsertConnection(ctx, "instagram", cType, username, toString(timestamp))
					}
				}
			}
		}
	}

	commentsPath := sourceDir + "/instagram/comments.json"
	if fileData, err := os.ReadFile(commentsPath); err == nil {
		var data map[string]interface{}
		if json.Unmarshal(fileData, &data) == nil {
			if items, ok := data["media_comments"].([]interface{}); ok {
				for _, item := range items {
					arr := item.([]interface{})
					if len(arr) >= 3 {
						timestamp := toString(arr[0])
						text := toString(arr[1])
						username := toString(arr[2])
						s.repo.InsertComment(ctx, "instagram", timestamp, username, text)
					}
				}
			}
		}
	}

	messagesPath := sourceDir + "/instagram/messages.json"
	if fileData, err := os.ReadFile(messagesPath); err == nil {
		var conversations []interface{}
		if json.Unmarshal(fileData, &conversations) == nil {
			for _, convo := range conversations {
				c := convo.(map[string]interface{})
				participants := c["participants"].([]interface{})
				var participantsStr []string
				for _, p := range participants {
					participantsStr = append(participantsStr, toString(p))
				}
				if msgs, ok := c["conversation"].([]interface{}); ok {
					for _, msg := range msgs {
						m := msg.(map[string]interface{})
						sender := toString(m["sender"])
						var receiver string
						for _, p := range participantsStr {
							if p != sender {
								receiver = p
								break
							}
						}
						timestamp := toString(m["created_at"])
						text := toString(m["text"])
						mediaURL := toString(m["media"])
						if mediaURL == "" {
							mediaURL = toString(m["media_url"])
						}
						storyShare := toString(m["story_share"])
						meta := make(map[string]interface{})
						for k, v := range m {
							if k != "sender" && k != "created_at" && k != "text" && k != "media" && k != "media_url" && k != "story_share" {
								meta[k] = v
							}
						}
						metaJSON, _ := json.Marshal(meta)
						s.repo.InsertMessage(ctx, "instagram", timestamp, sender, receiver, text, mediaURL, storyShare, string(metaJSON))
					}
				}
			}
		}
	}
}

func (s *Service) importFacebook(ctx context.Context, sourceDir string) {
	fmt.Println("Importing Facebook data...")

	fbPostsPath := sourceDir + "/facebook/2020-07-12/posts/your_posts_1.json"
	if fileData, err := os.ReadFile(fbPostsPath); err == nil {
		var posts []interface{}
		if json.Unmarshal(fileData, &posts) == nil {
			for _, post := range posts {
				p := post.(map[string]interface{})
				ts := toString(p["timestamp"])
				title := toString(p["title"])
				dataList := p["data"].([]interface{})
				postText := ""
				for _, d := range dataList {
					if dm, ok := d.(map[string]interface{}); ok {
						if post, ok := dm["post"]; ok {
							postText = toString(post)
							break
						}
					}
				}
				caption := strings.TrimSpace(title + "\n" + postText)
				s.repo.InsertPost(ctx, "facebook", "post", caption, "", ts, "", "", "")
			}
		}
	}

	fbLikesPath := sourceDir + "/facebook/2020-07-12/likes_and_reactions/posts_and_comments.json"
	if fileData, err := os.ReadFile(fbLikesPath); err == nil {
		var data map[string]interface{}
		if json.Unmarshal(fileData, &data) == nil {
			if reactions, ok := data["reactions"].([]interface{}); ok {
				for _, reaction := range reactions {
					r := reaction.(map[string]interface{})
					ts := toString(r["timestamp"])
					title := toString(r["title"])
					reactionType := ""
					if dataArr, ok := r["data"].([]interface{}); ok && len(dataArr) > 0 {
						if reactionObj, ok := dataArr[0].(map[string]interface{}); ok {
							if reactionData, ok := reactionObj["reaction"].(map[string]interface{}); ok {
								reactionType = toString(reactionData["reaction"])
							}
						}
					}
					s.repo.InsertLike(ctx, "facebook", ts, title, "reaction", reactionType)
				}
			}
		}
	}

	fbFriendsPath := sourceDir + "/facebook/2020-07-12/friends/friends.json"
	if fileData, err := os.ReadFile(fbFriendsPath); err == nil {
		var data map[string]interface{}
		if json.Unmarshal(fileData, &data) == nil {
			if friends, ok := data["friends"].([]interface{}); ok {
				for _, friend := range friends {
					f := friend.(map[string]interface{})
					name := toString(f["name"])
					ts := toString(f["timestamp"])
					s.repo.InsertConnection(ctx, "facebook", "friend", name, ts)
				}
			}
		}
	}
}

func (s *Service) importBumble(ctx context.Context, sourceDir string) {
	fmt.Println("Importing Bumble data...")

	matches, _ := filepath.Glob(sourceDir + "/bumble/**/messages_of_*.txt")
	msgRegex := regexp.MustCompile(`(Me|User)\s*\(([\d\-\s:]+)\):?\s*(.*)`)

	for _, filePath := range matches {
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		currentChat := "Unknown"

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.Contains(line, "Chat with User") {
				currentChat = "User"
				continue
			}

			match := msgRegex.FindStringSubmatch(line)
			if len(match) >= 4 {
				senderLabel := match[1]
				timestamp := match[2]
				text := strings.TrimSpace(match[3])

				sender := "Me"
				receiver := currentChat
				if senderLabel != "Me" {
					sender = currentChat
					receiver = "Me"
				}

				s.repo.InsertMessage(ctx, "bumble", timestamp, sender, receiver, text, "", "", "")
			}
		}
	}
}

func (s *Service) importHinge(ctx context.Context, sourceDir string) {
	fmt.Println("Importing Hinge data...")

	userPath := sourceDir + "/hinge/user.json"
	if fileData, err := os.ReadFile(userPath); err == nil {
		var data map[string]interface{}
		if json.Unmarshal(fileData, &data) == nil {
			if profile, ok := data["profile"].(map[string]interface{}); ok {
				profileJSON, _ := json.Marshal(profile)
				s.repo.InsertPost(ctx, "hinge", "profile", "Profile Information", "", "", "", "", string(profileJSON))
			}
		}
	}

	promptsPath := sourceDir + "/hinge/prompts.json"
	if fileData, err := os.ReadFile(promptsPath); err == nil {
		var prompts []interface{}
		if json.Unmarshal(fileData, &prompts) == nil {
			for _, p := range prompts {
				pm := p.(map[string]interface{})
				caption := toString(pm["prompt"]) + ": " + toString(pm["text"])
				ts := toString(pm["created"])
				pJSON, _ := json.Marshal(pm)
				s.repo.InsertPost(ctx, "hinge", "prompt", caption, "", ts, "", "", string(pJSON))
			}
		}
	}

	mediaPath := sourceDir + "/hinge/media.json"
	if fileData, err := os.ReadFile(mediaPath); err == nil {
		var media []interface{}
		if json.Unmarshal(fileData, &media) == nil {
			for _, m := range media {
				mi := m.(map[string]interface{})
				mType := toString(mi["type"])
				if mType == "" {
					mType = "photo"
				}
				url := toString(mi["url"])
				mJSON, _ := json.Marshal(mi)
				s.repo.InsertPost(ctx, "hinge", mType, "", "", "", url, "", string(mJSON))
			}
		}
	}
}

func (s *Service) importOpenAI(ctx context.Context, sourceDir string) {
	fmt.Println("Importing OpenAI data...")

	convosPath := sourceDir + "/openai/conversations.json"
	if fileData, err := os.ReadFile(convosPath); err == nil {
		var convos []interface{}
		if json.Unmarshal(fileData, &convos) == nil {
			for _, convo := range convos {
				c := convo.(map[string]interface{})
				title := toString(c["title"])
				if mapping, ok := c["mapping"].(map[string]interface{}); ok {
					for _, node := range mapping {
						if msg, ok := node.(map[string]interface{})["message"].(map[string]interface{}); ok {
							authorMap, _ := msg["author"].(map[string]interface{})
							author := toString(authorMap["role"])
							if author == "" {
								author = "unknown"
							}
							text := ""
							if content, ok := msg["content"].(map[string]interface{}); ok {
								if contentType, ok := content["content_type"].(string); ok && contentType == "text" {
									if parts, ok := content["parts"].([]interface{}); ok {
										var partsStr []string
										for _, p := range parts {
											partsStr = append(partsStr, toString(p))
										}
										text = strings.Join(partsStr, "\n")
									}
								}
							}
							ts := toString(msg["create_time"])
							receiver := "User"
							if author == "user" {
								receiver = "ChatGPT"
							}
							meta := map[string]string{"convo_title": title}
							metaJSON, _ := json.Marshal(meta)
							s.repo.InsertMessage(ctx, "openai", ts, author, receiver, text, "", "", string(metaJSON))
						}
					}
				}
			}
		}
	}
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%f", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}
